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
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
	"sync"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// Remote KV Implementation
///////////////////////////////////////////////////////////////////////////////

// RemoteKV kv-related constants.
const (
	remoteKvPrefix  = "remoteKvPrefix"
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

// RemoteKV implements a remote KV to handle transaction logs.
type RemoteKV struct {
	// local is the versioned KV store that will write the transaction.
	local *versioned.KV

	// txLog is the transaction log used to write transactions.
	txLog *TransactionLog

	// KeyUpdate is the callback used to report events when attempting to call Set.
	KeyUpdate KeyUpdateCallback

	// list of tracked keys
	tracked []string

	// UnsyncedWrites is the pending writes that we are waiting for on remote
	// storage. Anytime this is not empty, we are not synchronized and this
	// should be reported.
	UnsyncedWrites map[string][]byte

	// Connected determines the connectivity of the remote server.
	connected bool

	lck sync.RWMutex
}

// NewOrLoadRemoteKV constructs a new RemoteKV. If data exists on disk, it loads
// that context and handle it appropriately.
func NewOrLoadRemoteKV(transactionLog *TransactionLog, kv *versioned.KV,
	eventCb KeyUpdateCallback,
	updateCb RemoteStoreCallback) (*RemoteKV, error) {

	rkv := &RemoteKV{
		local:          kv.Prefix(remoteKvPrefix),
		txLog:          transactionLog,
		KeyUpdate:      eventCb,
		UnsyncedWrites: make(map[string][]byte, 0),
		connected:      true,
	}

	if err := rkv.loadUnsyncedWrites(); err != nil {
		return nil, err
	}

	// Re-trigger all lingering intents
	for key, val := range rkv.UnsyncedWrites {
		// Call the internal to avoid writing to intent what is already there
		go rkv.remoteSet(key, val, updateCb)
	}

	return rkv, nil
}

// Get retrieves the data stored in the underlying kv. Will return an error
// if the data at this key cannot be retrieved.
func (r *RemoteKV) Get(key string) ([]byte, error) {
	r.lck.RLock()
	defer r.lck.RUnlock()

	// Read from local KV
	obj, err := r.local.Get(key, remoteKvVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// UpsertLocal is a LOCAL ONLY operation which will write the Transaction
// to local store.
// todo: test this
func (r *RemoteKV) UpsertLocal(key string, newVal []byte) error {
	// Read from local KV
	obj, err := r.local.Get(key, remoteKvVersion)
	if err != nil {
		// Error means key does not exist, simply write to local
		return r.localSet(key, newVal)
	}

	curVal := obj.Data
	if bytes.Equal(curVal, newVal) {
		jww.TRACE.Printf("Same value for transaction %+v", curVal)
		return nil
	}

	if r.KeyUpdate != nil {
		r.KeyUpdate(key, curVal, newVal, true)
	}

	return r.localSet(key, newVal)
}

// Set will write a transaction to the remote and local store.
func (r *RemoteKV) Set(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	r.lck.Lock()
	defer r.lck.Unlock()

	// Add intent to write transaction
	if err := r.addUnsyncedWrite(key, val); err != nil {
		return err
	}

	// Save locally
	if err := r.localSet(key, val); err != nil {
		return errors.Errorf("failed to write to local kv: %+v", err)
	}

	return r.remoteSet(key, val, updateCb)
}

// RemoteSet will place this Transaction onto the remote server. This is an
// asynchronous operation and results will be passed back via the
// RemoteStoreCallback.
//
// NO LOCAL STORAGE OPERATION WIL BE PERFORMED.
func (r *RemoteKV) RemoteSet(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	return r.remoteSet(key, val, updateCb)
}

// GetList is a wrapper of [LocalStore.GetList]. This will return a JSON
// marshalled [KeyValueMap].
func (r *RemoteKV) GetList(name string) ([]byte, error) {
	valList, err := r.txLog.local.GetList(name)
	if err != nil {
		return nil, err
	}

	return json.Marshal(valList)
}

// remoteSet is a utility function which will write the transaction to
// the RemoteKV.
func (r *RemoteKV) remoteSet(key string, val []byte,
	updateCb RemoteStoreCallback) error {

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
// handle and modify state within the RemoteKV for failed remote sets.
func (r *RemoteKV) handleRemoteSet(newTx Transaction, err error,
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

	err = r.removeUnsyncedWrite(newTx.Key)
	if err != nil {
		jww.WARN.Printf("Failed to remove intent for key %s: %+v",
			newTx.Key, err)
	}
}

// localSet will save the key value pair in the local KV.
func (r *RemoteKV) localSet(key string, val []byte) error {
	// Create versioned object for kv.Set
	obj := &versioned.Object{
		Version:   remoteKvVersion,
		Timestamp: netTime.Now(),
		Data:      val,
	}

	// Write value to KV
	return r.local.Set(key, obj)
}

// addUnsyncedWrite will write the intent to the map. This map will be saved to disk
// using te kv.
func (r *RemoteKV) addUnsyncedWrite(key string, val []byte) error {
	r.UnsyncedWrites[key] = val
	return r.saveUnsyncedWrites()
}

// removeUnsyncedWrite will delete the intent from the map. This modified map will be
// saved to disk using the kv.
func (r *RemoteKV) removeUnsyncedWrite(key string) error {
	delete(r.UnsyncedWrites, key)
	return r.saveUnsyncedWrites()
}

// saveUnsyncedWrites is a utility function which writes the UnsyncedWrites map to disk.
func (r *RemoteKV) saveUnsyncedWrites() error {
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
func (r *RemoteKV) loadUnsyncedWrites() error {
	obj, err := r.local.Get(intentsKey, intentsVersion)
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
