////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/base64"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

const (
	// expectedTransactionLogSerializedBase64 is the base64 encoded serialized
	// TransactionLog. If the state set in the mock TransactionLog is changed,
	// this value should be changed to reflect this.
	expectedTransactionLogSerializedBase64 = `MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0VAAAAAAAAAFhYREtUWExPR0RWQ09GRlNUZTMwPQYAAAAAAAAAkgAAAAAAAAAwLEhKcEhGVnF0eDJvcVVvanZGbVBpVDBlR25EVEVyUDlmOFBOcC0wYVZ6c29RcnVQMXozQ2paU1VabXBSRDFXWmRjNGFRV1VaY25rTEFFV3RRcUZnU181cWI0YThQaTB4Yy04YUlzOHBKRXo1SnJKOVZKdzM2RWt1TzY0U0prOEQ2bzlURVhYM1V0ay1FVGYwcpIAAAAAAAAAMSxISnBIRlZxdHgyb3FVb2p2Rm1QaVQwZUduRFRFclA5ZnVQb19zdVBhaGdHSzZFQXhRZV9OeWRTMFlHdjJmcFJnRTg0YUdEY2RtbTkyQ3VhV0ZhUnJBZ2ZQWmNVYU41NXRLVTNHbEtoSkVfZTczamVud2dwNW5WemtYbUZvblMzckNNUVdQMUNHSUhwQncxb0GSAAAAAAAAADIsSEpwSEZWcXR4Mm9xVW9qdkZtUGlUMGVHbkRURXJQOWZwRm93UUdMYUpiVGNSNXIzSnY5QmJod1Y0eUozLTd1VlQzOXdIeU1XNTR2eldycjU0eUpTU2l1OF9GVUVMWlVuTDRMNTduc2lQdnp2TWZicE8wWWhEdEdDbHVweFdsZDJVdUxSWmNGWWxpQWFNSlJxkgAAAAAAAAAzLEhKcEhGVnF0eDJvcVVvanZGbVBpVDBlR25EVEVyUDlmLXdEbmMxd1dDWEZuVVcwRFB2Q3F2RngyVEpFYWpYbEd3S2JyclNZWWJhdm1hSmhIQVBNQTh2czZ2ak9CcjZfTFNuT1NvUlZKaFZfa24wQ3Z3RGt3NHAwTWNqWWJUajZfRi1PaXZXOG80X01ScUR6NZIAAAAAAAAANCxISnBIRlZxdHgyb3FVb2p2Rm1QaVQwZUduRFRFclA5ZkdWZXd0ay1BRElUdDhsSkZITE5PRGE1Z254RWZGaXhXYUg2TExTRldUbkMyZjY1WFhvdlJaS1BtMEZsb1ZocDhFX2ZCaUJscmNCUGl2blJ4cWxpc1lyNThmc3pjd0VoWnJRUlhYdkRTcmVJWnhFbm+SAAAAAAAAADUsSEpwSEZWcXR4Mm9xVW9qdkZtUGlUMGVHbkRURXJQOWZtN1ZBME44N1h4S2NLalFFTVBod1NpRjg1M2R3RThCaE1lN0c4NGF6dkdST05PSjctdDgxMWNlV0c2ZVAxRDRrYjduVmY4T0JKaG51OXlNbFBIZUVTeFFOZWN4RHUya3k1Q0o3NXNrRjhlR1RyRTJx`
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

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	// Construct transaction log
	txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
		deviceSecret, rngGen)
	require.NoError(t, err)

	// Construct expected transaction log object
	expected := &TransactionLog{
		path:               baseDir,
		local:              localStore,
		remote:             remoteStore,
		Header:             NewHeader(),
		txs:                make([]Transaction, 0),
		deviceSecret:       deviceSecret,
		rngStreamGenerator: rngGen,
		offsets:            make(deviceOffset, 0),
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

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	// Construct transaction log
	txLog, err := NewTransactionLog("baseDir", localStore,
		remoteStore,
		deviceSecret, rngGen)
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
		deviceSecret, rngGen)
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

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)

	// Construct transaction log
	txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
		deviceSecret, rngGen)
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
		path:               baseDir,
		local:              localStore,
		remote:             remoteStore,
		deviceSecret:       deviceSecret,
		rngStreamGenerator: txLog.rngStreamGenerator,
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

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	for i := 0; i < b.N; i++ {

		// Construct new transaction log for benchmark iteration
		txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
			deviceSecret, rngGen)
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

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	for i := 0; i < b.N; i++ {
		// Clear files so constructed transaction log does not load state
		require.NoError(b, os.RemoveAll(baseDir))

		// Construct new transaction log for benchmark iteration
		txLog, err := NewTransactionLog(baseDir, localStore, remoteStore,
			deviceSecret, rngGen)
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
