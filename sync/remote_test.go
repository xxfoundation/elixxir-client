////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

// Smoke test for FileSystemRemoteStorage that executes every method of
// RemoteStore.
//
// As of writing, FileSystemRemoteStorage heavily utilizes the xx network's
// primitives/utils package. As such, testing is light touch as heavier testing
// exists within the dependency.
func TestFileSystemRemoteStorage_Smoke(t *testing.T) {
	baseDir := "delete/"
	path := "test.txt"
	data := []byte("Test string.")

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	fsRemote := NewFileSystemRemoteStorage(baseDir)

	// Write to file
	writeTimestamp := time.Now()
	require.NoError(t, fsRemote.Write(baseDir+path, data))

	// Read file
	read, err := fsRemote.Read(baseDir + path)
	require.NoError(t, err)

	// Ensure read data matches originally written data
	require.Equal(t, data, read)

	// Retrieve the last modification of the file
	lastModified, err := fsRemote.GetLastModified(baseDir + path)
	require.NoError(t, err)

	//time.Sleep(50 * time.Millisecond)

	// The last modified timestamp should not differ by more than a few
	// milliseconds from the timestamp taken before the write operation took
	// place.
	require.True(t, lastModified.Sub(writeTimestamp) < 2*time.Millisecond ||
		lastModified.Sub(writeTimestamp) > 2*time.Millisecond)

	// Sleep here to ensure the new write timestamp significantly differs
	// from the old write timestamp

	// Ensure last write matches last modified when checking the filepath
	// of the file that was las written to
	lastWrite, err := fsRemote.GetLastWrite()
	require.NoError(t, err)

	require.Equal(t, lastWrite, lastModified)

	// Write a new file to remote
	newPath := "new.txt"
	newWriteTimestamp := time.Now()
	require.NoError(t, fsRemote.Write(baseDir+newPath, data))

	// Retrieve the last write
	newLastWrite, err := fsRemote.GetLastWrite()
	require.NoError(t, err)

	// The last write timestamp should not differ by more than a few
	// milliseconds from the timestamp taken before the write operation took
	// place.
	require.True(t, newWriteTimestamp.Sub(newLastWrite) < 2*time.Millisecond ||
		newWriteTimestamp.Sub(newLastWrite) > 2*time.Millisecond)

}
