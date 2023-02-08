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
	expectedTransactionLogSerializedBase64 = `MAAAAAAAAABYWERLVFhMT0dIRFJleUoyWlhKemFXOXVJam93TENKbGJuUnlhV1Z6SWpwN2ZYMD0GAAAAAAAAAJIAAAAAAAAAMCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWWRial9DNnlFc3FuTUk4LXVSNUJlVFBaeDZVSXZiSVV5c1FtTTNlbXVuSmN3OWVJYktpeVNwN2pWYWYxdTZlS2cxQWF0WkhxS0FvTnJ6aWRYaUtsY21uU0FsRWh3U1hzdENBN0llQnRteWxMeDExOG2SAAAAAAAAADEsR1JvYkhCMGVIeUFoSWlNa0pTWW5LQ2txS3l3dExpOHdDRUdqWjdKSG1JY3d4MG9oZ2xwVzhORW1mZzdUWll1SlBaUnR3alFPaXl1cTlZaFRlR3lEaVFsRHJ3a20tbmpXZk1YTE1sNm1WTmRyZGhPdV8zMWg5S3hJYlRDd25VdjVleXMydWEtTzFwQlhuZHBWkgAAAAAAAAAyLE1USXpORFUyTnpnNU9qczhQVDRfUUVGQ1EwUkZSa2RJSElDQUlFQ29kbXlvVTdsbGJhaFkxSzVvNGRVTWJ3SVZ6VFVDV09Benp2VmRNLVptUm1FcThqVk1FbHpFbWlpTzJaSkhLVmdISGk3aFBQSkxWa0xncThPTTBRbjdkMjlTd0o1X0lvTEhXUkZHUGJuUpIAAAAAAAAAMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z0dZdnNuWVd6X3lDSFV4Z1J0MXVWT1UtaWRxMk1xdm1pWF9PdlBaWHBjbmRabzFHVTBIM0RQeW5LRm9hRFNrekwzbmF6Y3JiMzk3a05mWTJPYm9qRDNqbkhieVlmZ28yZTNRS2pBZFpfcm4tWjVfNjmSAAAAAAAAADQsWVdKalpHVm1aMmhwYW10c2JXNXZjSEZ5YzNSMWRuZDRtQlJLeE5HeXlpQTFzRlMzOUZxRUlxWmVIeXVaQWIwSHNydF9QTzBYZF90RHlfeENiUTZ6Z0hhblljSXU5eWFST0xfUXFjaFhsejRJNkFoTjYwM3pEMVhVTGVTWXlPNy1kTnlIQm94STkzMllMNmhokgAAAAAAAAA1LGVYcDdmSDEtZjRDQmdvT0VoWWFIaUltS2k0eU5qby1RN1Myb2pvQ2QtRWRHUE55d1Utd2pzUkNITzV1V0lmZmNsTDhaaGFLOHk0WldsdEtWbFVtMU9QUjhiYkFXdXNLRFdZWVJUS3ZmSkZXRzRYYTNFWDFVWlJLQ1Zva1lUNmIzSkdVUW02cW01cEZoSDhZSQ==`
)

// Smoke test for NewOrLoadTransactionLog.
func TestNewOrLoadTransactionLog(t *testing.T) {
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

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore, rand.Reader,
		baseDir, deviceSecret)
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
	}

	// Ensure constructor generates expected object
	require.Equal(t, expected, txLog)

}

// Tests that NewOrLoadTransactionLog will load from local and deserialize
// the data into the TransactionLog file.
func TestNewOrLoadTransactionLog_Loading(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir/", "password"
	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))
	}()

	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore, rand.Reader,
		baseDir, deviceSecret)
	require.NoError(t, err)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t)

	for cnt, curTs := range mockTimestamps {
		// Construct transaction
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewTransaction(curTs, key, []byte(val))

		require.NoError(t, txLog.Append(newTx))
	}

	//newLocalStore, err := NewEkvLocalStore(baseDir, password)
	//require.NoError(t, err)

	newTxLog, err := NewOrLoadTransactionLog(localStore, remoteStore,
		rand.Reader, baseDir, deviceSecret)
	require.NoError(t, err)

	require.Equal(t, txLog, newTxLog)

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

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore, rand.Reader,
		baseDir, deviceSecret)
	require.NoError(t, err)

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

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore,
		&CountingReader{count: 0}, baseDir, deviceSecret)
	require.NoError(t, err)

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

// Unit test of TransactionLog.deserialize. Ensures that deserialize will
// construct the same TransactionLog that was serialized using
// TransactionLog.serialize.
func TestTransactionLog_Deserialize(t *testing.T) {
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

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore,
		&CountingReader{count: 0}, baseDir, deviceSecret)
	require.NoError(t, err)

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

	// Construct a log w/o header and transaction list
	newTxLog := &TransactionLog{
		path:         baseDir,
		local:        localStore,
		remote:       remoteStore,
		deviceSecret: deviceSecret,
		rng:          txLog.rng,
	}

	// Deserialize the transaction log
	err = newTxLog.deserialize(data)
	require.NoError(t, err)

	// Ensure deserialized object matches original object
	require.Equal(t, txLog, newTxLog)
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

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(localStore, remoteStore,
		&CountingReader{count: 0}, baseDir+"test.txt", deviceSecret)
	require.NoError(t, err)

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
