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
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
)

///////////////////////////////////////////////////////////////////////////////
// KV Implementation
///////////////////////////////////////////////////////////////////////////////

// KV kv-related constants.
const (
	remoteKvVersion = 0

	intentsVersion = 0
	intentsKey     = "intentsVersion"
)

// KeyUpdateCallback statuses.
const (
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

	lck sync.RWMutex
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

// GetList is a wrapper of [LocalStore.GetList]. This will return a JSON
// marshalled [KeyValueMap].
func (r *KV) GetList(name string) ([]byte, error) {
	valList, err := r.txLog.local.GetList(name)
	if err != nil {
		return nil, err
	}

	return json.Marshal(valList)
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

///////////////////////////////////////////////////////////////////////////////
// Remote File System Implementation
///////////////////////////////////////////////////////////////////////////////

// FileSystemRemoteStorage is a structure adhering to [RemoteStore]. This
// utilizes the [os.File] IO operations. Implemented for testing purposes for
// transaction logs.
type FileSystemRemoteStorage struct {
	baseDir string
}

// NewFileSystemRemoteStorage is a constructor for FileSystemRemoteStorage.
//
// Arguments:
//   - baseDir - string. Represents the base directory for which all file
//     operations will be performed. Must contain a file delimiter (i.e. `/`).
func NewFileSystemRemoteStorage(baseDir string) *FileSystemRemoteStorage {
	return &FileSystemRemoteStorage{
		baseDir: baseDir,
	}
}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes utils.ReadFile under the hood.
func (f *FileSystemRemoteStorage) Read(path string) ([]byte, error) {
	if utils.DirExists(path) {
		return utils.ReadFile(f.baseDir + path)
	}
	return utils.ReadFile(path)
}

// Write will write data to path. This will return an error if it fails to
// write.
//
// This utilizes utils.WriteFileDef under the hood.
func (f *FileSystemRemoteStorage) Write(path string, data []byte) error {
	if utils.DirExists(path) {
		return utils.WriteFileDef(f.baseDir+path, data)
	}
	return utils.WriteFileDef(path, data)

}

// GetLastModified will return the last modified timestamp of the file at path.
// It will return an error if it cannot retrieve any os.FileInfo from the file
// path.
//
// This utilizes utils.GetLastModified under the hood.
func (f *FileSystemRemoteStorage) GetLastModified(path string) (
	time.Time, error) {
	if utils.DirExists(path) {
		return utils.GetLastModified(f.baseDir + path)
	}
	return utils.GetLastModified(path)
}

// GetLastWrite will retrieve the most recent successful write operation
// that was received by RemoteStore.
func (f *FileSystemRemoteStorage) GetLastWrite() (time.Time, error) {
	return utils.GetLastModified(f.baseDir)
}
