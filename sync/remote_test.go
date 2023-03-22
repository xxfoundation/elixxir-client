////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"os"
	"strconv"
	"testing"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// Remote KV Testing
///////////////////////////////////////////////////////////////////////////////

// Smoke test of NewOrLoadRemoteKV.
func TestNewOrLoadRemoteKv(t *testing.T) {
	// Construct transaction log
	workingDir := baseDir + "removeKvSmoke/"
	txLog := makeTransactionLog(workingDir, password, t)
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	received, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Create expected remote kv

	expected := &RemoteKV{
		local:          kv.Prefix(remoteKvPrefix),
		txLog:          txLog,
		KeyUpdate:      nil,
		UnsyncedWrites: make(map[string][]byte, 0),
		connected:      true,
	}

	// Check equality of created vs expected remote kv
	require.Equal(t, expected, received)
}

// Unit test for NewOrLoadRemoteKV. Ensures that it will load if there is data
// on disk.
func TestNewOrLoadRemoteKv_Loading(t *testing.T) {

	// Construct transaction log
	workingDir := baseDir + "loading/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Add intents to remote KV
	const numTests = 100
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), "val"+strconv.Itoa(i)
		require.NoError(t, rkv.addUnsyncedWrite(key, []byte(val)))
	}

	// Ensure intents is not empty
	require.NotEmpty(t, rkv.UnsyncedWrites)

	// Call NewOrLoad where it should load intents
	loaded, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Ensure loaded matches original remoteKV
	require.Equal(t, rkv, loaded)
}

// Unit test of RemoteKV.Set.
func TestRemoteKV_Set(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	workingDir := baseDir + "set/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Construct mock update callback
	txChan := make(chan Transaction, numTests)
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		require.NoError(t, err)

		txChan <- newTx
	})

	// Add intents to remote KV
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		require.NoError(t, rkv.RemoteSet(key, val, updateCb))

		select {
		case <-time.After(500 * time.Second):
			t.Fatalf("Failed to recieve from callback")
		case tx := <-txChan:
			require.Equal(t, tx.Key, key)
		}
	}
}

// Unit test of RemoteKV.Get.
func TestRemoteKV_Get(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	workingDir := baseDir + "get/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Construct mock update callback
	txChan := make(chan Transaction, numTests)
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		require.NoError(t, err)

		txChan <- newTx
	})

	// Add intents to remote KV
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		require.NoError(t, rkv.Set(key, val, updateCb))

		// Ensure write has completed
		select {
		case <-time.After(500 * time.Second):
			t.Fatalf("Failed to recieve from callback")
		case <-txChan:
		}

		received, err := rkv.Get(key)
		require.NoError(t, err)

		require.Equal(t, val, received)
	}
}

// Unit test of RemoteKV.addUnsyncedWrite and RemoteKV.removeUnsyncedWrite.
func TestRemoteKV_AddRemoveUnsyncedWrite(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	workingDir := baseDir + "addRemove/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Ensure the map's length is incremented every time
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		require.NoError(t, rkv.addUnsyncedWrite(key, val))
		require.Equal(t, i+1, len(rkv.UnsyncedWrites))
	}

	// Ensure the map's length is decremented every time
	for i := 0; i < numTests; i++ {
		key := "key" + strconv.Itoa(i)
		require.NoError(t, rkv.removeUnsyncedWrite(key))
		require.Equal(t, numTests-i-1, len(rkv.UnsyncedWrites))
	}

}

// Unit test of RemoteKV.saveUnsyncedWrites and RemoteKV.loadUnsyncedWrites.
func TestRemoteKV_SaveLoadUnsyncedWrite(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	workingDir := baseDir + "addRemove/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Add unsynced writes to rkv
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		require.NoError(t, rkv.addUnsyncedWrite(key, val))
	}

	// Save unsynced writes to storage
	require.NoError(t, rkv.saveUnsyncedWrites())

	// Save current state into variable
	expected := rkv.UnsyncedWrites

	// Manually clear current state
	rkv.UnsyncedWrites = nil

	// Load map from store into object
	require.NoError(t, rkv.loadUnsyncedWrites())

	// Ensure RemoteKV's map matches previous state
	require.Equal(t, expected, rkv.UnsyncedWrites)
}

// Unit test for RemoteKV.UpsertLocal
func TestRemoteKV_UpsertLocal(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	workingDir := baseDir + "addRemove/"
	txLog := makeTransactionLog(workingDir, password, t)

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	mockKeyUpdateChan := make(chan mockUpsert, 2*numTests)

	var mockCb KeyUpdateCallback = func(key string, oldVal, newVal []byte, updated bool) {
		mockKeyUpdateChan <- mockUpsert{
			key:    key,
			curVal: oldVal,
			newVal: newVal,
		}
	}

	rkv, err := NewOrLoadRemoteKV(txLog, kv, mockCb, nil)
	require.NoError(t, err)

	// Populate w/ initial values
	firstVals := make(map[string][]byte, numTests)
	for i := 0; i < numTests; i++ {
		key, oldVal := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		require.NoError(t, rkv.UpsertLocal(key, oldVal))
		firstVals[key] = oldVal
	}

	// Update all initial vals
	for i := 0; i < numTests; i++ {
		// Upsert locally
		key, newVal := "key"+strconv.Itoa(i), []byte("newVal"+strconv.Itoa(i))
		require.NoError(t, rkv.UpsertLocal(key, newVal))

		// Should receive off of channel from mock upsert handler
		received := <-mockKeyUpdateChan

		// Expected value
		expected := mockUpsert{
			key:    key,
			curVal: firstVals[key],
			newVal: newVal,
		}

		// Ensure consistency between expected and received
		require.Equal(t, expected, received)

	}

}

///////////////////////////////////////////////////////////////////////////////
// Remote File System Testing
///////////////////////////////////////////////////////////////////////////////

// Smoke test for FileSystemRemoteStorage that executes every method of
// RemoteStore.
//
// As of writing, FileSystemRemoteStorage heavily utilizes the xx network's
// primitives/utils package. As such, testing is light touch as heavier testing
// exists within the dependency.
func TestFileSystemRemoteStorage_Smoke(t *testing.T) {
	workingDir := baseDir + "remoteFsSmoke/"
	path := "test.txt"
	data := []byte("Test string.")

	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	// Write to file
	writeTimestamp := time.Now()
	require.NoError(t, fsRemote.Write(workingDir+path, data))

	// Read file
	read, err := fsRemote.Read(workingDir + path)
	require.NoError(t, err)

	// Ensure read data matches originally written data
	require.Equal(t, data, read)

	// Retrieve the last modification of the file
	lastModified, err := fsRemote.GetLastModified(workingDir + path)
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
	require.NoError(t, fsRemote.Write(workingDir+newPath, data))

	// Retrieve the last write
	newLastWrite, err := fsRemote.GetLastWrite()
	require.NoError(t, err)

	// The last write timestamp should not differ by more than a few
	// milliseconds from the timestamp taken before the write operation took
	// place.
	require.True(t, newWriteTimestamp.Sub(newLastWrite) < 2*time.Millisecond ||
		newWriteTimestamp.Sub(newLastWrite) > 2*time.Millisecond)

}
