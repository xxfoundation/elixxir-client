////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"time"

	"gitlab.com/elixxir/client/v4/storage/versioned"
)

// FileIO is a simplified filesystem interface, providing Read and
// Write operations. Directories are implicitly created by the
// implementor of the interface, if necessary.
type FileIO interface {
	// Read reads from the provided file path and returns the data
	// at that path.  An error is returned if it failed to Read
	// the file.
	Read(path string) ([]byte, error)

	// Write writes to the file path the provided data. An error
	// is returned if it fails to Write to file.
	Write(path string, data []byte) error
}

// RemoteStore is a FileIO interface with additional functions for
// Write operation information. It is used to store mutate logs
// on cloud storage or another remote storage solution like an sftp
// server run by the user.
type RemoteStore interface {
	// FileIO is used to Write and Read files.
	FileIO

	// GetLastModified returns when the file at the given file
	// path was last modified. If the implementation that adheres
	// to this interface does not support this, FileIO.Write or
	// [FileIO.Read] should be implemented to either Write a
	// separate timestamp file or add a prefix.
	GetLastModified(path string) (time.Time, error)

	// GetLastWrite retrieves the most recent successful Write
	// operation that was received by RemoteStore.
	GetLastWrite() (time.Time, error)

	// ReadDir reads the named directory, returning all its directory entries
	// sorted by filename.
	ReadDir(path string) ([]string, error)
}

// UpsertCallback is a custom upsert handling for specific keys. When
// an upsert is not defined, the default is used (overwrite the
// previous key).
type UpsertCallback interface {
	Callback(key string, curVal, newVal []byte)
}

// KeyUpdateCallback is the callback used to report the event.
type KeyUpdateCallback func(key string, oldVal, newVal []byte,
	op versioned.KeyOperation)

// RemoteStoreCallback is a callback for reporting the status of
// writing the new mutate to remote storage.
type RemoteStoreCallback func(newTx Mutate, err error)

type dummyIO struct{}

func (dIO *dummyIO) Read(path string) ([]byte, error) {
	return nil, nil
}

func (dIO *dummyIO) Write(path string, data []byte) error {
	return nil
}
