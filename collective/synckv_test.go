////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

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
// different prefixes that are synched will collective all the keys and any
// not in the collective list will not. A separate test should add a prefix
// mid-way and show that the keys begin to collective after the prefix was
// added.
func TestVersionedKV(t *testing.T) {
	syncPrefixes := []string{"collective", "a", "abcdefghijklmnop", "b", "c"}
	nonSyncPrefixes := []string{"hello", "sync1", "1sync", "synd", "ak"}

	testKeys := []string{"hello", "how", "are", "you", "collective", "sync1",
		"1sync"}

	testSyncPrefixes, testNoSyncPrefixes := genTestCases(syncPrefixes,
		nonSyncPrefixes, t)

	t.Logf("SYNCH: %+v", testSyncPrefixes)
	t.Logf("NOSYNCH: %+v", testNoSyncPrefixes)

	// Initialize KV
	// Construct mock update callback
	remoteCallCnt := 0
	txs := make(map[string]*versioned.Object)
	var lck sync.Mutex
	updateCb := versioned.KeyChangedByRemoteCallback(
		func(key string, oldVal, newVal *versioned.Object,
			op versioned.KeyOperation) {
			lck.Lock()
			defer lck.Unlock()
			require.Nil(t, oldVal)
			t.Logf("KEY: %s", key)
			remoteCallCnt += 1
			_, ok := txs[key]
			require.False(t, ok, key)
			txs[key] = newVal

		})
	memKV := ekv.MakeMemstore()
	remoteStore := newMockRemote()
	rkv, _ := testingKV(t, memKV, syncPrefixes, remoteStore,
		NewCountingReader())

	// Overwrite remote w/ non file IO option
	rkv.remote.txLog.io = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	var err error
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
			tkv.ListenOnRemoteKey(testKeys[j], 0, updateCb)
			tkv.Set(testKeys[j], obj)

			data, err := rkv.remote.GetBytes(tkv.GetFullKey(testKeys[j],
				obj.Version))
			require.NoError(t, err)
			require.Equal(t, obj.Marshal(), data)
		}
	}
	require.Equal(t, 0, remoteCallCnt)
	require.Equal(t, 0, len(rkv.remote.txLog.state.keys))

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
			data, err := rkv.remote.GetBytes(k)
			require.NoError(t, err)
			require.Equal(t, v, data)

			expTxs[k] = v
		}
	}

	for k, v := range expTxs {
		storedV, ok := txs[k]
		require.True(t, ok, k)
		require.Equal(t, v, storedV)
	}

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
	txs := make(map[string]*versioned.Object)
	updateCb := versioned.MapChangedByRemoteCallback(
		func(mapName string, edits map[string]versioned.ElementEdit) {
			lck.Lock()
			defer lck.Unlock()
			t.Logf("KEY: %s", mapName)
			for k := range edits {
				require.Nil(t, edits[k].OldElement)
				_, ok := txs[k]
				require.False(t, ok, k)
				txs[k] = edits[k].NewElement
			}
			remoteCallCnt += 1

		})
	memKV := ekv.MakeMemstore()
	remoteStore := newMockRemote()

	rkv, _ := testingKV(t, memKV, nil, remoteStore, NewCountingReader())

	// Overwrite remote w/ non file IO option
	rkv.remote.txLog.io = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	mapKey := "mapkey"

	rkv.ListenOnRemoteMap(mapKey, 0, updateCb)

	// An empty map shouldn't return an error
	_, err := rkv.GetMap(mapKey, 0)
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
