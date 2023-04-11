////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"time"

	"gitlab.com/xx_network/primitives/utils"
)

///////////////////////////////////////////////////////////////////////////////
// File System Storage Implementation
///////////////////////////////////////////////////////////////////////////////

// FileSystemStorage implements [RemoteStore], and can be used as a
// local [FileIO] for the transaction log as well as for testing
// RemoteStorage users. This utilizes the [os.File] IO
// operations. Implemented for testing purposes for transaction logs.
type FileSystemStorage struct {
	baseDir string
}

// NewFileSystemRemoteStorage is a constructor for FileSystemRemoteStorage.
//
// Arguments:
//   - baseDir - string. Represents the base directory for which all file
//     operations will be performed. Must contain a file delimiter (i.e. `/`).
func NewFileSystemRemoteStorage(baseDir string) *FileSystemStorage {
	return &FileSystemStorage{
		baseDir: baseDir,
	}
}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes utils.ReadFile under the hood.
func (f *FileSystemStorage) Read(path string) ([]byte, error) {
	if utils.DirExists(path) {
		return utils.ReadFile(f.baseDir + path)
	}
	return utils.ReadFile(path)
}

// Write will write data to path. This will return an error if it fails to
// write.
//
// This utilizes utils.WriteFileDef under the hood.
func (f *FileSystemStorage) Write(path string, data []byte) error {
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
func (f *FileSystemStorage) GetLastModified(path string) (
	time.Time, error) {
	if utils.DirExists(path) {
		return utils.GetLastModified(f.baseDir + path)
	}
	return utils.GetLastModified(path)
}

// GetLastWrite will retrieve the most recent successful write operation
// that was received by RemoteStore.
func (f *FileSystemStorage) GetLastWrite() (time.Time, error) {
	return utils.GetLastModified(f.baseDir)
}
