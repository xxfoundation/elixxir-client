////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

// FilesystemRemoteStorage is a structure adhering to RemoteStore. This
// utilizes the os.File IO operations.
type FilesystemRemoteStorage struct{}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes utils.ReadFile under the hood.
func (f *FilesystemRemoteStorage) Read(path string) ([]byte, error) {
	return utils.ReadFile(path)
}

// Write will write data to path. This will return an error if it fails to write.
//
// This utilizes utils.WriteFileDef under the hood.
func (f *FilesystemRemoteStorage) Write(path string, data []byte) error {
	return utils.WriteFileDef(path, data)
}

// GetLastModified will return the last modified timestamp of the file at path.
// It will return an error if it cannot retrieve any os.FileInfo from the file
// path.
//
// This utilizes utils.GetLastModified under the hood.
func (f *FilesystemRemoteStorage) GetLastModified(path string) (
	time.Time, error) {
	return utils.GetLastModified(path)
}
