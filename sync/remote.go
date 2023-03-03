////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"fmt"
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

// EventCallback statuses.
const (
	Disconnected = "Disconnected"
	Connected    = "Connected"
	Successful   = "UpdatedKey"
)

// updateFailureDelay is the backoff period in between retrying to
const updateFailureDelay = 1 * time.Second

// UpsertCallback will alert the caller when key has been updated.
type UpsertCallback func(key string, old []byte, new []byte) ([]byte, error)

// EventCallback is the callback used to report the event.
type EventCallback func(k, v string)

// RemoteKV implements a remote KV to handle transaction logs.
type RemoteKV struct {
	// kv is the versioned KV store that will write the transaction.
	kv *versioned.KV

	// txLog is the transaction log used to write transactions.
	txLog *TransactionLog

	// Map of upserts to upsert call backs
	upserts map[string]UpsertCallback

	// Event is the callback used to report events when attempting to call Set.
	Event EventCallback

	// list of tracked keys
	// fixme: remove? seems like this is handled by upserts, unless I'm
	//  missing something?
	tracked []string

	// Intents is the current intents that we are waiting for on remote storage.
	// Anytime this is not empty, we are not synchronized. Report that.
	Intents map[string][]byte

	// Connected determines the connectivity of the remote server.
	connected bool

	lck sync.RWMutex
}

// NewOrLoadRemoteKv will construct a new RemoteKV. If data exists on disk, it
// will load that context and handle it appropriately.
func NewOrLoadRemoteKv(transactionLog *TransactionLog, kv *versioned.KV,
	upsertsCb map[string]UpsertCallback,
	eventCb EventCallback, updateCb RemoteStoreCallback) (*RemoteKV, error) {

	// Nil check upsert map
	if upsertsCb == nil {
		upsertsCb = make(map[string]UpsertCallback, 0)
	}

	rkv := &RemoteKV{
		kv:        kv.Prefix(remoteKvPrefix),
		txLog:     transactionLog,
		upserts:   upsertsCb,
		Event:     eventCb,
		Intents:   make(map[string][]byte, 0),
		connected: true,
	}

	if err := rkv.loadIntents(); err != nil {
		return nil, err
	}

	// Re-trigger all lingering intents
	for key, val := range rkv.Intents {
		// fixme: probably want to make a worker pool or async here?
		//  the issue is to handle the error
		// Call the internal to avoid writing to intent what is already there
		go rkv.set(key, val, updateCb)
	}

	return rkv, nil
}

// AddUpsertCallback will add an upsert callback with the given key.
func (r *RemoteKV) AddUpsertCallback(key string, cb UpsertCallback) {
	r.upserts[key] = cb
}

// Get retrieves the data stored in the underlying kv. Will return an error
// if the data at this key cannot be retrieved.
func (r *RemoteKV) Get(key string) ([]byte, error) {
	r.lck.RLock()
	defer r.lck.RUnlock()

	obj, err := r.kv.Get(key, remoteKvVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// Set will write a transaction to the transaction log.
func (r *RemoteKV) Set(key string, val []byte,
	updateCb RemoteStoreCallback) error {
	r.lck.Lock()
	defer r.lck.Unlock()

	// Add intent to write transaction
	if err := r.addIntent(key, val); err != nil {
		return err
	}

	return r.set(key, val, updateCb)
}

// set is a utility function which will write the transaction to the RemoteKV.
func (r *RemoteKV) set(key string, val []byte,
	updateCb RemoteStoreCallback) error {

	var old []byte

	// If an upsert callback exists for the given key, take note of the data
	// prior to setting
	if _, exists := r.upserts[key]; exists {
		old, _ = r.Get(key)
	}

	// Create versioned object for kv.Set
	obj := &versioned.Object{
		Version:   remoteKvVersion,
		Timestamp: netTime.Now(),
		Data:      val,
	}

	// Write value to KV
	if err := r.kv.Set(key, obj); err != nil {
		return errors.Errorf("failed to write to kv: %+v", err)
	}

	// Instantiate the remote store callback
	wrapper := func(newTx Transaction, err error) {

		// Pass context to user-defined callback so they may handle failure for
		// remote saving
		if updateCb != nil {
			updateCb(newTx, err)
		}

		// Handle error
		if err != nil {
			jww.DEBUG.Printf("Failed to write transaction new transaction (%v) to  remoteKV: %+v", newTx, err)

			// Report to event callback
			if r.Event != nil {
				r.Event(Disconnected, fmt.Sprintf("%v", err))
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
			if r.Event != nil {
				r.Event(Connected, "True")
			}
		}

		err = r.removeIntent(key)
		if err != nil {
			jww.WARN.Printf("Failed to remove intent for key %s: %+v", key, err)
		}
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
	if r.Event != nil {
		// Report write as successful
		r.Event(Successful, key)
	}

	// If an update callback exists for this key, report via the callback
	if upsertCb, exists := r.upserts[key]; exists {
		// fixme: how to handle an error here?
		upsertCb(key, old, val)
	}

	return nil
}

// addIntent will write the intent to the map. This map will be saved to disk
// using te kv.
func (r *RemoteKV) addIntent(key string, val []byte) error {
	r.Intents[key] = val
	return r.saveIntents()
}

// removeIntent will delete the intent from the map. This modified map will be
// saved to disk using the kv.
func (r *RemoteKV) removeIntent(key string) error {
	delete(r.Intents, key)
	return r.saveIntents()
}

// saveIntents is a utility function which writes the Intents map to disk.
func (r *RemoteKV) saveIntents() error {
	data, err := json.Marshal(r.Intents)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   intentsVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return r.kv.Set(intentsKey, obj)
}

// loadIntents will load any intents from kv if present and set it into
// Intents.
func (r *RemoteKV) loadIntents() error {
	obj, err := r.kv.Get(intentsKey, intentsVersion)
	if err != nil { // Return if there isn't any intents stored
		return nil
	}

	return json.Unmarshal(obj.Data, &r.Intents)
}

///////////////////////////////////////////////////////////////////////////////
// Remote File System Implementation
///////////////////////////////////////////////////////////////////////////////

// FileSystemRemoteStorage is a structure adhering to RemoteStore. This
// utilizes the os.File IO operations. Implemented for testing purposes for
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

// ReadAndGetLastWrite is a combination of FileIO.Read and GetLastWrite.
// todo: test.
func (f *FileSystemRemoteStorage) ReadAndGetLastWrite(
	path string) ([]byte, time.Time, error) {
	lastWrite, err := utils.GetLastModified(f.baseDir)
	if err != nil {
		return nil, time.Time{}, err
	}

	if !utils.Exists(path) {
		path = f.baseDir + path
	}

	data, err := utils.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}

	return data, lastWrite, nil
}

// ReadDir reads the named directory, returning all its directory entries
// sorted by filename.
func (f *FileSystemRemoteStorage) ReadDir(path string) ([]string, error) {
	return utils.ReadDir(path)
}
