////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"os"
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
	jww.INFO.Printf("ReadDir: %s %s", f.baseDir, path)
	joined := filepath.Join(f.baseDir, path)
	jww.INFO.Printf("joined: %s", joined)
	return readDir(joined)
}

// ReadDir reads the named directory, returning all its directory entries
// sorted by filename.
func readDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
