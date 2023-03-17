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
}

// FileIO contains the interface to write and read files to a specific path.
type FileIO interface {
	// Read reads from the provided file path and returns the data at that path.
	// An error is returned if it failed to read the file.
	Read(path string) ([]byte, error)

	// Write writes to the file path the provided data. An error is returned if
	// it fails to write to file.
	Write(path string, data []byte) error
}
