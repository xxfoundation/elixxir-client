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
	"io"
	"sync"
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
	mapKeysListFmt   = "%s_Keys"
	mapElementKeyFmt = "%s_element_%s"

	// KeyUpdateCallback statuses.
	Disconnected = "Disconnected"
	Connected    = "Connected"
	Successful   = "UpdatedKey"
)

// updateFailureDelay is the backoff period in between retrying to
const updateFailureDelay = 1 * time.Second

// KV implements a remote KV to handle transaction logs.
type KV struct {
	// local is the local EKV store that will write the transaction.
	local ekv.KeyValue

	// txLog is the transaction log used to write transactions.
	txLog *TransactionLog

	// KeyUpdate is the callback used to report events when
	// attempting to call Set.
	KeyUpdate KeyUpdateCallback

	// list of tracked keys
	tracked []string

	// UnsyncedWrites is the pending writes that we are waiting
	// for on remote storage. Anytime this is not empty, we are
	// not synchronized and this should be reported.
	UnsyncedWrites map[string][]byte

	// synchronizedPrefixes are prefixes that trigger remote
	// synchronization calls.
	synchronizedPrefixes []string

	// defaulteUpdateCB is called when the updateCB is not specified for
	// remote store and set operations
	defaultUpdateCB RemoteStoreCallback

	// Connected determines the connectivity of the remote server.
	connected bool

	// lck is used to lck access to Remote, if no remote needed,
	// then the underlying kv lock is all that is needed.
	lck sync.RWMutex

	// mapLck prevents asynchronous map operations
	mapLck sync.Mutex
}

// NewOrLoadKV constructs a new KV. If data exists on disk, it loads
// that context and handle it appropriately.
func NewOrLoadKV(transactionLog *TransactionLog, kv ekv.KeyValue,
	synchedPrefixes []string,
	eventCb KeyUpdateCallback,
	updateCb RemoteStoreCallback) (*KV, error) {

	sPrefixes := synchedPrefixes
	if sPrefixes == nil {
		sPrefixes = make([]string, 0)
	}

	rkv := &KV{
		local:                kv,
		txLog:                transactionLog,
		KeyUpdate:            eventCb,
		UnsyncedWrites:       make(map[string][]byte, 0),
		synchronizedPrefixes: sPrefixes,
		defaultUpdateCB:      updateCb,
		connected:            true,
	}

	if err := rkv.loadUnsyncedWrites(); err != nil {
		return nil, err
	}

	// Re-trigger all lingering intents
	rkv.lck.Lock()
	for key, val := range rkv.UnsyncedWrites {
		// Call the internal to avoid writing to intent what is already there
		go rkv.remoteSet(key, val, updateCb)
	}
	rkv.lck.Unlock()

	return rkv, nil
}

// LocalKV Loads or Creates a synchronized remote KV that uses a local-only
// transaction log. It panics if the underlying KV has ever been used
// for remote operations in the past.
func LocalKV(path string, deviceSecret []byte, filesystem FileIO, kv ekv.KeyValue,
	synchedPrefixes []string,
	eventCb KeyUpdateCallback,
	updateCb RemoteStoreCallback, rng io.Reader) (*KV, error) {
	if isRemote(kv) {
		jww.FATAL.Panicf("cannot open remote kv as local")
	}
	txLog, err := NewLocalTransactionLog(path, filesystem, deviceSecret,
		rng)
	if err != nil {
		return nil, err
	}
	return NewOrLoadKV(txLog, kv, synchedPrefixes, eventCb, updateCb)
}

///////////////////////////////////////////////////////////////////////////////
// Begin KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Set implements [ekv.KeyValue.Set]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
func (r *KV) Set(key string, objectToStore ekv.Marshaler) error {
	return r.SetBytes(key, objectToStore.Marshal())
}

// Get implements [ekv.KeyValue.Get]
func (r *KV) Get(key string, loadIntoThisObject ekv.Unmarshaler) error {
	data, err := r.GetBytes(key)
	if err != nil {
		return err
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Delete implements [ekv.KeyValue.Delete]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud
func (r *KV) Delete(key string) error {
	return r.local.Delete(key)
}

// SetInterface implements [ekv.KeyValue.SetInterface]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
func (r *KV) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err != nil {
		return err
	}
	return r.SetBytes(key, data)
}

// GetInterface implements [ekv.KeyValue.GetInterface]
func (r *KV) GetInterface(key string, objectToLoad interface{}) error {
	data, err := r.GetBytes(key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, objectToLoad)
}

// SetBytes implements [ekv.KeyValue.SetBytes]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
func (r *KV) SetBytes(key string, data []byte) error {
	return r.local.SetBytes(key, data)
}

// GetBytes implements [ekv.KeyValue.GetBytes]
func (r *KV) GetBytes(key string) ([]byte, error) {
	return r.local.GetBytes(key)
}

///////////////////////////////////////////////////////////////////////////////
// End KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// UpsertLocal performs an upsert operation and sets the resultant
// value to the local EKV. It is a LOCAL ONLY operation which will
// write the Transaction to local store.
// todo: test this
func (r *KV) UpsertLocal(key string, newVal []byte) error {
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
func (r *KV) SetRemote(key string, val []byte,
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
func (r *KV) SetRemoteOnly(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	return r.remoteSet(key, val, updateCb)
}

// StoreMapElement saves a given map element and updates
// the map keys list if it is a new key.
// All Map storage functions update the remote.
func (r *KV) StoreMapElement(mapName, elementKey string, value []byte) error {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()
	return r.storeMapElement(mapName, elementKey, value)
}

// keep this private method here because it is the logic of StoreMapElement
// without the lock.
func (r *KV) storeMapElement(mapName, elementKey string, value []byte) error {
	// Store the element
	key := fmt.Sprintf(mapElementKeyFmt, mapName, elementKey)
	err := r.SetRemote(key, value, nil)
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
		r.storeMapKeys(mapName, existingKeys)
	}

	return nil
}

// StoreMap saves each element of the map, then updates the map structure
// and deletes no longer used keys in the map.
// All Map storage functions update the remote.
func (r *KV) StoreMap(mapName string, value map[string][]byte) error {
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
		err := r.storeMapElement(mapName, k, v)
		if err != nil {
			return err
		}
	}
	err = r.storeMapKeys(mapName, newKeys)
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
func (r *KV) GetMapElement(mapName, elementKey string) ([]byte, error) {
	r.mapLck.Lock()
	defer r.mapLck.Unlock()
	key := fmt.Sprintf(mapElementKeyFmt, mapName, elementKey)
	return r.GetBytes(key)
}

// GetMap returns all values inside a map
func (r *KV) GetMap(mapName string) (map[string][]byte, error) {
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

func (r *KV) storeMapKeys(mapName string, keys map[string]struct{}) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	key := fmt.Sprintf(mapKeysListFmt, mapName)
	return r.SetRemote(key, data, nil)
}

func (r *KV) getMapKeys(mapName string) (map[string]struct{}, error) {
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
func (r *KV) WaitForRemote(timeout time.Duration) bool {
	return r.txLog.WaitForRemote(timeout)
}

// remoteSet is a utility function which will write the transaction to
// the KV.
func (r *KV) remoteSet(key string, val []byte,
	updateCb RemoteStoreCallback) error {

	if updateCb == nil {
		updateCb = r.defaultUpdateCB
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
func (r *KV) handleRemoteSet(newTx Transaction, err error,
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

		// Report to event callback
		if r.KeyUpdate != nil {
			r.KeyUpdate(newTx.Key, nil, newTx.Value, false)
		}

		r.connected = false
		// fixme: feels like more thought needs to be put. A recursive cb
		//  such as this seems like a poor idea. Maybe the callback is
		//  passed down, and it's the responsibility of the caller to ensure
		//  remote writing of the txLog?
		//time.Sleep(updateFailureDelay)
		//r.txLog.Append(newTx, updateCb)
		return
	} else if r.connected {
		// Report to event callback
		if r.KeyUpdate != nil {
			r.KeyUpdate(newTx.Key, nil, newTx.Value, true)
		}
	}

	r.lck.Lock()
	err = r.removeUnsyncedWrite(newTx.Key)
	if err != nil {
		jww.WARN.Printf("Failed to remove intent for key %s: %+v",
			newTx.Key, err)
	}
	r.lck.Unlock()

}

// addUnsyncedWrite will write the intent to the map. This map will be saved to disk
// using te kv.
func (r *KV) addUnsyncedWrite(key string, val []byte) error {
	r.UnsyncedWrites[key] = val
	return r.saveUnsyncedWrites()
}

// removeUnsyncedWrite will delete the intent from the map. This modified map will be
// saved to disk using the kv.
func (r *KV) removeUnsyncedWrite(key string) error {
	delete(r.UnsyncedWrites, key)
	return r.saveUnsyncedWrites()
}

// saveUnsyncedWrites is a utility function which writes the UnsyncedWrites map to disk.
func (r *KV) saveUnsyncedWrites() error {
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
func (r *KV) loadUnsyncedWrites() error {
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

// todo figure out details in next ticket
func isRemote(kv ekv.KeyValue) bool {
	return false
}
