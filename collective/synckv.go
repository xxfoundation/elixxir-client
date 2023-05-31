////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"bytes"
	"io"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

const syncStoppable = "syncStoppable"

const StandardRemoteSyncPrefix = "remoteSync"

type SyncKV interface {
	versioned.KV
	StartProcesses() (stoppable.Stoppable, error)
	RegisterConnectionTracker(nc NotifyCallback)
	IsConnected() bool
	IsSynched() bool
	WaitForRemote(timeout time.Duration) bool
}

// versionedKV wraps a [collective.KV] inside of a [storage.versioned.KV] interface.
type versionedKV struct {
	// synchronizedPrefixes are prefixes that trigger remote
	// synchronization calls.
	synchronizedPrefixes []string

	// hasSynchronizedPrefix tells us we are in a prefix that is synchronized.
	inSynchronizedPrefix bool

	// remote is the remote synching KV instance. This is used
	// when we intercept Set calls because we are synchronizing this prefix.
	remote *internalKV
	// local is a versioned KV instance that wraps the remoteKV, used
	// for all local operations.
	local versioned.KV
}

// CloneFromRemoteStorage copies state from RemoteStore and
// instantiates a SynchronizedKV
func CloneFromRemoteStorage(path string, deviceSecret []byte,
	remote RemoteStore, kv ekv.KeyValue,
	rng *fastRNG.StreamGenerator) error {

	rkv, err := SynchronizedKV(path, deviceSecret, remote, kv, nil, rng)
	if err != nil {
		return err
	}

	return rkv.remote.col.collect()
}

// SynchronizedKV loads or creates a synchronized remote KV that uses
// a remote RemoteStore to store defined synchronization prefixes to the
// network.
func SynchronizedKV(path string, deviceSecret []byte,
	remote RemoteStore, kv ekv.KeyValue, synchedPrefixes []string,
	rng *fastRNG.StreamGenerator) (*versionedKV, error) {

	rngStream := rng.GetStream()
	defer rngStream.Close()
	deviceID, err := getOrInitDeviceID(kv, rngStream)
	if err != nil {
		return nil, err
	}

	if !isRemoteKV(kv) {
		jww.INFO.Printf("Converting KV to a remote KV: %s", deviceID)
		enableRemoteKV(kv)
	}

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rng,
	}

	txLog, err := newRemoteWriter(path, deviceID, remote,
		crypt, kv)
	if err != nil {
		return nil, err
	}

	vkv := newVersionedKV(txLog, kv, synchedPrefixes)

	// FIXME: this is a bit circular. Both remoteWriter/txLog and
	// collector have goroutines that need to be started/stopped,
	// and we're doing this by connecting them to the internalKV
	// to initiate those via StartProcessies. The internalKV by
	// itself is just writing to a txLog interface and shouldn't
	// care about that. It also should't even know that the
	// collector exists, it just exposes endpoints that it uses to
	// do it's job. We'll leave it for now but the way this gets
	// instantiated needs a rework.
	vkv.remote.col = newCollector(deviceID, path, remote, vkv.remote,
		crypt, txLog)

	return vkv, nil
}

// LocalKV Loads or Creates a synchronized remote KV that uses a local-only
// mutate log. It panics if the underlying KV has ever been used
// for remote operations in the past.
func LocalKV(path string, deviceSecret []byte, kv ekv.KeyValue,
	rng *fastRNG.StreamGenerator) (SyncKV, error) {

	if isRemoteKV(kv) {
		jww.FATAL.Panicf("cannot open remote kv as local")
	}

	rngStream := rng.GetStream()
	defer rngStream.Close()
	deviceID, err := getOrInitDeviceID(kv, rngStream)
	if err != nil {
		return nil, err
	}

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rng,
	}

	dummy := &dummyIO{}

	txLog, err := newRemoteWriter(path, deviceID, dummy,
		crypt, kv)
	if err != nil {
		return nil, err
	}
	// Local collective KV's don't have callbacks or collective prefixes
	// Use newVersionedKV directly if this is needed for a test.
	return newVersionedKV(txLog, kv, nil), nil
}

// newVersionedKV returns a versioned KV instance wrapping a remote KV
func newVersionedKV(transactionLog *remoteWriter, kv ekv.KeyValue,
	synchedPrefixes []string) *versionedKV {

	sPrefixes := synchedPrefixes
	if sPrefixes == nil {
		sPrefixes = make([]string, 0)
	}

	remote := newKV(transactionLog, kv)

	v := &versionedKV{
		synchronizedPrefixes: sPrefixes,
		remote:               remote,
		local:                versioned.NewKV(remote),
	}
	return v
}

///////////////////////////////////////////////////////////////////////////////
// Begin Remote KV [storage.versioned.KV] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Get implements [storage.versioned.KV.Get]
func (r *versionedKV) Get(key string, version uint64) (*versioned.Object, error) {
	return r.local.Get(key, version)
}

// GetAndUpgrade implemenets [storage.versioned.KV.GetAndUpgrade]
func (r *versionedKV) GetAndUpgrade(key string, ut versioned.UpgradeTable) (
	*versioned.Object, error) {
	return r.local.GetAndUpgrade(key, ut)
}

// Delete implements [storage.versioned.KV.Delete]
func (r *versionedKV) Delete(key string, version uint64) error {
	return r.local.Delete(key, version)
}

// Set implements [storage.versioned.KV.Set]
// NOT: When calling this, you are responsible for prefixing the
// key with the correct type optionally unique id! Call
// [versioned.MakeKeyWithPrefix] to do so.
// The [Object] should contain the versioning if you are
// maintaining such a functionality.
func (r *versionedKV) Set(key string, object *versioned.Object) error {
	if r.inSynchronizedPrefix {
		k := r.local.GetFullKey(key, object.Version)
		jww.INFO.Printf("Setting Remote: %s", k)
		return r.remote.SetRemote(k, object.Marshal())
	}
	return r.local.Set(key, object)
}

// StoreMapElement stores a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// The version of the value must match the version of the map.
// All Map storage functions update the remote.
func (r *versionedKV) StoreMapElement(mapName,
	elementName string, value *versioned.Object, mapVersion uint64) error {
	if !r.inSynchronizedPrefix && isRemoteKV(r.remote) {
		return errors.New("Map operations must be remote" +
			"operations")
	}

	if value.Version != mapVersion {
		return errors.New("mismatched map and element versions")
	}

	mapKey := r.local.GetFullKey(mapName, mapVersion)

	return r.remote.StoreMapElement(mapKey, elementName, value.Marshal())
}

// StoreMap saves a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMap] function to lock and control
// updates, but it uses [versioned.Object] values.
// the version of values must match the version of the map
// All Map storage functions update the remote.
func (r *versionedKV) StoreMap(mapName string,
	values map[string]*versioned.Object, mapVersion uint64) error {
	if !r.inSynchronizedPrefix && isRemoteKV(r.remote) {
		return errors.New("Map operations must be remote" +
			"operations")
	}

	m := make(map[string][]byte, len(values))

	for key, value := range values {
		if value.Version != mapVersion {
			return errors.New("mismatched map and element versions")
		}
		m[key] = value.Marshal()
	}

	mapKey := r.local.GetFullKey(mapName, mapVersion)

	return r.remote.StoreMap(mapKey, m)
}

// GetMap loads a versioned map from the KV. This relies
// on the underlying remote [KV.GetMap] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) GetMap(mapName string, mapVersion uint64) (
	map[string]*versioned.Object, error) {
	if !r.inSynchronizedPrefix && isRemoteKV(r.remote) {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.local.GetFullKey(mapName, mapVersion)

	var mapVal map[string]*versioned.Object
	m, err := r.remote.GetMap(mapKey)
	if err == nil {
		mapVal, err = mapBytesToVersioned(m)
	}
	return mapVal, err
}

// GetMapElement loads a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) GetMapElement(mapName, elementName string, mapVersion uint64) (
	*versioned.Object, error) {
	if !r.inSynchronizedPrefix && isRemoteKV(r.remote) {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.local.GetFullKey(mapName, mapVersion)

	data, err := r.remote.GetMapElement(mapKey, elementName)
	if err != nil {
		return nil, err
	}

	obj := &versioned.Object{}
	if err = obj.Unmarshal(data); err != nil {
		return nil, err
	}

	return obj, err
}

// DeleteMapElement deletes a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) DeleteMapElement(mapName, elementName string,
	mapVersion uint64) (*versioned.Object, error) {
	if !r.inSynchronizedPrefix && isRemoteKV(r.remote) {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.local.GetFullKey(mapName, mapVersion)

	data, err := r.remote.DeleteMapElement(mapKey, elementName)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	obj := &versioned.Object{}
	if err = obj.Unmarshal(data); err != nil {
		return nil, err
	}

	return obj, err
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *versionedKV) ListenOnRemoteKey(key string, version uint64,
	callback versioned.KeyChangedByRemoteCallback, localEvents bool) error {

	versionedKey := r.local.GetFullKey(key, version)

	wrap := func(old, new []byte, op versioned.KeyOperation) {
		var oldObj *versioned.Object
		if old != nil && len(old) > 0 {
			oldObj = &versioned.Object{}
			err := oldObj.Unmarshal(old)
			if err != nil {
				jww.WARN.Printf("Failed to unmarshal old versioned object "+
					"for listener on key %s: %+v", key, err)
			}
		}

		var newObj *versioned.Object
		if op != versioned.Deleted {
			newObj = &versioned.Object{}
			err := newObj.Unmarshal(new)
			if err != nil {
				jww.FATAL.Panicf("Failed to unmarshal new versioned object "+
					"for listener on key %s", key)
			}
		}

		callback(oldObj, newObj, op)
	}

	return r.remote.ListenOnRemoteKey(versionedKey, wrap, localEvents)
}

// ListenOnRemoteMap allows the caller to receive updates when
// the map or map elements are updated
func (r *versionedKV) ListenOnRemoteMap(mapName string, version uint64,
	callback versioned.MapChangedByRemoteCallback, localEvents bool) error {

	versionedMap := r.local.GetFullKey(mapName, version)

	wrap := func(edits map[string]elementEdit) {
		versionedEdits := make(map[string]versioned.ElementEdit, len(edits))

		for key, edit := range edits {
			versionedEdit := versioned.ElementEdit{
				OldElement: &versioned.Object{},
				NewElement: &versioned.Object{},
				Operation:  edit.Operation,
			}
			if edit.OldElement != nil && len(edit.OldElement) > 0 {
				if err := versionedEdit.OldElement.Unmarshal(edit.OldElement); err != nil {
					jww.WARN.Printf("Failed to unmarshal old versioned object "+
						"for listener on map %s element %s: %+v", mapName, key, err)
				}
			}

			if err := versionedEdit.NewElement.Unmarshal(edit.NewElement); err != nil {
				jww.FATAL.Printf("Failed to unmarshal new versioned object "+
					"for listener on map %s element %s", mapName, key)
			}

			if bytes.Equal(versionedEdit.OldElement.Data, versionedEdit.NewElement.Data) {
				continue
			}

			versionedEdits[key] = versionedEdit
		}

		callback(versionedEdits)
	}

	return r.remote.ListenOnRemoteMap(versionedMap, wrap, localEvents)
}

// GetPrefix implements [storage.versioned.KV.GetPrefix]
func (r *versionedKV) GetPrefix() string {
	return r.local.GetPrefix()
}

// HasPrefix implements [storage.versioned.KV.HasPrefix]
func (r *versionedKV) HasPrefix(prefix string) bool {
	return r.local.HasPrefix(prefix)
}

// Prefix implements [storage.versioned.KV.Prefix]
func (r *versionedKV) Prefix(prefix string) (versioned.KV, error) {
	subKV, err := r.local.Prefix(prefix)
	if err == nil {
		v := &versionedKV{
			synchronizedPrefixes: r.synchronizedPrefixes,
			remote:               r.remote,
			local:                subKV,
		}
		v.updateIfSynchronizedPrefix()
		return v, nil
	}
	return nil, err
}

func (r *versionedKV) Root() versioned.KV {
	v := &versionedKV{
		synchronizedPrefixes: r.synchronizedPrefixes,
		remote:               r.remote,
		local:                r.local.Root(),
	}
	v.updateIfSynchronizedPrefix()
	return v
}

// IsMemStore implements [storage.versioned.KV.IsMemStore]
func (r *versionedKV) IsMemStore() bool {
	return r.local.IsMemStore()
}

// GetFullKey implements [storage.versioned.KV.GetFullKey]
func (r *versionedKV) GetFullKey(key string, version uint64) string {
	return r.local.GetFullKey(key, version)
}

// Exists implements [storage.versioned.KV.Exists]
func (r *versionedKV) Exists(err error) bool {
	return r.local.Exists(err)
}

func mapBytesToVersioned(m map[string][]byte) (map[string]*versioned.Object,
	error) {
	versionedM := make(map[string]*versioned.Object, len(m))

	for key, data := range m {
		obj := &versioned.Object{}
		if err := obj.Unmarshal(data); err != nil {
			return nil, errors.Wrapf(err, "with key: %s", key)
		}
		versionedM[key] = obj
	}
	return versionedM, nil
}

///////////////////////////////////////////////////////////////////////////////
// End Remote KV [storage.versioned.KV] interface
///////////////////////////////////////////////////////////////////////////////

func (r *versionedKV) StartProcesses() (stoppable.Stoppable, error) {
	return r.remote.StartProcesses()
}

func (r *versionedKV) RegisterConnectionTracker(nc NotifyCallback) {
	r.remote.RegisterConnectionTracker(nc)
}

func (r *versionedKV) IsConnected() bool {
	return r.remote.IsConnected()
}

func (r *versionedKV) IsSynched() bool {
	return r.remote.IsSynched()
}

// WaitForRemote block until timeout or remote operations complete
func (r *versionedKV) WaitForRemote(timeout time.Duration) bool {
	return r.remote.WaitForRemote(timeout)
}

func (r *versionedKV) Remote() RemoteKV {
	return r.remote
}

func (r *versionedKV) updateIfSynchronizedPrefix() bool {
	for i := range r.synchronizedPrefixes {
		if r.local.HasPrefix(r.synchronizedPrefixes[i]) {
			r.inSynchronizedPrefix = true
			return true
		}
	}
	r.inSynchronizedPrefix = false
	return false
}

func cleanKey(key string) string {

	versionLoc := -1
	prefixLoc := -1

	for i := len(key) - 1; i > 0; i-- {
		if versionLoc == -1 && key[i] == "_"[0] {
			versionLoc = i
		}
		if key[i] == versioned.PrefixSeparator[0] {
			prefixLoc = i
			// prefix always is after version, so we can break
			break
		}
	}

	cleanedKey := key[:versionLoc]
	if prefixLoc != -1 {
		cleanedKey = cleanedKey[prefixLoc+1:]
	}
	return cleanedKey
}

func getOrInitDeviceID(kv ekv.KeyValue, rng io.Reader) (InstanceID, error) {
	deviceID, err := GetInstanceID(kv)
	// if Instance id doesn't exist, create one.
	if err != nil {
		if !ekv.Exists(err) {
			deviceID, err = InitInstanceID(kv, rng)
		}
		if err != nil {
			return InstanceID{}, errors.WithStack(err)
		}
	}
	return deviceID, errors.WithStack(err)
}
