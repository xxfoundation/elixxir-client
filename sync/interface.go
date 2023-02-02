////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import "time"

// RemoteStore is the mechanism that all remote storage implementations should
// adhere to.
type RemoteStore interface {
	// Read will read from the provided file path and return the data at that
	// path. An error will be returned if it failed to read the file.
	Read(path string) ([]byte, error)

	// Write will write to the file path the provided data. An error will be
	// returned if it fails to write to file.
	Write(path string, data []byte) error

	// GetLastModified will return when the file at the given file path was last
	// modified. If the implementation that adheres to this interface does not
	// support this, Write or Read should be implemented to either write a
	// separate timestamp file or add a prefix.
	GetLastModified(path string) (time.Time, error)
}
