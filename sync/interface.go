////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"time"
)

// UpsertCallback is a custom upsert handling for specific keys. When an upsert
// is not defined, the default is used (overwrite the previous key).
type UpsertCallback interface {
	Callback(key string, curVal, newVal []byte)
}

// KeyUpdateCallback is the callback used to report the event.
type KeyUpdateCallback func(key string, oldVal, newVal []byte, updated bool)

// RemoteStoreCallback is a callback for reporting the status of writing the
// new transaction to remote storage.
type RemoteStoreCallback func(newTx Transaction, err error)

// DeviceId is the identified of a certain device that holds account state.
type DeviceId string

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to.
type RemoteStore interface {
	// FileIO is used to write and read files.
	FileIO

	// GetLastModified returns when the file at the given file path was last
	// modified. If the implementation that adheres to this interface does not
	// support this, FileIO.Write or FileIO.Read should be implemented to either
	// write a separate timestamp file or add a prefix.
	GetLastModified(path string) (time.Time, error)

	// GetLastWrite retrieves the most recent successful write operation that
	// was received by RemoteStore.
	GetLastWrite() (time.Time, error)
}

// LocalStore is the mechanism that all local storage implementations should
// adhere to.
type LocalStore interface {
	// FileIO is used to write and read files.
	FileIO

	// GetList returns a KeyValueMap. This return all locally stored data
	// for all keys starting with the provided name followed by the
	// LocalStoreKeyDelimiter.
	//
	// Example: Assuming a delimiter as "-", if you wrote "channels-123" and
	// "channels-abc" using [FileIO.Write] with some arbitrary data, calling
	// GetList("channels") would return a KeyValueMap containing keys
	// channels-123" and "channels-abc" with their respective data.
	GetList(name string) (KeyValueMap, error)
}

const LocalStoreKeyDelimiter = "-"

// FileIO contains the interface to write and read files to a specific path.
type FileIO interface {
	// Read reads from the provided file path and returns the data at that path.
	// An error is returned if it failed to read the file.
	Read(path string) ([]byte, error)

	// Write writes to the file path the provided data. An error is returned if
	// it fails to write to file.
	Write(path string, data []byte) error
}

// EkvLocalStore type definitions.
type (
	// KeyList is the type for the all keys added to LocalStore. If there is a
	// defined delimiter in the key, an entry will be added to the
	// DelimitedList. For example, assuming a delimiter of "-", the keys
	// "channels-123" and "channels-abc" will have 2 separate entries in the
	// DelimitedList, but will be retrievable from key list using the key
	// "channels".
	KeyList map[string]DelimitedList

	// DelimitedList is the list of all sub-keys for a given delimited key.
	// For the example given in KeyList's description, there would be entries
	// for "123" and "abc".
	DelimitedList map[string]struct{}

	// KeyValueMap maps the full key (non-delimited) to the data that was stored
	// when calling [FileIO.Write] on this key.
	KeyValueMap map[string][]byte
)
