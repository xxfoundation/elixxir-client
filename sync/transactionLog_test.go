////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/ekv"
)

const (
	// expectedTransactionLogSerializedBase64 is the base64 encoded serialized
	// TransactionLog. If the state set in the mock TransactionLog is changed,
	// this value should be changed to reflect this.
	expectedTransactionLogSerializedBase64 = `MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0VAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUZTMwPQYAAAAAAAAAkgAAAAAAAAAwLEFRSURCQVVHQndnSkNnc01EUTRQRUJFU0V4UVZGaGNZZGJqX0M2eUVzcW5NSTgtdVI1QmVUUFp4NlVJdmJJVXlzUW1NM2VtdW5KY3c5ZUliS2l5U3A3alZhZjF1NmVLZzFBYXRaSHFLQW9OcnppZFhpS2xjbW5TQWxFaHdTWHN0Q0E3SWVCdG15bEx4MTE4bZIAAAAAAAAAMSxHUm9iSEIwZUh5QWhJaU1rSlNZbktDa3FLeXd0TGk4d0NFR2paN0pIbUljd3gwb2hnbHBXOE5FbWZnN1RaWXVKUFpSdHdqUU9peXVxOVloVGVHeURpUWxEcndrbS1ualdmTVhMTWw2bVZOZHJkaE91XzMxaDlLeEliVEN3blV2NWV5czJ1YS1PMXBCWG5kcFaSAAAAAAAAADIsTVRJek5EVTJOemc1T2pzOFBUNF9RRUZDUTBSRlJrZElISUNBSUVDb2RteW9VN2xsYmFoWTFLNW80ZFVNYndJVnpUVUNXT0F6enZWZE0tWm1SbUVxOGpWTUVsekVtaWlPMlpKSEtWZ0hIaTdoUFBKTFZrTGdxOE9NMFFuN2QyOVN3SjVfSW9MSFdSRkdQYm5SkgAAAAAAAAAzLFNVcExURTFPVDFCUlVsTlVWVlpYV0ZsYVcxeGRYbDlnR1l2c25ZV3pfeUNIVXhnUnQxdVZPVS1pZHEyTXF2bWlYX092UFpYcGNuZFpvMUdVMEgzRFB5bktGb2FEU2t6TDNuYXpjcmIzOTdrTmZZMk9ib2pEM2puSGJ5WWZnbzJlM1FLakFkWl9ybi1aNV82OZIAAAAAAAAANCxZV0pqWkdWbVoyaHBhbXRzYlc1dmNIRnljM1IxZG5kNG1CUkt4Tkd5eWlBMXNGUzM5RnFFSXFaZUh5dVpBYjBIc3J0X1BPMFhkX3REeV94Q2JRNnpnSGFuWWNJdTl5YVJPTF9RcWNoWGx6NEk2QWhONjAzekQxWFVMZVNZeU83LWROeUhCb3hJOTMyWUw2aGiSAAAAAAAAADUsZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LaTR5TmpvLVE3UzJvam9DZC1FZEdQTnl3VS13anNSQ0hPNXVXSWZmY2xMOFpoYUs4eTRaV2x0S1ZsVW0xT1BSOGJiQVd1c0tEV1lZUlRLdmZKRldHNFhhM0VYMVVaUktDVm9rWVQ2YjNKR1VRbTZxbTVwRmhIOFlJ`
)

// Smoke test for NewOrLoadTransactionLog.
//
// Intentionally constructs TransactionLog manually for testing purposes.
func TestNewOrLoadTransactionLog(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir", "password"
	fs, err := ekv.NewFilestore(baseDir, password)
	require.NoError(t, err)
	localStore := NewKVFilesystem(fs)

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
		deviceSecret, rand.Reader)
	require.NoError(t, err)

	// Construct expected transaction log object
	expected := &TransactionLog{
		path:         baseDir,
		local:        localStore,
		remote:       remoteStore,
		Header:       NewHeader(),
		txs:          make([]Transaction, 0),
		deviceSecret: deviceSecret,
		rng:          rand.Reader,
		offsets:      make(deviceOffset, 0),
	}

	// Ensure constructor generates expected object
	require.Equal(t, expected, txLog)

}

// Unit test for NewOrLoadTransactionLog. Tests whether this will load from
// disk and deserialize the data into the TransactionLog file.
//
// Intentionally constructs TransactionLog manually for testing purposes.
func TestNewOrLoadTransactionLog_Loading(t *testing.T) {
	// Construct local store
	localStore := NewKVFilesystem(ekv.MakeMemstore())

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	appendCb := RemoteStoreCallback(func(newTx Transaction, err error) {})

	remoteStore := &mockRemote{data: make(map[string][]byte)}

	// Construct transaction log
	txLog, err := NewTransactionLog("baseDir", localStore,
		remoteStore,
		deviceSecret, rand.Reader)
	require.NoError(t, err)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert timestamps
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		require.NoError(t, txLog.Append(newTx, appendCb))
	}

	ok := txLog.WaitForRemote(60 * time.Second)
	if !ok {
		t.Errorf("threads failed to stop")
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	}

	// Construct a new TransactionLog, which will load from file
	newTxLog, err := NewTransactionLog("baseDir", localStore,
		remoteStore,
		deviceSecret, rand.Reader)
	require.NoError(t, err)

	// Ensure loaded log matches original log
	require.Equal(t, txLog, newTxLog)
}

// Unit test for Append. Ensure that callback is called with every call
// to TransactionLog.Append.
func TestTransactionLog_Append_Callback(t *testing.T) {
	// Construct transaction log
	txLog := makeTransactionLog("baseDir", password, t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert transaction
	for cnt, curTs := range mockTimestamps {
		curChan := make(chan Transaction, 1)
		// Set append callback manually
		appendCb := RemoteStoreCallback(func(newTx Transaction, err error) {
			curChan <- newTx
		})

		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Append transaction
		require.NoError(t, txLog.Append(newTx, appendCb))

		// Wait for signal sent in callback (or timeout)
		select {
		case <-time.After(50 * time.Millisecond):
			t.Fatalf("Failed to receive from callback")
		case receivedTx := <-curChan:
			require.Equal(t, newTx, receivedTx)
		}
		close(curChan)
	}

}

// Unit test for Save. Ensures that TransactionLog's save function writes to
// remote and local stores when they are set.
func TestTransactionLog_Save(t *testing.T) {
	// Construct transaction log
	txLog := makeTransactionLog("baseDir", password, t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)
	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction (without saving)
		txLog.appendUsingQuickSort(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Construct callback
	finishedWritingToRemote := make(chan struct{}, 1)
	appendCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		finishedWritingToRemote <- struct{}{}
	})

	// Write data to remote & local
	require.NoError(t, txLog.save(Transaction{}, data, appendCb))

	// Read from local
	dataFromLocal, err := txLog.local.Read(txLog.path)
	require.NoError(t, err)

	// Ensure read data from local matches originally written
	require.Equal(t, data, dataFromLocal)

	// Remote writing is done async, so wait for channel reception via
	// cb (or timeout)
	timeout := time.NewTimer(100 * time.Millisecond)
	select {
	case <-timeout.C:
		t.Fatalf("Test timed!")
	case <-finishedWritingToRemote:
		// Read from remote
		dataFromRemote, err := txLog.remote.Read(txLog.path)
		require.NoError(t, err)

		// Ensure read data from remote matches originally written
		require.Equal(t, data, dataFromRemote)
	}

	// Now that remote data is written, ensure it is present in remote:

	// Read from remote
	dataFromRemote, err := txLog.remote.Read(txLog.path)
	require.NoError(t, err)

	// Ensure read data from remote matches originally written
	require.Equal(t, data, dataFromRemote)
}

// Unit test for Append. Ensures that appendUsingInsertion function will insert
// new Transaction's into the TransactionLog, and that the transactions are
// sorted by timestamp after the insertion.
func TestTransactionLog_Append_Sorting(t *testing.T) {
	// Construct transaction log
	txLog := makeTransactionLog("baseDir", password, t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 6)

	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Append w/o saving using default append
		txLog.appendUsingInsertion(newTx)

		// Ensure that these transactions have been inserted in order for each
		// insertion
		require.True(t, sort.SliceIsSorted(txLog.txs, func(i, j int) bool {
			firstTs, secondTs := txLog.txs[i].Timestamp, txLog.txs[j].Timestamp
			return firstTs.Before(secondTs)
		}))
	}

	// Ensure that all insertions occurred (no rewrites).
	require.Equal(t, len(mockTimestamps), len(txLog.txs))
}

// Unit test for Serialize. Ensures the that function returns the serialized
// internal state. Checks against a hardcoded base64 string.
func TestTransactionLog_Serialize(t *testing.T) {
	// Construct transaction log
	txLog := makeTransactionLog("baseDir", password, t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction
		txLog.appendUsingInsertion(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Encode data to bas64
	data64 := base64.RawStdEncoding.EncodeToString(data)

	// Ensure encoded data using mock values matches hardcoded data.
	require.Equal(t, expectedTransactionLogSerializedBase64, data64)
}

// Unit test for Deserialize. Ensures that deserialize will construct the same
// TransactionLog that was serialized using TransactionLog.serialize.
//
// Intentionally constructs TransactionLog manually for testing purposes.
func TestTransactionLog_Deserialize(t *testing.T) {
	// Construct local store
	baseDir := "testDir"
	localStore := NewKVFilesystem(ekv.MakeMemstore())

	// Construct remote store
	remoteStore := &mockRemote{data: make(map[string][]byte)}

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
		deviceSecret, &CountingReader{count: 0})
	require.NoError(t, err)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction
		txLog.appendUsingInsertion(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Construct a log w/o header and transaction list
	newTxLog := &TransactionLog{
		path:         baseDir,
		local:        localStore,
		remote:       remoteStore,
		deviceSecret: deviceSecret,
		rng:          txLog.rng,
	}

	// Deserialize the transaction log
	require.NoError(t, newTxLog.deserialize(data))

	// Ensure deserialized object matches original object
	require.Equal(t, txLog, newTxLog)
}

// Error case for saveToRemote. Ensures that it should panic when
// TransactionLog's remoteStoreCallback is nil.
func TestTransactionLog_SaveToRemote_NilCallback(t *testing.T) {
	// Construct transaction log
	baseDir := "testDir/"
	txLog := makeTransactionLog(baseDir, password, t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)
	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction
		txLog.appendUsingInsertion(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("saveToRemote should panic as callback is nil")
		}
	}()

	// Write data to remote & local
	txLog.saveToRemote(Transaction{}, data, nil)

}

// Benchmark the performance of appending to a transaction log using insertion
// sort.
//
// Intentionally constructs TransactionLog manually for testing purposes.
func BenchmarkTransactionLog_AppendInsertion(b *testing.B) {
	// Construct local store
	baseDir := "testDir"
	localStore := NewKVFilesystem(ekv.MakeMemstore())

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// We expect the number of transactions to reach this checkpoint within
	// a few weeks.
	const numRandomTimestamps = 10000

	// Construct timestamps
	mockTimestamps := constructTimestamps(b, numRandomTimestamps)

	for i := 0; i < b.N; i++ {

		// Construct new transaction log for benchmark iteration
		txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
			deviceSecret, rand.Reader)
		require.NoError(b, err)

		for cnt, curTs := range mockTimestamps {
			// Construct transaction
			key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
			newTx := NewTransaction(curTs, key, []byte(val))

			// Append and use insertion sort
			txLog.appendUsingInsertion(newTx)

		}
	}
}

// Benchmark the performance of appending to a transaction log using quicksort
// (default algorithm of sort.Slice).
//
// Intentionally constructs TransactionLog manually for testing purposes.
func BenchmarkTransactionLog_AppendQuick(b *testing.B) {
	// Construct local store
	baseDir, password := "testDir", "password"
	fs, err := ekv.NewFilestore(baseDir, password)
	require.NoError(b, err)
	localStore := NewKVFilesystem(fs)
	require.NoError(b, err)

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// We expect the number of transactions to reach this checkpoint within
	// a few weeks.
	const numRandomTimestamps = 10000

	// Construct timestamps
	mockTimestamps := constructTimestamps(b, numRandomTimestamps)

	for i := 0; i < b.N; i++ {
		// Clear files so constructed transaction log does not load state
		require.NoError(b, os.RemoveAll(baseDir))

		// Construct new transaction log for benchmark iteration
		txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
			deviceSecret, rand.Reader)
		require.NoError(b, err)

		for cnt, curTs := range mockTimestamps {
			// Construct transaction
			key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
			newTx := NewTransaction(curTs, key, []byte(val))

			// Append and use insertion sort
			txLog.appendUsingQuickSort(newTx)

		}
	}
}
