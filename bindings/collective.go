////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/storage/versioned"
)

////////////////////////////////////////////////////////////////////////////////
// Remote Storage Interface and Implementation(s)                             //
////////////////////////////////////////////////////////////////////////////////

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to. Bindings clients must implement this interface to use
// collective synchronized KV functionality.
type RemoteStore interface {
	// FileIO is used to write and read files. Refer to [collective.FileIO].
	collective.FileIO

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

	// ReadDir reads the named directory, returning all its
	// directory entries sorted by filename as json of a []string
	ReadDir(path string) ([]byte, error)
}

// KeyChangedByRemoteCallback is the callback used to report local
// updates caused by a remote client editing their EKV
type KeyChangedByRemoteCallback interface {
	Callback(key string, old, new []byte, opType int8)
}

// MapChangedByRemoteCallback is the callback used to report local
// updates caused by a remote client editing their EKV
type MapChangedByRemoteCallback interface {
	Callback(mapName string, editsJSON []byte)
}

// TransactionOperation provides a function which mutates the object
// in the desired way. Receives and sends a JSON object.
// dataJSON must be JSON of the form:
//
//	{
//	   Error: "string",
//	   NewObject: {
//	        // Used to determine version Upgrade, if any
//	        Version uint64
//	        // Set when this object is written
//	        Timestamp time.Time
//	        // Serialized version of original object
//	        Data []byte
//	   }
//	}
type TransactionOperation interface {
	Operation(oldJSON []byte, existed bool) (dataJSON []byte)
}

////////////////////////////////////////////////////////////////////////////////
// RemoteKV Methods                                                           //
////////////////////////////////////////////////////////////////////////////////

// RemoteKV implements a bindings-friendly subset of a [versioned.KV]. It has
// additional features to store and load maps. It uses strings of json for
// [versioned.Object] to get and set all data. All operations over the bindings
// interface are prefixed by the "bindings" prefix, and this prefix is always
// remotely synchronized.
//
// RemoteKV is instantiated and an instance is acquired via the Cmix object
// [Cmix.GetRemoteKV] function. (TODO: write this function)
type RemoteKV struct {
	rkv versioned.KV
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
	// specific path was modified. Refer to collective.RemoteKV.GetLastModified.
	LastModified int64 `json:"lastModified"`

	// LastWrite is the timestamp (in nanoseconds) of the last write to the
	// remote storage interface by any device. Refer to
	// collective.RemoteKV.GetLastWrite.
	LastWrite int64 `json:"lastWrite"`

	// Any error that occurs. It is omitted when no error occurred.
	Error string `json:"error,omitempty"`
}

// Get returns the object stored at the specified version.
// returns a json of [versioned.Object]
func (r *RemoteKV) Get(key string, version int64) ([]byte, error) {
	jww.DEBUG.Printf("[RKV] Get(%s, %d)", key, version)
	obj, err := r.rkv.Get(key, uint64(version))
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

// Delete removes a given key from the data store.
func (r *RemoteKV) Delete(key string, version int64) error {
	jww.DEBUG.Printf("[RKV] Delete(%s, %d)", key, version)
	return r.rkv.Delete(key, uint64(version))
}

// Set upserts new data into the storage
// When calling this, you are responsible for prefixing the
// key with the correct type optionally unique id! Call
// MakeKeyWithPrefix() to do so.
// The [Object] should contain the versioning if you are
// maintaining such a functionality.
func (r *RemoteKV) Set(key string, objectJSON []byte) error {
	jww.DEBUG.Printf("[RKV] set(%s)", key)
	obj := versioned.Object{}
	err := json.Unmarshal(objectJSON, &obj)
	if err != nil {
		return err
	}
	return r.rkv.Set(key, &obj)
}

// GetPrefix returns the full Prefix of the KV
func (r *RemoteKV) GetPrefix() string {
	return r.rkv.GetPrefix()
}

// HasPrefix returns whether this prefix exists in the KV
func (r *RemoteKV) HasPrefix(prefix string) bool {
	return r.rkv.HasPrefix(prefix)
}

// Prefix returns a new KV with the new prefix appending
func (r *RemoteKV) Prefix(prefix string) (*RemoteKV, error) {
	jww.DEBUG.Printf("[RKV] Prefix(%s)", prefix)

	newK, err := r.rkv.Prefix(prefix)
	if err != nil {
		return nil, err
	}
	newRK := &RemoteKV{
		rkv: newK,
	}
	return newRK, nil
}

// Root returns the KV with no prefixes
func (r *RemoteKV) Root() (*RemoteKV, error) {
	jww.DEBUG.Printf("[RKV] Root()")

	newK, err := r.rkv.Root().Prefix("bindings")
	if err != nil {
		return nil, err
	}
	newRK := &RemoteKV{
		rkv: newK,
	}
	return newRK, nil
}

// IsMemStore returns true if the underlying KV is memory based
func (r *RemoteKV) IsMemStore() bool {
	return r.rkv.IsMemStore()
}

// GetFullKey returns the key with all prefixes appended
func (r *RemoteKV) GetFullKey(key string, version int64) string {
	return r.rkv.GetFullKey(key, uint64(version))
}

// StoreMapElement stores a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
// valueJSON is a json of a versioned.Object
func (r *RemoteKV) StoreMapElement(mapName, elementKey string,
	valueJSON []byte, version int64) error {
	jww.DEBUG.Printf("[RKV] StoreMapElement(%s, %s, %d)", mapName, elementKey,
		version)
	obj := versioned.Object{}
	err := json.Unmarshal(valueJSON, &obj)
	if err != nil {
		return err
	}
	return r.rkv.StoreMapElement(mapName, elementKey, &obj, uint64(version))
}

// StoreMap saves a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMap] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
// valueJSON is a json of map[string]*versioned.Object
func (r *RemoteKV) StoreMap(mapName string,
	valueJSON []byte, version int64) error {
	jww.DEBUG.Printf("[RKV] StoreMap(%s, %d)", mapName, version)
	obj := make(map[string]*versioned.Object)
	err := json.Unmarshal(valueJSON, &obj)
	if err != nil {
		return err
	}
	return r.rkv.StoreMap(mapName, obj, uint64(version))
}

// DeleteMapElement removes a versioned map element from the KV.
func (r *RemoteKV) DeleteMapElement(mapName, elementName string,
	mapVersion int64) ([]byte, error) {
	jww.DEBUG.Printf("[RKV] DeleteMapElement(%s, %s, %d)", mapName,
		elementName, mapVersion)
	oldVal, err := r.rkv.DeleteMapElement(mapName, elementName,
		uint64(mapVersion))
	if err != nil {
		return nil, err
	}

	oldValJSON, err := json.Marshal(oldVal)
	panicOnErr(err)

	return oldValJSON, nil
}

// GetMap loads a versioned map from the KV. This relies
// on the underlying remote [KV.GetMap] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *RemoteKV) GetMap(mapName string, version int64) ([]byte, error) {
	jww.DEBUG.Printf("[RKV] GetMap(%s, %d)", mapName, version)
	mapData, err := r.rkv.GetMap(mapName, uint64(version))
	if err != nil {
		return nil, err
	}
	return json.Marshal(mapData)
}

// GetMapElement loads a versioned map element from the KV. This relies
// on the underlying remote [KV.GetMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
func (r *RemoteKV) GetMapElement(mapName, element string, version int64) (
	[]byte, error) {
	jww.DEBUG.Printf("[RKV] GetMapElement(%s, %s, %d)", mapName, element, version)

	obj, err := r.rkv.GetMapElement(mapName, element, uint64(version))
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

// ListenOnRemoteKey sets up a callback listener for the object specified
// by the key and version. It returns the current [versioned.Object] JSON
// of the value.
func (r *RemoteKV) ListenOnRemoteKey(key string, version int64,
	callback KeyChangedByRemoteCallback) error {

	jww.DEBUG.Printf("[RKV] ListenOnRemoteKey(%s, %d)", key, version)

	bindingsCb := func(old, new *versioned.Object, op versioned.KeyOperation) {
		oldJSON, err := json.Marshal(old)
		panicOnErr(err)
		newJSON, err := json.Marshal(new)
		panicOnErr(err)
		callback.Callback(key, oldJSON, newJSON, int8(op))
	}

	return r.rkv.ListenOnRemoteKey(key, uint64(version), bindingsCb)
}

// ListenOnRemoteMap allows the caller to receive updates when
// the map or map elements are updated. Returns a JSON of
// map[string]versioned.Object of the current map value.
func (r *RemoteKV) ListenOnRemoteMap(mapName string, version int64,
	callback MapChangedByRemoteCallback) error {
	jww.DEBUG.Printf("[RKV] ListenOnRemoteMap(%s, %d)", mapName, version)

	bindingsCb := func(edits map[string]versioned.ElementEdit) {
		editsJSON, err := json.Marshal(edits)
		panicOnErr(err)
		callback.Callback(mapName, editsJSON)
	}

	return r.rkv.ListenOnRemoteMap(mapName, uint64(version),
		bindingsCb)
}

////////////////////////////////////////////////////////////////////////////////
// Other Methods and helper objects                                           //
////////////////////////////////////////////////////////////////////////////////

// remoteStoreWrapper is an internal Go wrapper for RemoteStore that
// adheres to the collective.RemoteStore interface. It is used to wrap
// the RemoteStore given by the bindings user for use on the rest of
// the system.
type remoteStoreWrapper struct {
	store RemoteStore
}

// newRemoteStoreFileSystemWrapper constructs a remoteStoreFileSystemWrapper.
func newRemoteStoreFileSystemWrapper(
	bindingsStore RemoteStore) *remoteStoreWrapper {
	return &remoteStoreWrapper{store: bindingsStore}
}

// Read reads from the provided file path and returns the data at that path.
// An error is returned if it failed to read the file.
func (r *remoteStoreWrapper) Read(path string) ([]byte, error) {
	return r.store.Read(path)
}

// Write writes to the file path the provided data. An error is returned if it
// fails to write to file.
func (r *remoteStoreWrapper) Write(path string, data []byte) error {
	return r.store.Write(path, data)
}

func (r *remoteStoreWrapper) ReadDir(path string) ([]string, error) {
	filesJSON, err := r.store.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	err = json.Unmarshal(filesJSON, &files)
	return files, err
}

// GetLastModified returns when the file at the given file path was last
// modified. If the implementation that adheres to this interface does not
// support this, [Write] or [Read] should be implemented to either write a
// separate timestamp file or add a prefix.
func (r *remoteStoreWrapper) GetLastModified(
	path string) (time.Time, error) {
	reportData, err := r.store.GetLastModified(path)
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
func (r *remoteStoreWrapper) GetLastWrite() (time.Time, error) {
	reportData, err := r.store.GetLastWrite()
	if err != nil {
		return time.Time{}, err
	}

	rsr := &RemoteStoreReport{}
	if err = json.Unmarshal(reportData, rsr); err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, rsr.LastWrite), nil
}

// RemoteStoreFileSystem is a structure adhering to [RemoteStore]. This utilizes
// the [os.File] IO operations. Implemented for testing purposes for transaction
// logs.
type RemoteStoreFileSystem struct {
	api *collective.FileSystemStorage
}

// NewFileSystemRemoteStorage is a constructor for [RemoteStoreFileSystem].
//
// Parameters:
//   - baseDir - The base directory that all file operations will be performed.
//     It must contain a file delimiter (i.e., `/`).
func NewFileSystemRemoteStorage(baseDir string) *RemoteStoreFileSystem {
	return &RemoteStoreFileSystem{collective.NewFileSystemRemoteStorage(baseDir)}
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

type transactionResult struct {
	OldValue *versioned.Object
	Existed  bool
}

// transactionOperationData is the data script for the bindings dataJSON
// TransactionOperation.
type transactionOperationData struct {
	Error     string
	NewObject *versioned.Object
}

func panicOnErr(err error) {
	if err != nil {
		jww.FATAL.Panicf("unexpected value from api: %+v", err)
	}
}
