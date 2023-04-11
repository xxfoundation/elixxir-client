////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/client/v4/sync"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

////////////////////////////////////////////////////////////////////////////////
// Remote Storage Interface and Implementation(s)                             //
////////////////////////////////////////////////////////////////////////////////

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to.
type RemoteStore interface {
	// FileIO is used to write and read files. Refer to [sync.FileIO].
	sync.FileIO

	// GetLastModified returns when the file at the given file path was last
	// modified. If the implementation that adheres to this interface does not
	// support this, FileIO.Write or FileIO.Read should be implemented to either
	// write a separate timestamp file or add a prefix.
	//
	// Returns the JSON of [RemoteStoreReport].
	GetLastModified(path string) ([]byte, error)

	// GetLastWrite retrieves the most recent successful write operation that
	// was received by RemoteStore.
	//
	// Returns the JSON of [RemoteStoreReport].
	GetLastWrite() ([]byte, error)
}

// RemoteStoreFileSystem is a structure adhering to [RemoteStore]. This utilizes
// the [os.File] IO operations. Implemented for testing purposes for transaction
// logs.
type RemoteStoreFileSystem struct {
	api *sync.FileSystemRemoteStorage
}

// NewFileSystemRemoteStorage is a constructor for [RemoteStoreFileSystem].
//
// Parameters:
//   - baseDir - The base directory that all file operations will be performed.
//     It must contain a file delimiter (i.e., `/`).
func NewFileSystemRemoteStorage(baseDir string) *RemoteStoreFileSystem {
	return &RemoteStoreFileSystem{sync.NewFileSystemRemoteStorage(baseDir)}
}

// Read reads from the provided file path and returns the data at that path.
// An error is returned if it failed to read the file.
func (r *RemoteStoreFileSystem) Read(path string) ([]byte, error) {
	return r.api.Read(path)
}

// Write writes to the file path the provided data. An error is returned if it
// fails to write to file.
func (r *RemoteStoreFileSystem) Write(path string, data []byte) error {
	return r.api.Write(path, data)
}

// GetLastModified returns when the file at the given file path was last
// modified. If the implementation that adheres to this interface does not
// support this, [Write] or [Read] should be implemented to either write a
// separate timestamp file or add a prefix.
//
// Returns:
//   - []byte - JSON of [RemoteStoreReport].
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

// GetLastWrite retrieves the most recent successful write operation that was
// received by [RemoteStoreFileSystem].
//
// Returns:
//   - []byte - JSON of [RemoteStoreReport].
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

// remoteStoreFileSystemWrapper is an internal Go wrapper for
// RemoteStoreFileSystem that adheres to sync.RemoteStore.
// fixme: reviewer, is this the correct solution?
type remoteStoreFileSystemWrapper struct {
	bindingsAPI RemoteStore
}

// newRemoteStoreFileSystemWrapper constructs a remoteStoreFileSystemWrapper.
func newRemoteStoreFileSystemWrapper(
	bindingsAPI RemoteStore) *remoteStoreFileSystemWrapper {
	return &remoteStoreFileSystemWrapper{bindingsAPI: bindingsAPI}
}

// Read reads from the provided file path and returns the data at that path.
// An error is returned if it failed to read the file.
func (r *remoteStoreFileSystemWrapper) Read(path string) ([]byte, error) {
	return r.bindingsAPI.Read(path)
}

// Write writes to the file path the provided data. An error is returned if it
// fails to write to file.
func (r *remoteStoreFileSystemWrapper) Write(path string, data []byte) error {
	return r.bindingsAPI.Write(path, data)
}

// GetLastModified returns when the file at the given file path was last
// modified. If the implementation that adheres to this interface does not
// support this, [Write] or [Read] should be implemented to either write a
// separate timestamp file or add a prefix.
func (r *remoteStoreFileSystemWrapper) GetLastModified(
	path string) (time.Time, error) {
	reportData, err := r.bindingsAPI.GetLastModified(path)
	if err != nil {
		return time.Time{}, err
	}

	rsr := &RemoteStoreReport{}
	if err = json.Unmarshal(reportData, rsr); err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, rsr.LastModified), nil
}

// GetLastWrite retrieves the most recent successful write operation that was
// received by RemoteStore.
func (r *remoteStoreFileSystemWrapper) GetLastWrite() (time.Time, error) {
	reportData, err := r.bindingsAPI.GetLastWrite()
	if err != nil {
		return time.Time{}, err
	}

	rsr := &RemoteStoreReport{}
	if err = json.Unmarshal(reportData, rsr); err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, rsr.LastWrite), nil
}

////////////////////////////////////////////////////////////////////////////////
// RemoteKV Methods                                                           //
////////////////////////////////////////////////////////////////////////////////

// RemoteKV implements a remote KV to handle transaction logs. It writes and
// reads state data from another device to a remote storage interface.
type RemoteKV struct {
	rkv *sync.VersionedKV
}

// RemoteStoreReport represents the report from any call to a method of
// [RemoteStore].
//
// Example JSON:
//
//		 {
//		  "key": "exampleKey",
//		  "value": "ZXhhbXBsZVZhbHVl",
//		  "lastModified": 1679173966663412908,
//		  "lastWrite": 1679130886663413268,
//		  "error": "Example error (may not exist if successful)"
//	   }
type RemoteStoreReport struct {
	// Key is the key of the transaction that was written to remote. Getting
	// this via the callback indicates that there is a report for this key.
	Key string

	// Value is the value of the transaction that was written to remote for
	// the key.
	Value []byte

	// LastModified is the timestamp (in nanoseconds) of the last time the
	// specific path was modified. Refer to sync.RemoteKV.GetLastModified.
	LastModified int64 `json:"lastModified"`

	// LastWrite is the timestamp (in nanoseconds) of the last write to the
	// remote storage interface by any device. Refer to
	// sync.RemoteKV.GetLastWrite.
	LastWrite int64 `json:"lastWrite"`

	// Any error that occurs. It is omitted when no error occurred.
	Error string `json:"error,omitempty"`
}

// RemoteKVCallbacks is an interface for the [RemoteKV]. This will handle all
// callbacks used for the various operations [RemoteKV] supports.
type RemoteKVCallbacks interface {
	// KeyUpdated is the callback to be called any time a key is updated by
	// another device tracked by the RemoteKV store.
	KeyUpdated(key string, oldVal, newVal []byte, updated bool)

	// RemoteStoreResult is called to report network save results after the key
	// has been updated locally.
	//
	// NOTE: Errors originate from the authentication and writing code in regard
	// to remote which is handled by the user of this API. As a result, this
	// callback provides no information in simple implementations.
	//
	// Parameters:
	//   - remoteStoreReport - JSON of [RemoteStoreReport].
	RemoteStoreResult(remoteStoreReport []byte)
}

// NewOrLoadSyncRemoteKV will construct a remote [KV].
//
// Parameters:
//   - storageDir - the path to the ekv
//   - remoteKvCallbacks - A [RemoteKVCallbacks]. These will be the callbacks
//     that are called for [RemoteStore] operations.
//   - remote - A [RemoteStore]. This will be a structure the consumer
//     implements. This acts as a wrapper around the remote storage API
//     (e.g., Google Drive's API, DropBox's API, etc.).
func NewOrLoadSyncRemoteKV(storageDir string, remoteKvCallbacks RemoteKVCallbacks,
	remote RemoteStore) (*RemoteKV, error) {

	// todo: properly define
	var deviceSecret = []byte("dummy, replace")
	// deviceSecret = e2eCl.GetDeviceSecret()

	localKV, err := ekv.NewFilestore(storageDir, string(deviceSecret))
	if err != nil {
		return nil, err
	}

	versionedKV := versioned.NewKV(localKV)

	// Construct the key update CB
	var eventCb sync.KeyUpdateCallback = func(key string, oldVal, newVal []byte,
		updated bool) {
		remoteKvCallbacks.KeyUpdated(key, oldVal, newVal, updated)
	}
	// Construct update CB
	var updateCb sync.RemoteStoreCallback = func(newTx sync.Transaction,
		err error) {
		remoteStoreCbUtil(remoteKvCallbacks, newTx, err)
	}

	// Construct local storage
	local, err := sync.NewOrLoadEkvLocalStore(versionedKV)
	if err != nil {
		return nil, err
	}

	// Construct txLog path
	// NOTE: the following assumes this is called after KV
	//       initialization from calling NewCmix, so this needs to
	//       be linked up to that somehow. That all likely needs to
	//       be refactored.
	instanceID, err := xxdk.LoadInstanceID(versionedKV)
	if err != nil {
		return nil, err
	}
	// Is the transmission key the right thing to load? Not sure..
	uid, err := user.LoadUser(versionedKV)
	if err != nil {
		return nil, err
	}
	txKey := base64.RawURLEncoding.EncodeToString(
		uid.GetTransmissionID().Bytes())
	txLogPath := fmt.Sprintf("%s/%s", txKey, instanceID)

	// Retrieve rng
	frng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	rng := frng.GetStream()

	// Construct or load a transaction log
	txLog, err := sync.NewOrLoadTransactionLog(txLogPath, local,
		newRemoteStoreFileSystemWrapper(remote),
		deviceSecret, rng)
	if err != nil {
		return nil, err
	}

	// Construct remote KV
	rkv, err := sync.NewVersionedKV(
		txLog, localKV, nil,
		eventCb, updateCb)
	if err != nil {
		return nil, err
	}

	return &RemoteKV{rkv: rkv}, nil
}

// Write writes a transaction to the remote and local store.
//
// Parameters:
//   - path - The key that this data will be written to (i.e., the device name,
//     the channel name, etc.). Certain keys should follow a pattern and contain
//     special characters (see [RemoteKV.GetList] for details).
//   - data - The data that will be stored (i.e., state data).
//   - cb - A [RemoteKVCallbacks]. This may be nil if you do not care about the
//     network report.
func (s *RemoteKV) Write(path string, data []byte, cb RemoteKVCallbacks) error {
	var updateCb = func(newTx sync.Transaction, err error) {
		remoteStoreCbUtil(cb, newTx, err)
	}
	return s.rkv.Remote().SetRemote(path, data, updateCb)
}

// Read retrieves the data stored in the underlying KV. Returns an error if the
// data at this key cannot be retrieved.
//
// Parameters:
//   - path - The key that this data will be written to (i.e., the device name).
func (s *RemoteKV) Read(path string) ([]byte, error) {
	return s.rkv.Remote().GetBytes(path)
}

// GetList returns all entries for a path (or key) that contain the name
// parameter from the local store.
//
// For example, assuming the usage of the [sync.LocalStoreKeyDelimiter], if both
// "channels-123" and "channels-abc" are written to [RemoteKV], then
// GetList("channels") will retrieve the data for both channels. All data that
// contains no [sync.LocalStoreKeyDelimiter] can be retrieved using GetList("").
//
// Parameters:
//   - name - Some prefix to a Write operation. If no prefix applies, simply
//     use the empty string.
//
// Returns:
//   - []byte - JSON of [sync.KeyValueMap].
func (s *RemoteKV) GetList(name string) ([]byte, error) {
	return s.rkv.Remote().GetList(name)
}

// remoteStoreCbUtil is a utility function for the sync.RemoteStoreCallback.
func remoteStoreCbUtil(cb RemoteKVCallbacks, newTx sync.Transaction, err error) {
	var report RemoteStoreReport
	if err != nil {
		report.Error = err.Error()
	} else {
		report.Key = newTx.Key
		report.Value = newTx.Value
	}

	reportJson, _ := json.Marshal(report)
	cb.RemoteStoreResult(reportJson)
}
