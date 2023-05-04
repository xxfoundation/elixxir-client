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
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
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
// and generally you should never access it on the versionedKV object. It is
// provided so that external xxdk libraries can access specific functionality.
// This is considered internal api and may be changed or removed at any time.
type RemoteKV interface {
	ekv.KeyValue

	// SetRemote will Write a mutate to the remote and write the key to the
	// local store.
	// Does not allow writing to keys with the suffix "_🗺️MapKeys",
	// it is reserved
	SetRemote(key string, value []byte) error

	// DeleteRemote will write a mutate to the remote with an instruction to delete
	// the key and will delete it locally.
	// Does not allow writing to keys with the suffix "_🗺️MapKeys",
	// it is reserved
	DeleteRemote(key string) error
}

// internalKV implements a remote internalKV to handle mutate logs.
type internalKV struct {
	// local is the local EKV store that will Write the mutate.
	local ekv.KeyValue

	// txLog is the mutate log used to Write transactions.
	txLog *remoteWriter

	// keyUpdateListeners holds callbacks called when a key is updated
	// by a remote
	UpdateListenerMux  sync.RWMutex
	keyUpdateListeners map[string]keyChangedByRemoteCallback
	mapUpdateListeners map[string]mapChangedByRemoteCallback
}

// newKV constructs a new remote KV. If data exists on disk, it loads
// that context and handle it appropriately.
func newKV(transactionLog *remoteWriter, kv ekv.KeyValue) *internalKV {

	rkv := &internalKV{
		local:              kv,
		txLog:              transactionLog,
		keyUpdateListeners: make(map[string]keyChangedByRemoteCallback),
		mapUpdateListeners: make(map[string]mapChangedByRemoteCallback),
	}

	return rkv
}

///////////////////////////////////////////////////////////////////////////////
// Begin KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Set implements [ekv.KeyValue.Set]. This is a LOCAL ONLY
// operation which will Write the Mutate to local store.
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
// operation which will Write the Mutate to local store.
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
// operation which will Write the Mutate to local store.
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
// operation which will Write the Mutate to local store.
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

	return r.local.Transaction(key, op)
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

	return r.local.MutualTransaction(keys, op)
}

// SetBytesFromRemote implements [ekv.KeyValue.SetBytes].
// This is a LOCAL ONLY operation which will Write the Mutate
// to local store. Only use this from the collector system, designed
// to allow event models to connect to the Write.
func (r *internalKV) SetBytesFromRemote(key string, data []byte) error {
	f := func(_ []byte, _ bool) ([]byte, bool, error) {
		return data, false, nil
	}

	old, existed, err := r.local.Transaction(key, f)

	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if cb, exists := r.keyUpdateListeners[key]; exists {
		op := versioned.Created
		if existed {
			op = versioned.Updated
		}
		go cb(key, old, data, op)
	}
	return nil
}

// MapTransactionFromRemote allows a map to be updated by the collector
// in a metallurgy locking manner
func (r *internalKV) MapTransactionFromRemote(mapName string,
	edits map[string]*Mutate) error {

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
		mapFile, err := getMapFile(old[mapKey], len(old)-1)
		if err != nil {
			return nil, err
		}

		// edit elements and update map file
		for elementKey, value := range edits {
			updates[elementKey] = ekv.Value{
				Data:   value.Value,
				Exists: !value.Deletion,
			}
			if value.Deletion {
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
	reportedEdits := make(map[string]elementEdit, len(edits))
	for elementKey := range edits {
		elementName := keysToName[elementKey]
		oldElement := old[elementKey]
		newElement := edits[elementKey]

		var mapOp versioned.KeyOperation
		if newElement.Deletion {
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

		reportedEdits[elementName] = elementEdit{
			OldElement: oldElement.Data,
			NewElement: newElement.Value,
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

func (r *internalKV) DeleteFromRemote(key string) error {
	f := func(_ []byte, _ bool) ([]byte, bool, error) {
		return nil, true, nil
	}

	old, _, err := r.local.Transaction(key, f)

	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if cb, exists := r.keyUpdateListeners[key]; exists {
		go cb(key, old, nil, versioned.Deleted)
	}
	return nil
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
func (r *internalKV) ListenOnRemoteKey(key string, callback keyChangedByRemoteCallback) {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()
	r.keyUpdateListeners[key] = callback
}

type keyChangedByRemoteCallback func(key string, old, new []byte, op versioned.KeyOperation)

// ListenOnRemoteMap allows the caller to receive updates when
// any element in the given map is updated by synching with another client.
// Only one callback can be written per key.
func (r *internalKV) ListenOnRemoteMap(mapName string, callback mapChangedByRemoteCallback) {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()
	r.mapUpdateListeners[mapName] = callback
}

type mapChangedByRemoteCallback func(mapName string, edits map[string]elementEdit)
type elementEdit struct {
	OldElement []byte
	NewElement []byte
	Operation  versioned.KeyOperation
}

///////////////////////////////////////////////////////////////////////////////
// End KV [ekv.KeyValue] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// SetRemote will Write a mutate to the remote and write the key to the
// local store.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetRemote(key string, value []byte) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	return r.txLog.Write(key, value)
}

// DeleteRemote will write a mutate to the remote with an instruction to delete
// the key and will delete it locally.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) DeleteRemote(key string) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	return r.txLog.Delete(key)
}

// setIsRemoteKV sets the kvIsRemoteVal for isRemoteKey, making isRemote turn
// true in the future.
func enableRemoteKV(kv ekv.KeyValue) {
	err := kv.SetBytes(isRemoteKey, kvIsRemoteVal)
	if err != nil {
		// we can't proceed if remote can't be set on the local kv
		jww.FATAL.Panicf("couldn't set remote: %+v", err)
	}
}

// isRemote checks for the presence and accurate setting of the
// isRemoteKey inside the kv.
func isRemoteKV(kv ekv.KeyValue) bool {
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
