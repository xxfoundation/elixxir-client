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
	expectedTransactionLogSerializedBase64 = `WFhES1RYTE9HSERSZXlKMlpYSnphVzl1SWpvd0xDSmxiblJ5YVdWeklqcDdmWDA9MCxBUUlEQkFVR0J3Z0pDZ3NNRFE0UEVCRVNFeFFWRmhjWTBjY2dhSHE2QkJDRXlVMmRFNHk2a1llWV9UalNKdGZycTh4U0s2SlBWYzlVemRYTWo3SHJINzUwb1NqZVkyVzhaeXdPYVM0aDU5Rk9IcG5qaUhIc3BqQl9DeVRpRWNES3dsb1puaTFybU44OVJOeFAxLEdSb2JIQjBlSHlBaElpTWtKU1luS0NrcUt5d3RMaTh3ZXA0N2ZrVTc3QUIyY2JKcnhZNlZWUm1RcVl4Qzc1Rmd1dEpGNE1jX2NpcDg3NHFiOEIyUjFJa3lncHJoZE5YeVlHM3dxS2N6dkxNYUtOSFVXVXV0RXZKNWd2clRLbWhDdEJJenRxbmh4SHBQZHNmbDIsTVRJek5EVTJOemc1T2pzOFBUNF9RRUZDUTBSRlJrZElVTjI4MFRHNFVNcjQzSFRZU0c4RUNFZE0tNURjZUpRcU03MFktT2dSRnYtdVMwRlRQVHkzTWM5b3FScl9BNW56YWpCQWZLUEwwcTlKcmc5QVBCdjZtVWhKblZZMUllTEc0ekFQSjR5YmhWcThOX3MyMyxTVXBMVEUxT1QxQlJVbE5VVlZaWFdGbGFXMXhkWGw5Z3d4VEVsTmdvTlkwQk1zcWQzLXoweTRyYndXa1RPTVpYSFg4WWhmQUdiX1VrQUphbWxNeDlIZTR2blhSVnZrQ2JmQ1hLLUprejA3RUJHV29HZm83b1YtZFdHdFhxbkRTdUZUdy1KOTJUMkNwZG4wUXY0LFlXSmpaR1ZtWjJocGFtdHNiVzV2Y0hGeWMzUjFkbmQ0WWZSZzdZM2JUdmk5MGdFUUFSU1ByMmhhQjZFMDNZM1I3VTRMZ3NPZEFQcmwwMWRkN0o4RDdlWi1DWDVhbGlVZlY4MnJ0TnMyaTFxSmN4dVIzOFM1X1BISE5FcVJGcld1YzFpbUxnRDMwTkxadG5fQzUsZVhwN2ZIMS1mNENCZ29PRWhZYUhpSW1LaTR5TmpvLVF1bzREeTc4Z0xTUTBuMlpaU2E4SVZTQ1dsc2I5dDhFT2JsMjJxTHJYUXJPQTlGN0FsNmU0QmF5NVdKa3d1SGFSSG5STkd6UnZQWkNnYno3YnJUbVNMR3hNc2ZIdi00VTBsY3Nrdk1mR3hYcnJsOHNn`
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
	txLog := NewTransactionLog(localStore, remoteStore, rand.Reader,
		baseDir, deviceSecret)
	txLog.SetHeader(hdr)

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
	txLog := NewTransactionLog(localStore, remoteStore, rand.Reader,
		baseDir, deviceSecret)
	txLog.SetHeader(hdr)

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
	txLog := NewTransactionLog(localStore, remoteStore,
		&CountingReader{count: 0}, baseDir, deviceSecret)
	txLog.SetHeader(hdr)

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
	txLog := NewTransactionLog(localStore, remoteStore,
		&CountingReader{count: 0}, baseDir+"test.txt", deviceSecret)
	txLog.SetHeader(hdr)

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
