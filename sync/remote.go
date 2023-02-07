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

// FileSystemRemoteStorage is a structure adhering to RemoteStore. This
// utilizes the os.File IO operations.
type FileSystemRemoteStorage struct {
	baseDir string
}

// NewFileSystemRemoteStorage is a constructor for FileSystemRemoteStorage.
//
// Arguments:
//  - baseDir - string. Represents the base directory for which all file
//    operations will be performed. Must contain a file delimiter (i.e. `/`).
func NewFileSystemRemoteStorage(baseDir string) *FileSystemRemoteStorage {
	return &FileSystemRemoteStorage{
		baseDir: baseDir,
	}
}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes utils.ReadFile under the hood.
func (f *FileSystemRemoteStorage) Read(path string) ([]byte, error) {
	return utils.ReadFile(f.baseDir + path)
}

// Write will write data to path. This will return an error if it fails to write.
//
// This utilizes utils.WriteFileDef under the hood.
func (f *FileSystemRemoteStorage) Write(path string, data []byte) error {
	return utils.WriteFileDef(f.baseDir+path, data)
}

// GetLastModified will return the last modified timestamp of the file at path.
// It will return an error if it cannot retrieve any os.FileInfo from the file
// path.
//
// This utilizes utils.GetLastModified under the hood.
func (f *FileSystemRemoteStorage) GetLastModified(path string) (
	time.Time, error) {
	return utils.GetLastModified(f.baseDir + path)
}

// GetLastWrite will retrieve the most recent successful write operation
// that was received by RemoteStore.
func (f *FileSystemRemoteStorage) GetLastWrite() (time.Time, error) {
	return utils.GetLastModified(f.baseDir)
}
