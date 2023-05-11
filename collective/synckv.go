////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"io"
	"sync"
	"sync/atomic"
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

	// is the synchronization thread active?
	isSynchronizing *atomic.Bool
	mux             sync.Mutex

	col   *collector
	txLog *remoteWriter
	// remoteKV is the remote synching KV instance. This is used
	// when we intercept Set calls because we are synchronizing this prefix.
	remoteKV *internalKV
	// vkv is a versioned KV instance that wraps the remoteKV, used
	// for all local operations.
	vkv versioned.KV
}

// SynchronizedKV loads or creates a synchronized remote KV that uses
// a remote RemoteStore to store defined synchronization prefixes to the
// network.
func SynchronizedKV(path string, deviceSecret []byte,
	remote RemoteStore, kv ekv.KeyValue, synchedPrefixes []string,
	rng *fastRNG.StreamGenerator) (SyncKV, error) {

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

	vkv.col = newCollector(deviceID, path, remote, vkv.remoteKV, crypt, txLog)

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
		remoteKV:             remote,
		vkv:                  versioned.NewKV(remote),
	}
	return v
}

///////////////////////////////////////////////////////////////////////////////
// Begin Remote KV [storage.versioned.KV] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Get implements [storage.versioned.KV.Get]
func (r *versionedKV) Get(key string, version uint64) (*versioned.Object, error) {
	return r.vkv.Get(key, version)
}

// GetAndUpgrade implemenets [storage.versioned.KV.GetAndUpgrade]
func (r *versionedKV) GetAndUpgrade(key string, ut versioned.UpgradeTable) (
	*versioned.Object, error) {
	return r.vkv.GetAndUpgrade(key, ut)
}

// Delete implements [storage.versioned.KV.Delete]
func (r *versionedKV) Delete(key string, version uint64) error {
	return r.vkv.Delete(key, version)
}

// Set implements [storage.versioned.KV.Set]
// NOT: When calling this, you are responsible for prefixing the
// key with the correct type optionally unique id! Call
// [versioned.MakeKeyWithPrefix] to do so.
// The [Object] should contain the versioning if you are
// maintaining such a functionality.
func (r *versionedKV) Set(key string, object *versioned.Object) error {
	if r.inSynchronizedPrefix {
		k := r.vkv.GetFullKey(key, object.Version)
		return r.remoteKV.SetRemote(k, object.Marshal())
	}
	return r.vkv.Set(key, object)
}

// StoreMapElement stores a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// The version of the value must match the version of the map.
// All Map storage functions update the remote.
func (r *versionedKV) StoreMapElement(mapName,
	elementName string, value *versioned.Object, mapVersion uint64) error {
	if !r.inSynchronizedPrefix {
		return errors.New("Map operations must be remote" +
			"operations")
	}

	if value.Version != mapVersion {
		return errors.New("mismatched map and element versions")
	}

	mapKey := r.vkv.GetFullKey(mapName, mapVersion)

	return r.remoteKV.StoreMapElement(mapKey, elementName, value.Marshal())
}

// StoreMap saves a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMap] function to lock and control
// updates, but it uses [versioned.Object] values.
// the version of values must match the version of the map
// All Map storage functions update the remote.
func (r *versionedKV) StoreMap(mapName string,
	values map[string]*versioned.Object, mapVersion uint64) error {
	if !r.inSynchronizedPrefix {
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

	mapKey := r.vkv.GetFullKey(mapName, mapVersion)

	return r.remoteKV.StoreMap(mapKey, m)
}

// GetMap loads a versioned map from the KV. This relies
// on the underlying remote [KV.GetMap] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) GetMap(mapName string, mapVersion uint64) (
	map[string]*versioned.Object, error) {
	if !r.inSynchronizedPrefix {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.vkv.GetFullKey(mapName, mapVersion)

	m, err := r.remoteKV.GetMap(mapKey)
	if err != nil {
		return nil, err
	}

	versionedM := make(map[string]*versioned.Object, len(m))

	for key, data := range m {
		obj := &versioned.Object{}
		if err = obj.Unmarshal(data); err != nil {
			return nil, err
		}
		versionedM[key] = obj
	}

	return versionedM, nil
}

// GetMapElement loads a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) GetMapElement(mapName, elementName string, mapVersion uint64) (
	*versioned.Object, error) {
	if !r.inSynchronizedPrefix {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.vkv.GetFullKey(mapName, mapVersion)

	data, err := r.remoteKV.GetMapElement(mapKey, elementName)
	if err != nil {
		return nil, err
	}

	obj := &versioned.Object{}
	if err = obj.Unmarshal(data); err != nil {
		return nil, err
	}

	// FIXME: this needs to be synchronized
	err = r.vkv.Delete(mapKey, mapVersion)

	return obj, err
}

// DeleteMapElement loads a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *versionedKV) DeleteMapElement(mapName, elementName string,
	mapVersion uint64) (*versioned.Object, error) {
	if !r.inSynchronizedPrefix {
		return nil, errors.New("Map operations must be remote" +
			"operations")
	}

	mapKey := r.vkv.GetFullKey(mapName, mapVersion)

	data, err := r.remoteKV.GetMapElement(mapKey, elementName)
	if err != nil {
		return nil, err
	}

	obj := &versioned.Object{}
	if err = obj.Unmarshal(data); err != nil {
		return nil, err
	}

	return obj, err
}

// Transaction locks a key while it is being mutated then stores the result
// and returns the old value if it existed.
// Transactions cannot be remote operations
// If the op returns an error, the operation will be aborted.
func (r *versionedKV) Transaction(key string, op versioned.TransactionOperation,
	version uint64) (*versioned.Object, bool, error) {

	if r.inSynchronizedPrefix {
		return nil, false, errors.New("Transactions cannot be remote" +
			"operations")
	}

	fullKey := r.vkv.GetFullKey(key, version)

	var oldObj *versioned.Object

	wrapper := func(old []byte, existed bool) (data []byte, delete bool, err error) {
		oldObj = &versioned.Object{}
		err = oldObj.Unmarshal(old)
		if err != nil {
			return nil, false, err
		}
		newObj, err := op(oldObj, existed)
		if err != nil {
			return nil, false, err
		}

		return newObj.Marshal(), false, nil
	}

	_, existed, err := r.remoteKV.Transaction(fullKey, wrapper)

	return oldObj, existed, err
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *versionedKV) ListenOnRemoteKey(key string, version uint64,
	callback versioned.KeyChangedByRemoteCallback) (*versioned.Object,
	error) {

	r.mux.Lock()
	defer r.mux.Unlock()

	if r.isSynchronizing.Load() {
		jww.FATAL.Panic("cannot add listener when synchronizing")
	}

	versionedKey := r.vkv.GetFullKey(key, version)

	wrap := func(key string, old, new []byte, op versioned.KeyOperation) {
		var oldObj *versioned.Object
		if old != nil {
			oldObj = &versioned.Object{}
			err := oldObj.Unmarshal(old)
			if err != nil {
				jww.WARN.Printf("Failed to unmarshal old versioned object "+
					"for listener on key %s", key)
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

		cleanedKey := cleanKey(key)

		callback(cleanedKey, oldObj, newObj, op)
	}

	r.remoteKV.ListenOnRemoteKey(versionedKey, wrap)

	return r.Get(key, version)
}

// ListenOnRemoteMap allows the caller to receive updates when
// the map or map elements are updated
func (r *versionedKV) ListenOnRemoteMap(mapName string, version uint64,
	callback versioned.MapChangedByRemoteCallback) (
	map[string]*versioned.Object, error) {

	r.mux.Lock()
	defer r.mux.Unlock()

	if r.isSynchronizing.Load() {
		jww.FATAL.Panic("cannot add map listener when synchronizing")
	}

	versionedMap := r.vkv.GetFullKey(mapName, version)

	wrap := func(mapName string, edits map[string]elementEdit) {
		versionedEdits := make(map[string]versioned.ElementEdit, len(edits))

		for key, edit := range edits {
			versionedEdit := versioned.ElementEdit{
				OldElement: &versioned.Object{},
				NewElement: &versioned.Object{},
				Operation:  edit.Operation,
			}

			if err := versionedEdit.OldElement.Unmarshal(edit.OldElement); err != nil {
				jww.WARN.Printf("Failed to unmarshal old versioned object "+
					"for listener on map %s element %s", mapName, key)
			}

			if err := versionedEdit.NewElement.Unmarshal(edit.NewElement); err != nil {
				jww.FATAL.Printf("Failed to unmarshal new versioned object "+
					"for listener on map %s element %s", mapName, key)
			}

			versionedEdits[key] = versionedEdit
		}

		cleanedMapName := cleanKey(mapName)
		callback(cleanedMapName, versionedEdits)
	}

	r.remoteKV.ListenOnRemoteMap(versionedMap, wrap)

	return r.GetMap(mapName, version)
}

// GetPrefix implements [storage.versioned.KV.GetPrefix]
func (r *versionedKV) GetPrefix() string {
	return r.vkv.GetPrefix()
}

// HasPrefix implements [storage.versioned.KV.HasPrefix]
func (r *versionedKV) HasPrefix(prefix string) bool {
	return r.vkv.HasPrefix(prefix)
}

// Prefix implements [storage.versioned.KV.Prefix]
func (r *versionedKV) Prefix(prefix string) (versioned.KV, error) {
	subKV, err := r.vkv.Prefix(prefix)
	if err == nil {
		v := &versionedKV{
			synchronizedPrefixes: r.synchronizedPrefixes,
			col:                  r.col,
			txLog:                r.txLog,
			remoteKV:             r.remoteKV,
			vkv:                  subKV,
		}
		v.updateIfSynchronizedPrefix()
		return v, nil
	}
	return nil, err
}

func (r *versionedKV) Root() versioned.KV {
	v := &versionedKV{
		synchronizedPrefixes: r.synchronizedPrefixes,
		col:                  r.col,
		txLog:                r.txLog,
		remoteKV:             r.remoteKV,
		vkv:                  r.vkv.Root(),
	}
	v.updateIfSynchronizedPrefix()
	return v
}

// IsMemStore implements [storage.versioned.KV.IsMemStore]
func (r *versionedKV) IsMemStore() bool {
	return r.vkv.IsMemStore()
}

// GetFullKey implements [storage.versioned.KV.GetFullKey]
func (r *versionedKV) GetFullKey(key string, version uint64) string {
	return r.vkv.GetFullKey(key, version)
}

// Exists implements [storage.versioned.KV.Exists]
func (r *versionedKV) Exists(err error) bool {
	return r.vkv.Exists(err)
}

///////////////////////////////////////////////////////////////////////////////
// End Remote KV [storage.versioned.KV] interface
///////////////////////////////////////////////////////////////////////////////

func (r *versionedKV) StartProcesses() (stoppable.Stoppable, error) {

	// Lock up while we start to prevent Listen functions from overlapping
	// with this function.
	r.mux.Lock()
	defer r.mux.Unlock()

	r.isSynchronizing.Store(true)

	// Construct stoppables
	multiStoppable := stoppable.NewMulti(syncStoppable)

	if r.col != nil {
		colStopper := stoppable.NewSingle(collectorRunnerStoppable)
		multiStoppable.Add(colStopper)
		go r.col.runner(colStopper)
	}

	writerStopper := stoppable.NewSingle(writerRunnerStoppable)
	multiStoppable.Add(writerStopper)
	go r.txLog.Runner(writerStopper)

	// Switch my state back to not synchronizing when stopped
	myStopper := stoppable.NewSingle(syncStoppable + "_synchronizing")
	go func(s *stoppable.Single) {
		<-s.Quit()
		r.isSynchronizing.Store(false)
	}(myStopper)
	multiStoppable.Add(myStopper)

	return multiStoppable, nil
}

func (r *versionedKV) RegisterConnectionTracker(nc NotifyCallback) {
	r.col.Register(nc)
	go nc(r.col.IsConnected())
}

func (r *versionedKV) IsConnected() bool {
	return r.col.IsConnected()
}

func (r *versionedKV) IsSynched() bool {
	return r.IsSynched()
}

// WaitForRemote block until timeout or remote operations complete
func (r *versionedKV) WaitForRemote(timeout time.Duration) bool {
	return r.col.WaitUntilSynched(timeout)
}

func (r *versionedKV) Remote() RemoteKV {
	return r.remoteKV
}

func (r *versionedKV) updateIfSynchronizedPrefix() bool {
	for i := range r.synchronizedPrefixes {
		if r.vkv.HasPrefix(r.synchronizedPrefixes[i]) {
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
			return InstanceID{}, err
		}
	}
	return deviceID, err
}
