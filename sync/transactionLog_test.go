////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"github.com/stretchr/testify/require"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
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
	remoteStore := NewFileSystemRemoteStorage()

	// Construct header
	hdr := NewHeader()

	// Construct log pth
	logPath := baseDir + "/test.log"

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr, rand.Reader,
		logPath, deviceSecret)

	// Construct reader/writer for the buffer
	var (
		writer = bufio.NewWriter(&bytes.Buffer{})
		reader = bufio.NewReader(bytes.NewReader([]byte{}))
	)

	// Construct expected transaction log object
	expected := &TransactionLog{
		path:         logPath,
		local:        localStore,
		remote:       remoteStore,
		hdr:          hdr,
		txs:          make([]Transaction, 0),
		curBuf:       bufio.NewReadWriter(reader, writer),
		deviceSecret: deviceSecret,
		rng:          rand.Reader,
	}

	// Ensure constructor generates expected object
	require.Equal(t, expected, txLog)

}

func TestTransactionLog_Insert(t *testing.T) {
	// Construct local store
	baseDir, password := "testDir", "password"
	localStore, err := NewEkvLocalStore(baseDir, password)
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()
	
	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage()

	// Construct header
	hdr := NewHeader()

	// Construct log pth
	logPath := baseDir + "/test.log"

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog := NewTransactionLog(localStore, remoteStore, hdr, rand.Reader,
		logPath, deviceSecret)

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

func TestTransactionLog_Serialize(t *testing.T) {
}

// constructTimestamps is a testing utility function. It constructs a list of
// out-of order mock timestamps.
func constructTimestamps(t *testing.T) []time.Time {
	var (
		timestamp0, timestamp1, timestamp2, timestamp3, timestamp4,
		timestamp5 time.Time
		err error
	)

	// Construct timestamps. All of these are the same date but with different
	// years.
	timestamp0, err = time.Parse(time.RFC3339,
		"2015-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp1, err = time.Parse(time.RFC3339,
		"2013-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp2, err = time.Parse(time.RFC3339,
		"2003-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp3, err = time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp4, err = time.Parse(time.RFC3339,
		"2014-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp5, err = time.Parse(time.RFC3339,
		"2001-12-21T22:08:41+00:00")
	require.NoError(t, err)

	return []time.Time{
		timestamp0, timestamp1, timestamp2, timestamp3, timestamp4, timestamp5,
	}
}
