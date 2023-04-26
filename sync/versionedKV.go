////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"sync"
	"time"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

// VersionedKV wraps a [sync.KV] inside of a [storage.versioned.KV] interface.
type VersionedKV struct {
	// synchronizedPrefixes are prefixes that trigger remote
	// synchronization calls.
	synchronizedPrefixes []string

	// hasSynchronizedPrefix tells us we are in a prefix that is synchronized.
	inSynchronizedPrefix bool

	// remoteKV is the remote synching KV instance. This is used
	// when we intercept Set calls because we are synchronizing this prefix.
	remoteKV *internalKV
	// vkv is a versioned KV instance that wraps the remoteKV, used
	// for all local operations.
	vkv versioned.KV

	lck sync.Mutex
}

// NewVersionedKV returns a versioned KV instance wrapping a remote KV
func NewVersionedKV(transactionLog *TransactionLog, kv ekv.KeyValue,
	synchedPrefixes []string,
	eventCb KeyUpdateCallback,
	updateCb RemoteStoreCallback) (*VersionedKV, error) {

	sPrefixes := synchedPrefixes
	if sPrefixes == nil {
		sPrefixes = make([]string, 0)
	}

	remote, err := newKV(transactionLog, kv, eventCb, updateCb)
	if err != nil {
		return nil, err
	}

	v := &VersionedKV{
		synchronizedPrefixes: sPrefixes,
		remoteKV:             remote,
		vkv:                  versioned.NewKV(remote),
	}
	v.updateIfSynchronizedPrefix()

	return v, nil
}

///////////////////////////////////////////////////////////////////////////////
// Begin Remote KV [storage.versioned.KV] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Get implements [storage.versioned.KV.Get]
func (r *VersionedKV) Get(key string, version uint64) (*versioned.Object, error) {
	return r.vkv.Get(key, version)
}

// GetAndUpgrade implemenets [storage.versioned.KV.GetAndUpgrade]
func (r *VersionedKV) GetAndUpgrade(key string, ut versioned.UpgradeTable) (
	*versioned.Object, error) {
	return r.vkv.GetAndUpgrade(key, ut)
}

// Delete implements [storage.versioned.KV.Delete]
func (r *VersionedKV) Delete(key string, version uint64) error {
	return r.vkv.Delete(key, version)
}

// Set implements [storage.versioned.KV.Set]
// NOT: When calling this, you are responsible for prefixing the
// key with the correct type optionally unique id! Call
// [versioned.MakeKeyWithPrefix] to do so.
// The [Object] should contain the versioning if you are
// maintaining such a functionality.
func (r *VersionedKV) Set(key string, object *versioned.Object) error {
	if r.inSynchronizedPrefix {
		k := r.vkv.GetFullKey(key, object.Version)
		return r.remoteKV.SetRemote(k, object.Marshal(), nil)
	}
	return r.vkv.Set(key, object)
}

// GetPrefix implements [storage.versioned.KV.GetPrefix]
func (r *VersionedKV) GetPrefix() string {
	return r.vkv.GetPrefix()
}

// HasPrefix implements [storage.versioned.KV.HasPrefix]
func (r *VersionedKV) HasPrefix(prefix string) bool {
	return r.vkv.HasPrefix(prefix)
}

// Prefix implements [storage.versioned.KV.Prefix]
func (r *VersionedKV) Prefix(prefix string) (versioned.KV, error) {
	newKV, err := r.vkv.Prefix(prefix)
	if err == nil {
		v := &VersionedKV{
			synchronizedPrefixes: r.synchronizedPrefixes,
			remoteKV:             r.remoteKV,
			vkv:                  newKV,
		}
		v.updateIfSynchronizedPrefix()
		return v, nil
	}
	return nil, err
}

func (r *VersionedKV) Root() versioned.KV {
	v := &VersionedKV{
		synchronizedPrefixes: r.synchronizedPrefixes,
		remoteKV:             r.remoteKV,
		vkv:                  r.vkv.Root(),
	}
	v.updateIfSynchronizedPrefix()
	return v
}

// IsMemStore implements [storage.versioned.KV.IsMemStore]
func (r *VersionedKV) IsMemStore() bool {
	return r.vkv.IsMemStore()
}

// GetFullKey implements [storage.versioned.KV.GetFullKey]
func (r *VersionedKV) GetFullKey(key string, version uint64) string {
	return r.vkv.GetFullKey(key, version)
}

// Exists implements [storage.versioned.KV.Exists]
func (r *VersionedKV) Exists(err error) bool {
	return r.vkv.Exists(err)
}

///////////////////////////////////////////////////////////////////////////////
// End Remote KV [storage.versioned.KV] interface
///////////////////////////////////////////////////////////////////////////////

// SyncPrefix adds a prefix to the synchronization list. Please note that when
// you call [VersionedKV.Prefix] it creates a new object, so calls to this will
// not update other instances of the kv.
func (r *VersionedKV) SyncPrefix(prefix string) {
	r.lck.Lock()
	defer r.lck.Unlock()
	for i := range r.synchronizedPrefixes {
		if prefix == r.synchronizedPrefixes[i] {
			return
		}
	}
	r.synchronizedPrefixes = append(r.synchronizedPrefixes, prefix)
}

// StoreMapElement stores a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
func (r *VersionedKV) StoreMapElement(mapName, elementKey string,
	value *versioned.Object, version uint64) error {
	// Generate the full key mapping (Prefixes + mapName + objectVersion)
	mapFullKey := r.GetFullKey(mapName, version)
	return r.remoteKV.StoreMapElement(mapFullKey, elementKey,
		value.Marshal(), r.inSynchronizedPrefix)
}

// StoreMap saves a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMap] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
func (r *VersionedKV) StoreMap(mapName string,
	value map[string]*versioned.Object, version uint64) error {
	mapFullKey := r.GetFullKey(mapName, version)
	newMap := make(map[string][]byte, len(value))
	for k, v := range value {
		// we don't have to prepend the fullkey because the mapName
		// will be prepended
		newMap[k] = v.Marshal()
	}
	return r.remoteKV.StoreMap(mapFullKey, newMap, r.inSynchronizedPrefix)
}

// GetMap loads a versioned map from the KV. This relies
// on the underlying remote [KV.GetMap] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *VersionedKV) GetMap(mapName string, version uint64) (
	map[string]*versioned.Object, error) {
	mapFullKey := r.GetFullKey(mapName, version)
	mapData, err := r.remoteKV.GetMap(mapFullKey)
	if err != nil {
		return nil, err
	}

	newMap := make(map[string]*versioned.Object, len(mapData))
	for k, v := range mapData {
		obj := versioned.Object{}
		err = obj.Unmarshal(v)
		if err != nil {
			return nil, err
		}
		newMap[k] = &obj
	}
	return newMap, nil
}

// GetMapElement loads a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *VersionedKV) GetMapElement(mapName, element string, version uint64) (
	*versioned.Object, error) {
	mapFullKey := r.GetFullKey(mapName, version)
	data, err := r.remoteKV.GetMapElement(mapFullKey, element)
	if err != nil {
		return nil, err
	}

	obj := versioned.Object{}
	err = obj.Unmarshal(data)
	return &obj, err
}

func (r *VersionedKV) Remote() RemoteKV {
	return r.remoteKV
}

func (r *VersionedKV) updateIfSynchronizedPrefix() bool {
	r.lck.Lock()
	defer r.lck.Unlock()
	for i := range r.synchronizedPrefixes {
		if r.vkv.HasPrefix(r.synchronizedPrefixes[i]) {
			r.inSynchronizedPrefix = true
			return true
		}
	}
	r.inSynchronizedPrefix = false
	return false
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *VersionedKV) ListenOnRemoteKey(key string,
	callback versioned.KeyChangedByRemoteCallback) {
	r.remoteKV.ListenOnRemoteKey(key, callback)
}

// WaitForRemote block until timeout or remote operations complete
func (r *VersionedKV) WaitForRemote(timeout time.Duration) bool {
	return r.remoteKV.WaitForRemote(timeout)
}
