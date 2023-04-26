////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
)

///////////////////////////////////////////////////////////////////////////////
// KV Implementation
///////////////////////////////////////////////////////////////////////////////

// KV kv-related constants.
const (
	remoteKvVersion = 0

	intentsVersion = 0
	intentsKey     = "intentsVersion"

	// map handler constants
	mapKeysListSuffix  = "_🗺️MapKeys"
	mapKeysListFmt     = "%s" + mapKeysListSuffix
	mapElementKeyFmt   = "%s_🗺️MapElement_%s"
	usedMapSuffixError = "cannot use \"" + mapKeysListSuffix + "\" at the " +
		"end of a key, it is reserved"

	// KeyUpdateCallback statuses.
	Disconnected = "Disconnected"
	Connected    = "Connected"
	Successful   = "UpdatedKey"

	// remote key
	isRemoteKey = "kv_remote_status_0"
)

var (
	// is remote or local values
	kvIsRemoteVal = []byte("REMOTE")
)

// updateFailureDelay is the backoff period in between retrying to
const updateFailureDelay = 1 * time.Second

// RemoteKV exposes some internal KV functions. you cannot create a RemoteKV,
// and generally you should never access it on the VersionedKV object. It is
// provided so that external xxdk libraries can access specific functionality.
// This is considered internal api and may be changed or removed at any time.
type RemoteKV interface {
	ekv.KeyValue

	// SetRemote will write a transaction to the remote and local store
	// with the specified RemoteCB RemoteStoreCallback
	SetRemote(key string, val []byte, updateCb RemoteStoreCallback) error

	// UpsertLocal performs an upsert operation and sets the resultant
	// value to the local EKV. It is a LOCAL ONLY operation which will
	// write the Transaction to local store.
	UpsertLocal(key string, newVal []byte) error
}

// internalKV implements a remote internalKV to handle transaction logs.
type internalKV struct {
	// local is the local EKV store that will write the transaction.
	local ekv.KeyValue

	// txLog is the transaction log used to write transactions.
	txLog *TransactionLog

	UpdateListenerMux sync.RWMutex

	// KeyUpdateListeners holds callbacks called when a key is updated
	// by a remote
	KeyUpdateListeners map[string]versioned.KeyChangedByRemoteCallback

	// KeyUpdate is the callback used to report events when
	// attempting to call Set.
	KeyUpdate KeyUpdateCallback

	// list of tracked keys
	tracked []string

	// UnsyncedWrites is the pending writes that we are waiting
	// for on remote storage. Anytime this is not empty, we are
	// not synchronized and this should be reported.
	UnsyncedWrites map[string][]byte

	// defaulteUpdateCB is called when the updateCB is not specified for
	// remote store and set operations
	defaultRemoteWriteCB RemoteStoreCallback

	// Connected determines the connectivity of the remote server.
	connected bool

	// lck is used to lck access to Remote, if no remote needed,
	// then the underlying kv lock is all that is needed.
	lck sync.RWMutex

	// mapLck prevents asynchronous map operations
	mapLck       sync.Mutex
	openRoutines int32
}

// newKV constructs a new remote KV. If data exists on disk, it loads
// that context and handle it appropriately.
func newKV(transactionLog *TransactionLog, kv ekv.KeyValue,
	eventCb KeyUpdateCallback,
	updateCb RemoteStoreCallback) (*internalKV, error) {

	rkv := &internalKV{
		local:                kv,
		txLog:                transactionLog,
		KeyUpdate:            eventCb,
		UnsyncedWrites:       make(map[string][]byte, 0),
		defaultRemoteWriteCB: updateCb,
		connected:            true,
	}

	if err := rkv.loadUnsyncedWrites(); err != nil {
		return nil, err
	}

	// Re-trigger all lingering intents
	rkv.lck.Lock()
	for key := range rkv.UnsyncedWrites {
		// Call the internal to avoid writing to intent what
		// is already there
		k := key
		v := rkv.UnsyncedWrites[k]
		atomic.AddInt32(&rkv.openRoutines, 1)
		go func() {
			rkv.remoteSet(k, v, updateCb)
			atomic.AddInt32(&rkv.openRoutines, -1)
		}()
	}
	rkv.lck.Unlock()

	return rkv, nil
}

///////////////////////////////////////////////////////////////////////////////
// Begin KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Set implements [ekv.KeyValue.Set]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) Set(key string, objectToStore ekv.Marshaler) error {
	if strings.HasSuffix(key, mapKeysListSuffix) {
		return errors.New(usedMapSuffixError)
	}
	return r.SetBytes(key, objectToStore.Marshal())
}

// Get implements [ekv.KeyValue.Get]
func (r *internalKV) Get(key string, loadIntoThisObject ekv.Unmarshaler) error {
	data, err := r.GetBytes(key)
	if err != nil {
		return err
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Delete implements [ekv.KeyValue.Delete]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) Delete(key string) error {
	if strings.HasSuffix(key, mapKeysListSuffix) {
		return errors.New(usedMapSuffixError)
	}
	return r.local.Delete(key)
}

// SetInterface implements [ekv.KeyValue.SetInterface]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetInterface(key string, objectToStore interface{}) error {
	if strings.HasSuffix(key, mapKeysListSuffix) {
		return errors.New(usedMapSuffixError)
	}

	data, err := json.Marshal(objectToStore)
	if err != nil {
		return err
	}
	return r.SetBytes(key, data)
}

// GetInterface implements [ekv.KeyValue.GetInterface]
func (r *internalKV) GetInterface(key string, objectToLoad interface{}) error {
	data, err := r.GetBytes(key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, objectToLoad)
}

// SetBytes implements [ekv.KeyValue.SetBytes]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetBytes(key string, data []byte) error {
	if strings.HasSuffix(key, mapKeysListSuffix) {
		return errors.New(usedMapSuffixError)
	}

	return r.setBytesUnsafe(key, data)
}

// SetBytesFromRemote implements [ekv.KeyValue.SetBytes].
// This is a LOCAL ONLY operation which will write the Transaction
// to local store. Only use this from the collector system, designed
// to allow event models to connect to the write.
func (r *internalKV) SetBytesFromRemote(key string, data []byte) error {

	if err := r.setBytesUnsafe(key, data); err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if cb, exists := r.KeyUpdateListeners[key]; exists {
		go cb(key)
	}
	return nil
}

// SetBytesUnsafe implements [ekv.KeyValue.SetBytes]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
// Does not check for reserved keys
func (r *internalKV) setBytesUnsafe(key string, data []byte) error {
	return r.local.SetBytes(key, data)
}

// GetBytes implements [ekv.KeyValue.GetBytes]
func (r *internalKV) GetBytes(key string) ([]byte, error) {
	return r.local.GetBytes(key)
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *internalKV) ListenOnRemoteKey(key string, callback versioned.KeyChangedByRemoteCallback) {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()
	r.KeyUpdateListeners[key] = callback
}

///////////////////////////////////////////////////////////////////////////////
// End KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// UpsertLocal performs an upsert operation and sets the resultant
// value to the local EKV. It is a LOCAL ONLY operation which will
// write the Transaction to local store.
// todo: test this
func (r *internalKV) UpsertLocal(key string, newVal []byte) error {
	// Read from local KV
	obj, err := r.local.GetBytes(key)
	if err != nil {
		// Error means key does not exist, simply write to local
		return r.local.SetBytes(key, newVal)
	}

	if bytes.Equal(obj, newVal) {
		jww.TRACE.Printf("duplicate transaction value for key %s", key)
		return nil
	}

	if r.KeyUpdate != nil {
		r.KeyUpdate(key, obj, newVal, true)
	}

	return r.local.SetBytes(key, newVal)
}

// SetRemote will write a transaction to the remote and local store
// with the specified RemoteCB RemoteStoreCallback
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetRemote(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	if strings.HasSuffix(key, mapKeysListSuffix) {
		return errors.New(usedMapSuffixError)
	}

	return r.setRemoteUnsafe(key, val, updateCb)
}

// SetRemote will write a transaction to the remote and local store
// with the specified RemoteCB RemoteStoreCallback
func (r *internalKV) setRemoteUnsafe(key string, val []byte,
	updateCb RemoteStoreCallback) error {

	r.lck.Lock()
	defer r.lck.Unlock()

	// Add intent to write transaction
	if err := r.addUnsyncedWrite(key, val); err != nil {
		return err
	}

	// Save locally
	if err := r.SetBytes(key, val); err != nil {
		return errors.Errorf("failed to write to local kv: %+v", err)
	}

	return r.remoteSet(key, val, updateCb)
}

// SetRemoteOnly will place this Transaction onto the remote server. This is an
// asynchronous operation and results will be passed back via the
// RemoteStoreCallback.
//
// NO LOCAL STORAGE OPERATION WIL BE PERFORMED.
func (r *internalKV) SetRemoteOnly(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	return r.remoteSet(key, val, updateCb)
}

// StoreMapElement saves a given map element and updates
// the map keys list if it is a new key.
// All Map storage functions update the remote.
func (r *internalKV) StoreMapElement(mapName, elementKey string, value []byte,
	sync bool) error {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()
	return r.storeMapElement(mapName, elementKey, value, sync)
}

// keep this private method here because it is the logic of StoreMapElement
// without the lock.
func (r *internalKV) storeMapElement(mapName, elementKey string, value []byte,
	sync bool) error {
	// Store the element
	key := fmt.Sprintf(mapElementKeyFmt, mapName, elementKey)
	var err error
	if sync {
		err = r.SetRemote(key, value, nil)
		if err != nil {
			return err
		}
	} else {
		err = r.SetBytes(key, value)
	}
	if err != nil {
		return err
	}

	// Detect if this key is new, and update mapkeys if so
	existingKeys, err := r.getMapKeys(mapName)
	if err != nil {
		return err
	}
	_, ok := existingKeys[elementKey]
	if !ok {
		existingKeys[elementKey] = struct{}{}
		r.storeMapKeys(mapName, existingKeys, sync)
	}

	return nil
}

// StoreMap saves each element of the map, then updates the map structure
// and deletes no longer used keys in the map.
// All Map storage functions update the remote.
func (r *internalKV) StoreMap(mapName string, value map[string][]byte,
	sync bool) error {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()

	oldKeys, err := r.getMapKeys(mapName)
	if err != nil {
		return err
	}

	// Store each element, then the keys for the map
	newKeys := make(map[string]struct{})
	for k, v := range value {
		newKeys[k] = struct{}{}
		err := r.storeMapElement(mapName, k, v, sync)
		if err != nil {
			return err
		}
	}
	err = r.storeMapKeys(mapName, newKeys, sync)
	if err != nil {
		return err
	}

	// Check if any elements need to be deleted
	for k := range oldKeys {
		_, ok := newKeys[k]
		if ok {
			continue
		}
		// Delete this element
		key := fmt.Sprintf(mapElementKeyFmt, mapName, k)
		err := r.Delete(key)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetMapElement looks up the element for the given map
func (r *internalKV) GetMapElement(mapName, elementKey string) ([]byte, error) {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()
	key := fmt.Sprintf(mapElementKeyFmt, mapName, elementKey)
	return r.GetBytes(key)
}

// GetMap returns all values inside a map
func (r *internalKV) GetMap(mapName string) (map[string][]byte, error) {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()

	// Get the current list of keys
	keys, err := r.getMapKeys(mapName)
	if err != nil {
		// Exists(err) is already checked in getMapKeys
		return nil, err
	}

	// Load each key into a new map to return
	ret := make(map[string][]byte, len(keys))
	for k := range keys {
		key := fmt.Sprintf(mapElementKeyFmt, mapName, k)
		ret[k], err = r.GetBytes(key)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (r *internalKV) storeMapKeys(mapName string, keys map[string]struct{},
	sync bool) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(mapKeysListFmt, mapName)
	if sync {
		return r.setRemoteUnsafe(key, data, nil)
	} else {
		return r.setBytesUnsafe(key, data)
	}
}

func (r *internalKV) getMapKeys(mapName string) (map[string]struct{}, error) {
	keys := make(map[string]struct{})

	key := fmt.Sprintf(mapKeysListFmt, mapName)
	data, err := r.GetBytes(key)
	if err != nil {
		// If the key doesn't exist then this is not an error
		// return empty object
		if !ekv.Exists(err) {
			return keys, nil
		}
		return nil, err
	}

	err = json.Unmarshal(data, &keys)
	return keys, err
}

// WaitForRemote waits until the remote has finished its queued writes or
// until the specified timeout occurs.
// Note: This waits for both itself and for the transaction log, so it can
// take up to 2*timeout for this function to close in the worst case.
func (r *internalKV) WaitForRemote(timeout time.Duration) bool {
	//First, wait for my own open threads..
	t := time.NewTimer(timeout)
	done := false
	for !done {
		select {
		case <-time.After(time.Millisecond * 100):
			x := atomic.LoadInt32(&r.openRoutines)
			if x == 0 {
				done = true
			}
		case <-t.C:
			return false
		}
	}

	//Now wait for tx log
	return r.txLog.WaitForRemote(timeout)
}

// remoteSet is a utility function which will write the transaction to
// the KV.
func (r *internalKV) remoteSet(key string, val []byte,
	updateCb RemoteStoreCallback) error {

	if updateCb == nil {
		updateCb = r.defaultRemoteWriteCB
	}

	wrapper := func(newTx Transaction, err error) {
		r.handleRemoteSet(newTx, err, updateCb)
	}

	// Write the transaction
	newTx := NewTransaction(netTime.Now(), key, val)
	if err := r.txLog.Append(newTx, wrapper); err != nil {
		return err
	}

	// Return an error if we are no longer connected.
	if !r.connected {
		return errors.Errorf("disconnected from the remote KV")
	}

	// Report to event callback
	if r.KeyUpdate != nil {
		// Report write as successful
		r.KeyUpdate(key, nil, val, true)
	}

	return nil
}

// handleRemoteSet contains the logic for handling a remoteSet attempt. It will
// handle and modify state within the KV for failed remote sets.
func (r *internalKV) handleRemoteSet(newTx Transaction, err error,
	updateCb RemoteStoreCallback) {

	// Pass context to user-defined callback, so they may handle failure for
	// remote saving
	if updateCb != nil {
		updateCb(newTx, err)
	}

	// Handle error
	if err != nil {
		jww.DEBUG.Printf("Failed to write new transaction (%v) to  remoteKV: %+v",
			newTx, err)

		r.connected = false
		go func() {
			time.Sleep(updateFailureDelay)
			r.txLog.Append(newTx, updateCb)
		}()

		return
	}

	r.lck.Lock()
	defer r.lck.Unlock()
	err = r.removeUnsyncedWrite(newTx.Key)
	if err != nil {
		jww.WARN.Printf("Failed to remove intent for key %s: %+v",
			newTx.Key, err)
	}

}

// addUnsyncedWrite will write the intent to the map. This map will be saved to disk
// using te kv.
func (r *internalKV) addUnsyncedWrite(key string, val []byte) error {
	r.UnsyncedWrites[key] = val
	return r.saveUnsyncedWrites()
}

// removeUnsyncedWrite will delete the intent from the map. This modified map will be
// saved to disk using the kv.
func (r *internalKV) removeUnsyncedWrite(key string) error {
	delete(r.UnsyncedWrites, key)
	return r.saveUnsyncedWrites()
}

// saveUnsyncedWrites is a utility function which writes the UnsyncedWrites map to disk.
func (r *internalKV) saveUnsyncedWrites() error {
	//fmt.Printf("unsynced: %v\n", r.UnsyncedWrites)
	data, err := json.Marshal(r.UnsyncedWrites)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   intentsVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return r.local.Set(intentsKey, obj)
}

// loadUnsyncedWrites will load any intents from kv if present and set it into
// UnsyncedWrites.
func (r *internalKV) loadUnsyncedWrites() error {
	// NOTE: obj is a versioned.Object, but we are not using a
	// versioned KV because we are implementing the uncoupled,
	// base KV interface, so you will need to update and check old
	// keys if we ever change this format.
	var obj versioned.Object
	err := r.local.Get(intentsKey, &obj)
	if err != nil { // Return if there isn't any intents stored
		return nil
	}

	return json.Unmarshal(obj.Data, &r.UnsyncedWrites)
}

// isRemote checks for the presence and accurate setting of the
// isRemoteKey inside the kv.
func isRemote(kv ekv.KeyValue) bool {
	val, err := kv.GetBytes(isRemoteKey)
	if err != nil && ekv.Exists(err) {
		jww.WARN.Printf("error checking if kv is remote: %+v", err)
		return false
	}
	if val != nil && bytes.Equal(val, kvIsRemoteVal) {
		return true
	}
	return false
}

// setRemote sets the kvIsRemoteVal for isRemoteKey, making isRemote turn
// true in the future.
func setRemote(kv ekv.KeyValue) {
	err := kv.SetBytes(isRemoteKey, kvIsRemoteVal)
	if err != nil {
		// we can't proceed if remote can't be set on the local kv
		jww.FATAL.Panicf("couldn't set remote: %+v", err)
	}
}
