////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"sort"
	"strconv"
	"testing"
)

const (
	// expectedTransactionLogSerializedBase64 is the base64 encoded serialized
	// TransactionLog. If the state set in the mock TransactionLog is changed,
	// this value should be changed to reflect this.
	expectedTransactionLogSerializedBase64 = `WFhES1RYTE9HSERSZXlKMlpYSnphVzl1SWpvd0xDSmxiblJ5YVdWeklqcDdmWDA9MCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRQNGgycHEzWGVrT0tGX0YyeU1oTEY3bE9VaFYwb204Vm9kTXd2eU5abXpyM1RSOENVS2FuWFBTNTUzbUNaUHNFMVNqdDVaRGpIS21pT2FPZ1NsZC1YVl9GdEg2QTVvVGVsM0JaR1dhYXA1YXdKeG8xLEdSb2JIQjBlSHlBaElpTWtKU1luS0NrcUt5d3RMaTh3NnRzd09LRTlCZEZIVS1DT1FYVmp4WjhTNWt4MFJMbWk5ZVNZb0VfVUg1d0hXcnlmYm5VcF95TlRqVjV4bXdNMHZhdG9xOUxYclQ0V0VsSEZuOWstM0pUb1QyQWxReUdYZFhWRjR2Y3V5ckJEaW5QQjIsTVRJek5EVTJOemc1T2pzOFBUNF9RRUZDUTBSRlJrZElZeFkyUk5QRlN0OXI0NHJOS0JJUksxOVZMWlROOWsxN2pFOFM4LVRUWENrMXRtbm55bmlhZ2h3enB0M1cwTVQ5SFMwSUMwaGRxY3RNSGtIY1hDeWZmb2JzLXk0ajRTSlgwTE4yR0VpZ3N6OEw0MXBqMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z3pKVDdVaGZ5Q0FVNEhnamEzS0RURm5pSlR2YzNQQzRLdkx4TGR1Vmt0eDlOOHNPR2x1RFEySFUyZzFKcndBMm9TNkxyQWM1dzhwTDVQanhHNzY2ak1KX3JpeDYtRGtINVhDWi12aVB5NTVZbXlnRzI0LFlXSmpaR1ZtWjJocGFtdHNiVzV2Y0hGeWMzUjFkbmQ0RV9yeU1yXzl0VDBPQkx5WVZtd2g5bnRmcW1JRHlfWEltcE56MTZxa05MVHRZbzVMN21vWXVFRnNHZ0RSMUJIc29NTm9pdTQ3c2VtMGk5QWQ4UWNlWW1yWmVxS3JjUVZKWFVXZHl5Uk5hX082bC10YTUsZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LaTR5TmpvLVFJTjFEWURNZzZMZElWWjFPTkRJTTdGekhhdC1nbmlIS01JN04yajBlQndXeDlZeUlwQVFyRzBJdHVEQmYtRHoyUVdFalpYR0JfSC1fR3FINEpOellnZnE5eTFRRTdnRWdsZWpuOVpPc21WNGVTOUNx`
)

// Smoke test for NewTransactionLog.
func TestNewTransactionLog(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir", "password"
	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct header
	hdr := NewHeader()

	// Construct log pth

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr, rand.Reader,
		baseDir, deviceSecret)

	// Construct expected transaction log object
	expected := &TransactionLog{
		path:         baseDir,
		local:        localStore,
		remote:       remoteStore,
		hdr:          hdr,
		txs:          make([]Transaction, 0),
		curBuf:       &bytes.Buffer{},
		deviceSecret: deviceSecret,
		rng:          rand.Reader,
	}

	// Ensure constructor generates expected object
	require.Equal(t, expected, txLog)

}

// Tests that TransactionLog's append function will insert new Transaction's
// into the TransactionLog, and that the transactions are sorted by timestamp
// after the insertion.
func TestTransactionLog_Append(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir", "password"
	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct header
	hdr := NewHeader()

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr, rand.Reader,
		baseDir, deviceSecret)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t)

	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		txLog.append(newTx)

		// Ensure that these transactions have been inserted in order for each
		// insertion
		require.True(t, sort.SliceIsSorted(txLog.txs, func(i, j int) bool {
			firstTs, secondTs := txLog.txs[i].Timestamp, txLog.txs[j].Timestamp
			return firstTs.Before(secondTs)
		}))
	}

}

// Tests that TransactionLog's serialize function returns the serialized
// internal state. Checks against a hardcoded base64 string.
func TestTransactionLog_Serialize(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir", "password"
	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct header
	hdr := NewHeader()

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr,
		&CountingReader{count: 0}, baseDir, deviceSecret)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t)

	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction
		txLog.append(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Encode data to bas64
	data64 := base64.StdEncoding.EncodeToString(data)

	// Ensure encoded data using mock values matches hardcoded data.
	require.Equal(t, expectedTransactionLogSerializedBase64, data64)
}

// Tests that TransactionLog's save function writes to remote and local stores
// when they are set.
func TestTransactionLog_Save(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir/", "password"
	require.NoError(t, utils.MakeDirs(baseDir, utils.DirPerms))

	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct header
	hdr := NewHeader()

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr,
		&CountingReader{count: 0}, baseDir+"test.txt", deviceSecret)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t)

	// Insert mock data into transaction log
	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		// Insert transaction
		txLog.append(newTx)
	}

	// Serialize data
	data, err := txLog.serialize()
	require.NoError(t, err)

	// Write data to remote & local
	err = txLog.save(data)
	require.NoError(t, err)

	// Read from remote
	dataFromRemote, err := txLog.remote.Read(txLog.path)
	require.NoError(t, err)

	// Ensure read data from remote matches originally written
	require.Equal(t, data, dataFromRemote)

	// Read from local
	dataFromLocal, err := txLog.remote.Read(txLog.path)
	require.NoError(t, err)

	// Ensure read data from local matches originally written
	require.Equal(t, data, dataFromLocal)
}
