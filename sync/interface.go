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
type UpsertCallback func(key string, curVal, newVal []byte) ([]byte, error)

// KeyUpdateCallback is the callback used to report the event.
type KeyUpdateCallback func(k, v string)

// RemoteStoreCallback is a callback for reporting the status of writing the
// new transaction to remote storage.
type RemoteStoreCallback func(newTx Transaction, err error)

// DeviceID is the identified of a certain device that holds account state.
type DeviceID string

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to.
type RemoteStore interface {
	// FileIO will be used to write and read files.
	FileIO

	// GetLastModified will return when the file at the given file path was last
	// modified. If the implementation that adheres to this interface does not
	// support this, Write or Read should be implemented to either write a
	// separate timestamp file or add a prefix.
	GetLastModified(path string) (time.Time, error)

	// GetLastWrite will retrieve the most recent successful write operation
	// that was received by RemoteStore.
	GetLastWrite() (time.Time, error)

	// ReadAndGetLastWrite is a combination of FileIO.Read and GetLastWrite.
	ReadAndGetLastWrite(path string) ([]byte, time.Time, error)

	// ReadDir reads the named directory, returning all its directory entries
	// sorted by filename.
	ReadDir(path string) ([]string, error)
}

// LocalStore is the mechanism that all local storage implementations should
// adhere to.
type LocalStore interface {
	// FileIO will be used to write and read files.
	FileIO
}

// FileIO contains the interface to write and read files to a specific path.
type FileIO interface {
	// Read will read from the provided file path and return the data at that
	// path. An error will be returned if it failed to read the file.
	Read(path string) ([]byte, error)

	// Write will write to the file path the provided data. An error will be
	// returned if it fails to write to file.
	Write(path string, data []byte) error
}

// Miscellaneous and private types.
type (
	// changeLogger maps the device ID to the last time the device had updates
	// read from remote.
	changeLogger map[DeviceID]time.Time
)
