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

	// keyUpdateListeners holds callbacks called when a key is updated
	// by a remote
	keyUpdateListeners map[string]versioned.KeyChangedByRemoteCallback
	mapUpdateListeners map[string]versioned.MapChangedByRemoteCallback

	// keyUpdate is the callback used to report events when
	// attempting to call Set.
	keyUpdate KeyUpdateCallback

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
		keyUpdate:            eventCb,
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
	if err := versioned.IsValidKey(key); err != nil {
		return err
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
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	return r.local.Delete(key)
}

// SetInterface implements [ekv.KeyValue.SetInterface]. This is a LOCAL ONLY
// operation which will write the Transaction to local store.
// Use [SetRemote] to set keys synchronized to the cloud.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetInterface(key string, objectToStore interface{}) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
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
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	return r.local.SetBytes(key, data)
}

// GetBytes implements [ekv.KeyValue.GetBytes]
func (r *internalKV) GetBytes(key string) ([]byte, error) {
	return r.local.GetBytes(key)
}

// Transaction locks a key while it is being mutated then stores the result
// and returns the old value if it existed.
// If the op returns an error, the operation will be aborted.
func (r *internalKV) Transaction(key string, op ekv.TransactionOperation) (
	old []byte, existed bool, err error) {
	if err = versioned.IsValidKey(key); err != nil {
		return nil, false, err
	}

	return r.transactionUnsafe(key, op)
}

// MutualTransaction locks all keys while operating, getting the initial values
// for all keys, passing them into the MutualTransactionOperation, writing
// the resulting values for all keys to disk, and returns the initial value
// the return value is the same as is sent to the op, if it is edited they
// will reflect in the returned old dataset
func (r *internalKV) MutualTransaction(keys []string, op ekv.MutualTransactionOperation) (
	old, written map[string]ekv.Value, err error) {
	for _, k := range keys {
		if err = versioned.IsValidKey(k); err != nil {
			return nil, nil, errors.WithMessagef(err,
				"Failed to execute due to malformed key %s", k)
		}
	}

	return r.mutualTransactionUnsafe(keys, op)
}

func (r *internalKV) mutualTransactionUnsafe(keys []string, op ekv.MutualTransactionOperation) (
	old, written map[string]ekv.Value, err error) {
	return r.local.MutualTransaction(keys, op)
}

func (r *internalKV) transactionUnsafe(key string, op ekv.TransactionOperation) (
	old []byte, existed bool, err error) {
	return r.local.Transaction(key, op)
}

// SetBytesFromRemote implements [ekv.KeyValue.SetBytes].
// This is a LOCAL ONLY operation which will write the Transaction
// to local store. Only use this from the collector system, designed
// to allow event models to connect to the write.
func (r *internalKV) SetBytesFromRemote(key string, data []byte) error {
	f := func(_ []byte, _ bool) ([]byte, error) {
		return data, nil
	}

	old, existed, err := r.transactionUnsafe(key, f)

	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if cb, exists := r.keyUpdateListeners[key]; exists {
		go cb(key, old, data, existed)
	}
	return nil
}

// TransactionFromRemote locks a key while it is being mutated then stores the result
// and returns the old value if it existed.
// If the op returns an error, the operation will be aborted.
func (r *internalKV) TransactionFromRemote(key string,
	op ekv.TransactionOperation) error {
	var data []byte

	wrap := func(old []byte, existed bool) ([]byte, error) {
		var err error
		data, err = op(old, existed)
		return data, err
	}

	old, existed, err := r.transactionUnsafe(key, wrap)
	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if cb, exists := r.keyUpdateListeners[key]; exists {
		go cb(key, old, data, existed)
	}

	return nil
}

var noUpdateNeeded = errors.New("no map file update is needed")

// MapTransactionFromRemote
func (r *internalKV) MapTransactionFromRemote(mapName string,
	edits map[string]ekv.Value) error {

	// add element keys
	keys := make([]string, 0, len(edits)+1)
	keysToName := make(map[string]string, len(edits))
	for elementKey := range edits {
		keys = append(keys, elementKey)
		_, elementName := versioned.GetElementName(elementKey)
		keysToName[elementKey] = elementName
	}

	//add the map key list
	mapKey := versioned.MakeMapKey(mapName)
	keys = append(keys, mapKey)

	op := func(old map[string]ekv.Value) (map[string]ekv.Value, error) {
		updates := make(map[string]ekv.Value, len(old))

		//process key map, will always be the last value due to it being
		mapFile := newSet(uint(len(old) - 1))
		mapFileValue := old[mapKey]
		if mapFileValue.Exists {
			err := mapFile.UnmarshalJSON(mapFileValue.Data)
			if err != nil {
				return nil, err
			}
		}

		// edit elements and update map file
		for elementKey, value := range edits {
			updates[elementKey] = value
			if value.Exists {
				mapFile.Add(keysToName[elementKey])
			} else {
				mapFile.Delete(keysToName[elementKey])
			}
		}

		// add the map file to updates
		mapFileUpdate, err := mapFile.MarshalJSON()
		if err != nil {
			return nil, err
		}

		updates[mapKey] = ekv.Value{
			Data:   mapFileUpdate,
			Exists: true,
		}

		return updates, nil
	}

	//run the operation on the ekv
	old, _, err := r.local.MutualTransaction(keys, op)
	if err != nil {
		return err
	}

	//build the return data, ignore deletion failures
	reportedEdits := make(map[string]versioned.ElementEdit, len(edits))
	for elementKey := range edits {
		elementName := keysToName[elementKey]
		oldElement := old[elementKey]
		newElement := edits[elementKey]

		var mapOp versioned.MapOperation
		if !newElement.Exists {
			if !oldElement.Exists {
				// if the element was already deleted, dont report the operation
				continue
			}
			mapOp = versioned.Deleted
		} else {
			if oldElement.Exists {
				mapOp = versioned.Updated
			} else {
				mapOp = versioned.Created
			}
		}

		reportedEdits[elementName] = versioned.ElementEdit{
			OldElement: oldElement.Data,
			NewElement: oldElement.Data,
			Operation:  mapOp,
		}
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()

	if cb, exists := r.mapUpdateListeners[mapName]; exists {
		go cb(mapName, reportedEdits)
	}

	return nil
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *internalKV) ListenOnRemoteKey(key string, callback versioned.KeyChangedByRemoteCallback) {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()
	r.keyUpdateListeners[key] = callback
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

	if r.keyUpdate != nil {
		r.keyUpdate(key, obj, newVal, true)
	}

	return r.local.SetBytes(key, newVal)
}

// SetRemote will write a transaction to the remote and local store
// with the specified RemoteCB RemoteStoreCallback
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetRemote(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
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
	if err := r.local.SetBytes(key, val); err != nil {
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

/

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
	if r.keyUpdate != nil {
		// Report write as successful
		r.keyUpdate(key, nil, val, true)
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
