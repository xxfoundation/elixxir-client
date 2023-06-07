////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"bytes"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/stoppable"
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
	// kv is the local EKV store that will Write the mutate.
	kv ekv.KeyValue

	// txLog is the mutate log used to Write transactions.
	txLog *remoteWriter
	col   *collector

	// is the synchronization thread active?
	isSynchronizing *atomic.Bool
	mux             sync.Mutex

	// keyUpdateListeners holds callbacks called when a key is updated
	// by a remote
	UpdateListenerMux  sync.RWMutex
	keyUpdateListeners map[string]keyUpdate
	mapUpdateListeners map[string]mapUpdate
}

// newKV constructs a new remote KV. If data exists on disk, it loads
// that context and handle it appropriately.
func newKV(transactionLog *remoteWriter, kv ekv.KeyValue) *internalKV {

	isSync := atomic.Bool{}
	isSync.Store(false)

	rkv := &internalKV{
		kv:                 kv,
		txLog:              transactionLog,
		keyUpdateListeners: make(map[string]keyUpdate),
		mapUpdateListeners: make(map[string]mapUpdate),
		isSynchronizing:    &isSync,
	}

	// Panic if an instance ID doesn't exist
	instanceID, err := GetInstanceID(kv)
	if err != nil {
		jww.FATAL.Panicf("[COLLECTIVE] kv load fail: %+v", err)
	}

	jww.INFO.Printf("[COLLECTIVE] kv loaded: %s", instanceID)

	return rkv
}

///////////////////////////////////////////////////////////////////////////////
// KV [ekv.KeyValue] interface implementation functions
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
	if updater, exists := r.keyUpdateListeners[key]; exists && updater.local {
		var old []byte
		var existed bool
		op := func(files map[string]ekv.Operable, _ ekv.Extender) error {
			file := files[key]
			old, existed = file.Get()
			file.Delete()
			return file.Flush()
		}

		if err := r.kv.Transaction(op, key); err != nil {
			return err
		}

		if existed {
			go updater.cb(old, nil, versioned.Deleted)
		}

		return nil
	} else {
		return r.kv.Delete(key)
	}
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
	if updater, exists := r.keyUpdateListeners[key]; exists && updater.local {
		var old []byte
		var existed bool
		op := func(files map[string]ekv.Operable, _ ekv.Extender) error {
			file := files[key]
			old, existed = file.Get()
			file.Set(data)
			return file.Flush()
		}

		if err := r.kv.Transaction(op, key); err != nil {
			return err
		}

		operation := versioned.Created
		if existed {
			operation = versioned.Updated
		}
		go updater.cb(old, data, operation)
		return nil
	} else {
		return r.kv.SetBytes(key, data)
	}
}

// GetBytes implements [ekv.KeyValue.GetBytes]
func (r *internalKV) GetBytes(key string) ([]byte, error) {
	return r.kv.GetBytes(key)
}

// Transaction implements [ekv.KeyValue.Transaction]
// does not filter for invalid keys
// does not call event callbacks if they are set locally
func (r *internalKV) Transaction(op ekv.TransactionOperation, keys ...string) error {
	return r.kv.Transaction(op, keys...)
}

///////////////////////////////////////////////////////////////////////////////
// Collector API - collector uses these to update local
///////////////////////////////////////////////////////////////////////////////

// SetBytesFromRemote implements [ekv.KeyValue.SetBytes].
// This is a LOCAL ONLY operation which will Write the Mutate
// to local store. Only use this from the collector system, designed
// to allow event models to connect to the Write.
func (r *internalKV) SetBytesFromRemote(key string, data []byte) error {
	var old []byte
	var existed bool

	op := func(files map[string]ekv.Operable, _ ekv.Extender) error {
		file := files[key]
		old, existed = file.Get()
		file.Set(data)
		return file.Flush()
	}

	err := r.kv.Transaction(op, key)

	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if updater, exists := r.keyUpdateListeners[key]; exists {
		opp := versioned.Created
		if existed {
			opp = versioned.Updated
		}
		go updater.cb(old, data, opp)
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

	// build map in which the updates will be reported
	reportedEdits := make(map[string]elementEdit, len(edits))

	// construct the operation to do on the ekv
	op := func(files map[string]ekv.Operable, _ ekv.Extender) error {

		//process key map, will always be the last value due to it being
		mapFile := files[mapKey]
		mapFileBytes, _ := mapFile.Get()
		mapSet, err := getMapFile(mapFileBytes, len(edits))
		if err != nil {
			return err
		}

		// edit elements and update map file while building the update
		// structure
		for elementKey, value := range edits {

			// get the data about the element
			elementName := keysToName[elementKey]
			file := files[elementKey]
			old, exists := file.Get()

			element := elementEdit{OldElement: old}

			//handle the operation
			if value.Deletion {
				mapSet.Delete(elementName)
				file.Delete()
				element.Operation = versioned.Deleted
			} else {
				mapSet.Add(elementName)
				file.Set(value.Value)
				element.NewElement = value.Value
				if exists {
					element.Operation = versioned.Updated
				} else {
					element.Operation = versioned.Created
				}
			}

			// store the description of the operation in the report map
			reportedEdits[elementName] = element
		}

		// add the map file to updates
		mapFileUpdate, err := json.Marshal(mapSet)
		if err != nil {
			return err
		}
		mapFile.Set(mapFileUpdate)

		return nil
	}

	// run the operation on the ekv
	err := r.kv.Transaction(op, keys...)
	if err != nil {
		return err
	}

	// Call callbacks reporting on the operation
	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()

	if updater, exists := r.mapUpdateListeners[mapName]; exists {
		go updater.cb(reportedEdits)
	}

	return nil
}

func (r *internalKV) DeleteFromRemote(key string) error {
	var old []byte

	op := func(files map[string]ekv.Operable, _ ekv.Extender) error {
		file := files[key]
		old, _ = file.Get()
		file.Delete()
		return file.Flush()
	}

	err := r.kv.Transaction(op, key)

	if err != nil {
		return err
	}

	r.UpdateListenerMux.RLock()
	defer r.UpdateListenerMux.RUnlock()
	if updater, exists := r.keyUpdateListeners[key]; exists {
		go updater.cb(old, nil, versioned.Deleted)
	}
	return nil
}

// ListenOnRemoteKey allows the caller to receive updates when
// a key is updated by synching with another client.
// Only one callback can be written per key.
// NOTE: It may make more sense to listen for updates via the collector directly
func (r *internalKV) ListenOnRemoteKey(key string, cb keyUpdateCallback,
	localEvents bool) error {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()

	r.keyUpdateListeners[key] = keyUpdate{
		cb:    cb,
		local: localEvents,
	}
	curData, err := r.GetBytes(key)
	if err != nil && ekv.Exists(err) {
		return err
	}
	if curData != nil {
		cb(nil, curData, versioned.Loaded)
	}
	return nil
}

// keyUpdateCallback is the callback used to report the event.
type keyUpdateCallback func(oldVal, newVal []byte,
	op versioned.KeyOperation)

type keyUpdate struct {
	cb    keyUpdateCallback
	local bool
}

// ListenOnRemoteMap allows the caller to receive updates when
// any element in the given map is updated by synching with another client.
// Only one callback can be written per key.
// NOTE: It may make more sense to listen for updates via the collector directly
func (r *internalKV) ListenOnRemoteMap(mapName string,
	cb mapChangedByRemoteCallback, localEvents bool) error {
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()

	r.mapUpdateListeners[mapName] = mapUpdate{
		cb:    cb,
		local: localEvents,
	}
	curMap, err := r.GetMap(mapName)
	if err != nil && ekv.Exists(err) {
		return err
	}

	if len(curMap) > 0 {
		ee := make(map[string]elementEdit, len(curMap))
		for key := range curMap {
			ee[key] = elementEdit{
				OldElement: nil,
				NewElement: curMap[key],
				Operation:  versioned.Loaded,
			}
		}
		cb(ee)
	}

	return nil
}

type mapChangedByRemoteCallback func(edits map[string]elementEdit)
type elementEdit struct {
	OldElement []byte
	NewElement []byte
	Operation  versioned.KeyOperation
}

type mapUpdate struct {
	cb    mapChangedByRemoteCallback
	local bool
}

///////////////////////////////////////////////////////////////////////////////
// Remote functions -- writers to the txLog
///////////////////////////////////////////////////////////////////////////////

// SetRemote will Write a mutate to the remote and write the key to the
// local store.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) SetRemote(key string, value []byte) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	_, _, err := r.txLog.Write(key, value)
	return err
}

// DeleteRemote will write a mutate to the remote with an instruction to delete
// the key and will delete it locally.
// Does not allow writing to keys with the suffix "_🗺️MapKeys",
// it is reserved
func (r *internalKV) DeleteRemote(key string) error {
	if err := versioned.IsValidKey(key); err != nil {
		return err
	}
	_, _, err := r.txLog.Delete(key)
	return err
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

///////////////////////////////////////////////////////////////////////////////
// Collector Wrappers -- FIXME, this should probably be put elsewhere on stack
// but we don't have a good design solution for it yet.
///////////////////////////////////////////////////////////////////////////////

func (r *internalKV) IsSynchronizing() bool {
	return r.isSynchronizing.Load()
}

func (r *internalKV) StartProcesses() (stoppable.Stoppable, error) {

	// Lock up while we start to prevent Listen functions from overlapping
	// with this function.
	r.UpdateListenerMux.Lock()
	defer r.UpdateListenerMux.Unlock()

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
		s.ToStopped()
		// We explicitly do not do the following to prevent users
		// from listening to keys after networking starts
		// r.isSynchronizing.Store(false)
	}(myStopper)
	multiStoppable.Add(myStopper)

	return multiStoppable, nil
}

func (r *internalKV) RegisterConnectionTracker(nc NotifyCallback) {
	r.col.Register(nc)
	go nc(r.col.IsConnected())
}

func (r *internalKV) IsConnected() bool {
	return r.col.IsConnected()
}

func (r *internalKV) IsSynched() bool {
	return r.col.IsSynched()
}

// WaitForRemote block until timeout or remote operations complete
func (r *internalKV) WaitForRemote(timeout time.Duration) bool {
	// FIXME: txLog needs to wait here as well!
	return r.col.WaitUntilSynched(timeout)
}
