////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/randomness"
)

// TestVersionedKV uses a RemoteKV and shows that several
// different prefixes that are synched will sync all the keys and any
// not in the sync list will not. A separate test should add a prefix
// mid-way and show that the keys begin to sync after the prefix was
// added.
func TestVersionedKV(t *testing.T) {
	syncPrefixes := []string{"sync", "a", "abcdefghijklmnop", "b", "c"}
	nonSyncPrefixes := []string{"hello", "sync1", "1sync", "synd", "ak"}

	testKeys := []string{"hello", "how", "are", "you", "sync", "sync1",
		"1sync"}

	testSyncPrefixes, testNoSyncPrefixes := genTestCases(syncPrefixes,
		nonSyncPrefixes, t)

	t.Logf("SYNCH: %+v", testSyncPrefixes)
	t.Logf("NOSYNCH: %+v", testNoSyncPrefixes)

	// Initialize KV
	// Construct mock update callback
	remoteCallCnt := 0
	txs := make(map[string][]byte)
	var lck sync.Mutex
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		lck.Lock()
		defer lck.Unlock()
		require.NoError(t, err)
		remoteCallCnt += 1
		_, ok := txs[newTx.Key]
		require.False(t, ok, newTx.Key)
		txs[newTx.Key] = newTx.Value
	})
	txLog := makeTransactionLog("versionedKV_TestWorkDir", password, t)
	ekv := ekv.MakeMemstore()
	rkv, err := NewVersionedKV(txLog, ekv, syncPrefixes, nil, updateCb)
	require.NoError(t, err)
	// Overwrite remote w/ non file IO option
	rkv.remoteKV.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	// There should be 0 activity when working with non tracked prefixes
	for i := range testNoSyncPrefixes {
		var tkv versioned.KV
		for j := range testNoSyncPrefixes[i] {
			curP := testNoSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = rkv.Prefix(curP)
			} else {
				tkv, err = tkv.Prefix(curP)
			}
			require.NoError(t, err, curP)
		}
		for j := range syncPrefixes {
			require.False(t, tkv.HasPrefix(syncPrefixes[j]),
				syncPrefixes[j])
		}
		for j := range testKeys {
			obj := &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      []byte("WhatsUpDoc?"),
			}
			tkv.Set(testKeys[j], obj)

			data, err := rkv.remoteKV.GetBytes(tkv.GetFullKey(testKeys[j],
				obj.Version))
			require.NoError(t, err)
			require.Equal(t, obj.Marshal(), data)
		}
	}
	require.Equal(t, 0, remoteCallCnt)
	require.Equal(t, 0, len(rkv.remoteKV.txLog.txs))

	// There should be 1 tx per synchronized key
	txCnt := 0
	expTxs := make(map[string][]byte)
	for i := range testSyncPrefixes {
		var tkv versioned.KV
		for j := range testSyncPrefixes[i] {
			curP := testSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = rkv.Prefix(curP)
			} else {
				tkv, err = tkv.Prefix(curP)
			}
			require.NoError(t, err, curP)
		}
		for j := range testSyncPrefixes[i] {
			require.True(t, tkv.HasPrefix(testSyncPrefixes[i][j]))
		}
		for j := range testKeys {
			txCnt = txCnt + 1
			obj := &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      []byte("WhatsUpDoc?"),
			}
			tkv.Set(testKeys[j], obj)

			k := tkv.GetFullKey(testKeys[j], obj.Version)
			v := obj.Marshal()
			data, err := rkv.remoteKV.GetBytes(k)
			require.NoError(t, err)
			require.Equal(t, v, data)

			expTxs[k] = v
		}
	}

	rkv.remoteKV.WaitForRemote(30 * time.Second)

	for k, v := range expTxs {
		storedV, ok := txs[k]
		require.True(t, ok, k)
		require.Equal(t, v, storedV)
	}

	require.Equal(t, txCnt, remoteCallCnt)
}

// TestVersionedKVNewPrefix adds a prefix mid-way and show that the
// keys begin to sync after the prefix was added.
func TestVersionedKVNewPrefix(t *testing.T) {
	syncPrefixes := []string{"sync", "a", "abcdefghijklmnop", "b", "c"}
	nonSyncPrefixes := []string{"hello", "sync1", "1sync", "synd", "ak"}

	testKeys := []string{"hello", "how", "are", "you", "sync", "sync1",
		"1sync"}

	testSyncPrefixes, _ := genTestCases(syncPrefixes,
		nonSyncPrefixes, t)

	// Initialize KV
	// Construct mock update callback
	remoteCallCnt := 0
	txs := make(map[string][]byte)
	var lck sync.Mutex
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		lck.Lock()
		defer lck.Unlock()
		require.NoError(t, err)
		t.Logf("KEY: %s", newTx.Key)
		remoteCallCnt += 1
		_, ok := txs[newTx.Key]
		require.False(t, ok, newTx.Key)
		txs[newTx.Key] = newTx.Value
	})
	txLog := makeTransactionLog("versionedKV_TestNewPrefix", password, t)
	ekv := ekv.MakeMemstore()
	rkv, err := NewVersionedKV(txLog, ekv, nil, nil, updateCb)
	require.NoError(t, err)
	// Overwrite remote w/ non file IO option
	rkv.remoteKV.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	// Even these are all "sync prefixes" there should be 0 of
	// them because they aren't tracked.
	for i := range testSyncPrefixes {
		var tkv versioned.KV
		for j := range testSyncPrefixes[i] {
			curP := testSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = rkv.Prefix(curP)
			} else {
				tkv, err = tkv.Prefix(curP)
			}
			require.NoError(t, err, curP)
		}
		for j := range testSyncPrefixes[i] {
			require.True(t, tkv.HasPrefix(testSyncPrefixes[i][j]))
		}
		for j := range testKeys {
			obj := &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      []byte("WhatsUpDoc?"),
			}
			tkv.Set(testKeys[j], obj)

			data, err := rkv.remoteKV.GetBytes(
				tkv.GetFullKey(testKeys[j], obj.Version))
			require.NoError(t, err)
			require.Equal(t, obj.Marshal(), data)
		}
	}
	require.Equal(t, 0, remoteCallCnt)
	require.Equal(t, 0, len(rkv.remoteKV.txLog.txs))

	// Add the sync prefixes
	for i := range syncPrefixes {
		rkv.SyncPrefix(syncPrefixes[i])
	}
	require.Equal(t, syncPrefixes, rkv.synchronizedPrefixes)

	// Now there should be 1 tx per synchronized key
	txCnt := 0
	expTxs := make(map[string][]byte)
	for i := range testSyncPrefixes {
		var tkv versioned.KV
		for j := range testSyncPrefixes[i] {
			curP := testSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = rkv.Prefix(curP)
			} else {
				tkv, err = tkv.Prefix(curP)
			}
			require.NoError(t, err, curP)
		}
		for j := range testSyncPrefixes[i] {
			require.True(t, tkv.HasPrefix(testSyncPrefixes[i][j]))
		}
		for j := range testKeys {
			txCnt = txCnt + 1
			obj := &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      []byte("WhatsUpDoc?"),
			}
			tkv.Set(testKeys[j], obj)

			k := tkv.GetFullKey(testKeys[j], obj.Version)
			v := obj.Marshal()
			data, err := rkv.remoteKV.GetBytes(k)
			require.NoError(t, err)
			require.Equal(t, v, data)

			expTxs[k] = v
		}
	}

	rkv.remoteKV.WaitForRemote(30 * time.Second)

	for k, v := range expTxs {
		storedV, ok := txs[k]
		require.True(t, ok, k)
		require.Equal(t, v, storedV)
	}

	// NOTE: need a better way to detect when remote writes are done...
	require.Equal(t, txCnt, remoteCallCnt)

}

// TestVersionedKVMapFuncs tests getting and setting a map of versioned.Objects
func TestVersionedKVMapFuncs(t *testing.T) {
	// generate a map of bytes 1->100
	first := make(map[string]*versioned.Object)
	for i := 0; i < 100; i++ {
		k := fmt.Sprintf("%d", i)
		v := versioned.Object{
			Timestamp: time.Now(),
			Version:   0,
			Data:      []byte{byte(i)},
		}
		first[k] = &v
	}

	// This has half the elements, every other one above
	second := make(map[string]*versioned.Object)
	for i := 0; i < 100; i += 2 {
		k := fmt.Sprintf("%d", i)
		second[k] = first[k]
	}

	// Create KV
	// Construct mock update callback
	remoteCallCnt := 0
	var lck sync.Mutex
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		lck.Lock()
		defer lck.Unlock()
		require.NoError(t, err)
		t.Logf("KEY: %s", newTx.Key)
		remoteCallCnt += 1
	})
	txLog := makeTransactionLog("versionedKV_TestMaps", password, t)
	ekv := ekv.MakeMemstore()
	rkv, err := NewVersionedKV(txLog, ekv, nil, nil, updateCb)
	require.NoError(t, err)
	// Overwrite remote w/ non file IO option
	rkv.remoteKV.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	mapKey := "mapkey"

	// An empty map shouldn't return an error
	_, err = rkv.GetMap(mapKey, 0)
	require.NoError(t, err)

	// A nonexistent map element should
	_, err = rkv.GetMapElement(mapKey, "blah", 0)
	require.Error(t, err)

	// Set & Get first, 1 element at a time
	for k, v := range first {
		err = rkv.StoreMapElement(mapKey, k, v, 0)
		require.NoError(t, err)
		e, err := rkv.GetMapElement(mapKey, k, 0)
		require.NoError(t, err)
		require.Equal(t, v, e)
	}
	newFirst, err := rkv.GetMap(mapKey, 0)
	require.NoError(t, err)
	require.Equal(t, first, newFirst)

	// Overwrite with second
	err = rkv.StoreMap(mapKey, second, 0)
	require.NoError(t, err)
	for k, v := range second {
		newV, err := rkv.GetMapElement(mapKey, k, 0)
		require.NoError(t, err)
		require.Equal(t, v, newV)
	}
	newSecond, err := rkv.GetMap(mapKey, 0)
	require.NoError(t, err)
	require.Equal(t, second, newSecond)
}

func genTestCases(syncPrefixes, nonSyncPrefixes []string, t *testing.T) (sync,
	nosync [][]string) {

	// Generate test cases, base cases:
	testSyncPrefixes := make([][]string, 0)
	testNoSyncPrefixes := make([][]string, 0)
	// simple
	for i := range syncPrefixes {
		testSyncPrefixes = append(testSyncPrefixes,
			[]string{syncPrefixes[i]})
	}
	for i := range nonSyncPrefixes {
		testNoSyncPrefixes = append(testNoSyncPrefixes,
			[]string{nonSyncPrefixes[i]})
	}

	// 2-level
	for i := range syncPrefixes {
		for j := range nonSyncPrefixes {
			if i%2 == 0 {
				testSyncPrefixes = append(testSyncPrefixes,
					[]string{syncPrefixes[i],
						nonSyncPrefixes[j]})
			} else {
				testSyncPrefixes = append(testSyncPrefixes,
					[]string{nonSyncPrefixes[j],
						syncPrefixes[i]})
			}
		}
	}
	for i := range nonSyncPrefixes {
		testNoSyncPrefixes = append(testNoSyncPrefixes,
			[]string{nonSyncPrefixes[i]})
	}

	// multilevel
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 4; i++ {
		sizeOf := uint32(3)
		howMany := int(randomness.ReadRangeUint32(1, sizeOf, rng))

		syncMe := make([]string, sizeOf)
		dontSyncMe := make([]string, sizeOf)

		for j := 0; j < howMany; j++ {
			offset := (i + j) % int(sizeOf)
			syncMe[j] = syncPrefixes[offset]
		}
		numLeft := int(sizeOf) - howMany
		for j := 0; j < int(sizeOf); j++ {
			offset := (i + j) % int(sizeOf)
			if j < numLeft {
				dOff := (j + howMany) % int(sizeOf)
				syncMe[dOff] = nonSyncPrefixes[offset]
			}
			dontSyncMe[j] = nonSyncPrefixes[offset]
		}

		seed := make([]byte, 32)
		rng.Read(seed)
		shuffle.ShuffleSwap(seed, int(sizeOf), func(i, j int) {
			tmp := syncMe[i]
			syncMe[i] = syncMe[j]
			syncMe[j] = tmp
		})
		rng.Read(seed)
		shuffle.ShuffleSwap(seed, int(sizeOf), func(i, j int) {
			tmp := dontSyncMe[i]
			dontSyncMe[i] = dontSyncMe[j]
			dontSyncMe[j] = tmp
		})

		testSyncPrefixes = append(testSyncPrefixes, syncMe)
		testNoSyncPrefixes = append(testNoSyncPrefixes, dontSyncMe)
	}

	sync = testSyncPrefixes
	nosync = testNoSyncPrefixes

	return sync, nosync
}
