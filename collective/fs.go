////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"path/filepath"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/utils"
)

///////////////////////////////////////////////////////////////////////////////
// File System Storage Implementation
///////////////////////////////////////////////////////////////////////////////

// FileSystemStorage implements [RemoteStore], and can be used as a
// local [FileIO] for the mutate log as well as for testing
// RemoteStorage users. This utilizes the [os.File] IO
// operations.
type FileSystemStorage struct {
	baseDir   string
	lastWrite time.Time
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

// Read reads data from path. This will return an error if it fails to Read
// from the file path.
//
// This utilizes utils.ReadFile under the hood.
func (f *FileSystemStorage) Read(path string) ([]byte, error) {
	return utils.ReadFile(filepath.Join(f.baseDir, path))
}

// Write will Write data to path. This will return an error if it fails to
// Write.
//
// This utilizes utils.WriteFileDef under the hood.
func (f *FileSystemStorage) Write(path string, data []byte) error {
	p := filepath.Join(f.baseDir, path)
	jww.INFO.Printf("Writing: %s", p)
	err := utils.WriteFileDef(p, data)
	if err != nil {
		return err
	}
	f.lastWrite, err = f.GetLastModified(path)
	return err
}

// GetLastModified will return the last modified timestamp of the file at path.
// It will return an error if it cannot retrieve any os.FileInfo from the file
// path.
//
// This utilizes utils.GetLastModified under the hood.
func (f *FileSystemStorage) GetLastModified(path string) (
	time.Time, error) {
	return utils.GetLastModified(filepath.Join(f.baseDir, path))
}

// GetLastWrite will retrieve the most recent successful Write operation
// that was received by RemoteStore.
func (f *FileSystemStorage) GetLastWrite() (time.Time, error) {
	return f.lastWrite, nil
}

// ReadDir implements [RemoteStore.ReadDir] and gets a file listing.
func (f *FileSystemStorage) ReadDir(path string) ([]string, error) {
	return utils.ReadDir(filepath.Join(f.baseDir, path))
}
