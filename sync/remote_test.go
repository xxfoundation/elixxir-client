////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

///////////////////////////////////////////////////////////////////////////////
// Remote KV Testing
///////////////////////////////////////////////////////////////////////////////

// Smoke test of NewOrLoadRemoteKV.
func TestNewOrLoadRemoteKv(t *testing.T) {
	// Construct transaction log
	txLog := makeTransactionLog("", password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	received, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Create expected remote kv

	kv, err = kv.Prefix(remoteKvPrefix)
	require.NoError(t, err)

	expected := &RemoteKV{
		local:          kv,
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
	txLog := makeTransactionLog("workingDir", password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Call NewOrLoad where it should load intents
	done := make(chan struct{})
	var updateCb RemoteStoreCallback = func(newTx Transaction, err error) {
		done <- struct{}{}
	}

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, updateCb)
	require.NoError(t, err)

	// Wait for loaded to be complete
	//select {
	//case <-done:
	//	break
	//case <-time.NewTimer(5 * time.Second).C:
	//	t.Fatalf("Failed to recieve from callback")
	//}

	// Add intents to remote KV
	const numTests = 100
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), "val"+strconv.Itoa(i)
		require.NoError(t, rkv.addUnsyncedWrite(key, []byte(val)))
	}

	// Ensure intents is not empty
	require.NotEmpty(t, rkv.UnsyncedWrites)

	// Call NewOrLoad where it should load intents
	loaded, err := NewOrLoadRemoteKV(txLog, kv, nil, updateCb)
	require.NoError(t, err)

	require.Len(t, loaded.UnsyncedWrites, numTests)
}

// Unit test of RemoteKV.Set.
func TestRemoteKV_Set(t *testing.T) {
	const numTests = 100

	// Construct transaction log
	txLog := makeTransactionLog("workingDirSet", password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	rkv.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

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
	txLog := makeTransactionLog("workingDir", password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	rkv, err := NewOrLoadRemoteKV(txLog, kv, nil, nil)
	require.NoError(t, err)

	// Overwrite remote w/ non file IO option
	rkv.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

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
	txLog := makeTransactionLog("workingDir", password, t)

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
	txLog := makeTransactionLog("workingDir", password, t)

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
	txLog := makeTransactionLog("workingDir", password, t)

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
