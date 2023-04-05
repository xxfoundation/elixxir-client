////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/randomness"
)

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
	rng := csprng.NewSystemRNG()
	for i := 0; i < 4; i++ {
		sizeOf := uint32(3)
		howMany := int(randomness.ReadRangeUint32(0, sizeOf, rng))
		s := int(i) % (len(syncPrefixes) - int(sizeOf))
		k := int(i) % (len(nonSyncPrefixes) - int(sizeOf))
		syncMe := make([]string, int(sizeOf))
		copy(syncMe, syncPrefixes[s:s+howMany])
		copy(syncMe[s+howMany:],
			nonSyncPrefixes[k:k+(int(sizeOf)-howMany)])
		dontSyncMe := make([]string, int(sizeOf))
		copy(dontSyncMe, nonSyncPrefixes[k:k+int(sizeOf)])
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

	// Initialize KV
	// Construct mock update callback
	remoteCallCnt := 0
	var lck sync.Mutex
	updateCb := RemoteStoreCallback(func(newTx Transaction, err error) {
		require.NoError(t, err)
		lck.Lock()
		defer lck.Unlock()
		remoteCallCnt += 1
	})
	txLog := makeTransactionLog("versionedKVWorkDir", password, t)
	ekv := ekv.MakeMemstore()
	rkv, err := NewOrLoadKV(txLog, ekv, syncPrefixes, nil, updateCb)
	require.NoError(t, err)
	// Overwrite remote w/ non file IO option
	rkv.txLog.remote = &mockRemote{
		data: make(map[string][]byte, 0),
	}

	kv := NewVersionedKV(rkv)

	// There should be 0 activity when working with non tracked prefixes
	for i := range testNoSyncPrefixes {
		var tkv versioned.KV
		for j := range testNoSyncPrefixes[i] {
			curP := testNoSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = kv.Prefix(curP)
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

			data, err := rkv.GetBytes(tkv.GetFullKey(testKeys[j],
				obj.Version))
			require.NoError(t, err)
			require.Equal(t, obj.Marshal(), data)
		}
	}
	require.Equal(t, 0, remoteCallCnt)
	require.Equal(t, 0, len(rkv.txLog.txs))

	// There should be 1 tx per synchronized key
	txCnt := 0
	for i := range testSyncPrefixes {
		var tkv versioned.KV
		for j := range testSyncPrefixes[i] {
			curP := testSyncPrefixes[i][j]
			if tkv == nil {
				tkv, err = kv.Prefix(curP)
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

			data, err := rkv.GetBytes(tkv.GetFullKey(testKeys[j],
				obj.Version))
			require.NoError(t, err)
			require.Equal(t, obj.Marshal(), data)
		}
	}

	// NOTE: need a better way to detect when remote writes are done...
	time.Sleep(1 * time.Second)
	require.Equal(t, txCnt, remoteCallCnt)

}

// TestVersionedKVNewPrefix adds a prefix mid-way and show that the
// keys begin to sync after the prefix was added.
func TestVersionedKVNewPrefix(t *testing.T) {
}
