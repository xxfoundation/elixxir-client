////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/sync"
)

////////////////////////////////////////////////////////////////////////////////
// Local Storage Interface & Implementation(s)                                //
////////////////////////////////////////////////////////////////////////////////

// FileIO contains the interface to write and read files to a specific path.
type FileIO interface {
	// Read will read from the provided file path and return the data at that
	// path. An error will be returned if it failed to read the file.
	Read(path string) ([]byte, error)

	// Write will write to the file path the provided data. An error will be
	// returned if it fails to write to file.
	Write(path string, data []byte) error
}

// LocalStore is the mechanism that all local storage implementations should
// adhere to.
type LocalStore interface {
	// FileIO will be used to write and read files.
	FileIO
}

// LocalStoreEKV is a structure adhering to LocalStore. This utilizes
// versioned.KV file IO operations.
type LocalStoreEKV struct {
	api *sync.EkvLocalStore
}

// NewEkvLocalStore is a constructor for LocalStoreEKV.
func NewEkvLocalStore(baseDir, password string) (*LocalStoreEKV, error) {
	api, err := sync.NewEkvLocalStore(baseDir, password)
	if err != nil {
		return nil, err
	}

	return &LocalStoreEKV{api: api}, nil
}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes ekv.KeyValue under the hood.
func (ls *LocalStoreEKV) Read(path string) ([]byte, error) {
	return ls.api.Read(path)
}

// Write will write data to path. This will return an error if it fails to
// write.
//
// This utilizes ekv.KeyValue under the hood.
func (ls *LocalStoreEKV) Write(path string, data []byte) error {
	return ls.api.Write(path, data)
}

////////////////////////////////////////////////////////////////////////////////
// remote Storage Interface & Implementation(s)                               //
////////////////////////////////////////////////////////////////////////////////

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to.
type RemoteStore interface {
	// FileIO will be used to write and read files.
	FileIO

	// GetLastModified will return when the file at the given file path was last
	// modified. If the implementation that adheres to this interface does not
	// support this, Write or Read should be implemented to either write a
	// separate timestamp file or add a prefix.
	GetLastModified(path string) ([]byte, error)

	// GetLastWrite will retrieve the most recent successful write operation
	// that was received by RemoteStore.
	GetLastWrite() ([]byte, error)
}

// RemoteStoreFileSystem is a structure adhering to RemoteStore. This
// utilizes the os.File IO operations. Implemented for testing purposes for
// transaction logs.
type RemoteStoreFileSystem struct {
	api *sync.FileSystemRemoteStorage
}

// NewFileSystemRemoteStorage is a constructor for FileSystemRemoteStorage.
//
// Arguments:
//   - baseDir - string. Represents the base directory for which all file
//     operations will be performed. Must contain a file delimiter (i.e. `/`).
func NewFileSystemRemoteStorage(baseDir string) *RemoteStoreFileSystem {
	return &RemoteStoreFileSystem{
		api: sync.NewFileSystemRemoteStorage(baseDir),
	}
}

// Read will read from the provided file path and return the data at that
// path. An error will be returned if it failed to read the file.
func (r *RemoteStoreFileSystem) Read(path string) ([]byte, error) {
	return r.api.Read(path)
}

// Write will write to the file path the provided data. An error will be
// returned if it fails to write to file.
func (r *RemoteStoreFileSystem) Write(path string, data []byte) error {
	return r.api.Write(path, data)
}

// GetLastModified will return when the file at the given file path was last
// modified. If the implementation that adheres to this interface does not
// support this, Write or Read should be implemented to either write a
// separate timestamp file or add a prefix.
func (r *RemoteStoreFileSystem) GetLastModified(path string) ([]byte, error) {
	ts, err := r.api.GetLastModified(path)
	if err != nil {
		return nil, err
	}

	rsr := &RemoteStoreReport{
		LastModified: ts.UnixNano(),
	}

	return json.Marshal(rsr)
}

// GetLastWrite will retrieve the most recent successful write operation
// that was received by RemoteStore.
func (r *RemoteStoreFileSystem) GetLastWrite() ([]byte, error) {
	ts, err := r.api.GetLastWrite()
	if err != nil {
		return nil, err
	}

	rsr := &RemoteStoreReport{
		LastWrite: ts.UnixNano(),
	}

	return json.Marshal(rsr)
}

////////////////////////////////////////////////////////////////////////////////
// RemoteKV Methods                                                           //
////////////////////////////////////////////////////////////////////////////////

// SyncRemoteKV implements a remote KV to handle transaction logs. These will
// write and read state data from another device to a remote storage interface.
type SyncRemoteKV struct {
	rkv *sync.RemoteKV
}

// RemoteStoreReport will contain the data from the remote storage interface.
type RemoteStoreReport struct {
	// LastModified is the timestamp (in ns) of the last time the specific path
	// was modified. Refer to SyncRemoteKV.GetLastModified.
	LastModified int64

	// LastWrite is the timestamp (in ns) of the last write to the remote
	// storage interface by any device. Refer to SyncRemoteKV.GetLastWrite.
	LastWrite int64
	// Data []byte
}

// KeyUpdateCallback is the callback used to report the event.
type KeyUpdateCallback interface {
	Callback(key, val string)
}

// RemoteStoreCallback is a callback for reporting the status of writing the
// new transaction to remote storage.
type RemoteStoreCallback interface {
	Callback(newTx []byte, err string)
}

// NewOrLoadSyncRemoteKV will construct a SyncRemoteKV.
//
// Parameters:
//   - e2eID - ID of the e2e object in the tracker.
//   - txLogPath - the path that the state data for this device will be written to
//     locally (e.g. sync/txLog.txt)
func NewOrLoadSyncRemoteKV(e2eID int, txLogPath string,
	keyUpdateCb KeyUpdateCallback, remoteStoreCb RemoteStoreCallback,
	remote RemoteStore, local LocalStore,
	upsertCbKeys []string) (*SyncRemoteKV, error) {
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	rng := e2eCl.api.GetRng().GetStream()

	// todo: properly define
	var deviceSecret []byte
	//deviceSecret = e2eCl.GetDeviceSecret()

	// todo: How to do this one?
	var upsertCb map[string]sync.UpsertCallback
	// fixme do this? :
	//	 pass in upsertCbKeys []string, upsertCbsList []upsertCallback?
	//	 require.EqualLen(upsertCbKeys, upsertCbsList)
	// 	 upsertCbs = make(map[string]sync.UpsertCallback, 0)
	// 	 for i, key := range upsertCbKeys {
	// 	 		upsertCbs[key] = sync.UpsertCallback {
	//	 			upsertCbsList[i].Callback()
	//       	}
	// 	 }
	//  is upsertCbsList []upsertCallback a valid param via bidnings?

	// Construct the key update CB
	var eventCb sync.KeyUpdateCallback = func(k, v string) {
		keyUpdateCb.Callback(k, v)
	}
	// Construct update CB
	var updateCb sync.RemoteStoreCallback = func(newTx sync.Transaction,
		err error) {
		if err != nil {
			remoteStoreCb.Callback(nil, err.Error())
		}

		serialized, err := newTx.MarshalJSON()
		if err != nil {
			remoteStoreCb.Callback(nil, err.Error())
		}

		remoteStoreCb.Callback(serialized, "")
	}

	// Construct or load a transaction loc
	txLog, err := sync.NewOrLoadTransactionLog(txLogPath, local, remote,
		deviceSecret, rng)
	if err != nil {
		return nil, err
	}

	// Construct remote KV
	rkv, err := sync.NewOrLoadRemoteKV(txLog, e2eCl.api.GetStorage().GetKV(),
		upsertCb, eventCb, updateCb)
	if err != nil {
		return nil, err
	}

	return &SyncRemoteKV{rkv: rkv}, nil
}

// Write will write a transaction to the remote and local store.
func (s *SyncRemoteKV) Write(path string, data []byte) error {
	var updateCb sync.RemoteStoreCallback
	return s.rkv.Set(path, data, updateCb)
}

// Read retrieves the data stored in the underlying kv. Will return an error
// if the data at this key cannot be retrieved.
func (s *SyncRemoteKV) Read(path string) ([]byte, error) {
	return s.rkv.Get(path)
}
