////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	kvsync "gitlab.com/elixxir/client/v4/sync"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	password = "password"
)

// TestRemoteKV uses a RemoteKV and shows that several
// different prefixes that are synched will sync all the keys and any
// not in the sync list will not. A separate test should add a prefix
// mid-way and show that the keys begin to sync after the prefix was
// added.
func TestRemoteKV(t *testing.T) {
	testKeys := []string{"hello", "how", "are", "you", "sync", "sync1",
		"1sync"}

	// Initialize KV
	// Construct mock update callback
	remoteCallCnt := 0
	txs := make(map[string][]byte)
	var lck sync.Mutex
	updateCb := kvsync.RemoteStoreCallback(func(newTx kvsync.Transaction,
		err error) {
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
	kv, err := kvsync.NewVersionedKV(txLog, ekv, nil, nil, updateCb)
	require.NoError(t, err)
	kv.SyncPrefix("bindings")
	newKV, err := kv.Prefix("bindings")
	rkv := &RemoteKV{
		rkv: newKV.(*kvsync.VersionedKV),
	}
	require.NoError(t, err)

	// There should be 1 tx per synchronized key
	txCnt := 0
	expTxs := make(map[string][]byte)
	for j := range testKeys {
		txCnt = txCnt + 1
		obj := &versioned.Object{
			Version:   0,
			Timestamp: time.Now(),
			Data:      []byte("WhatsUpDoc?"),
		}
		objJSON, err := json.Marshal(obj)
		require.NoError(t, err)
		rkv.Set(testKeys[j], objJSON)

		data, err := rkv.Get(testKeys[j], int64(obj.Version))
		require.NoError(t, err)
		require.Equal(t, objJSON, data)
	}

	ok := rkv.rkv.WaitForRemote(60 * time.Second)
	if !ok {
		t.Errorf("threads failed to stop")
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	}

	for k, v := range expTxs {
		storedV, ok := txs[k]
		require.True(t, ok, k)
		require.Equal(t, v, storedV)
	}

	require.Equal(t, txCnt, remoteCallCnt)
}

// TestRemoteKVMapFuncs tests getting and setting a map of versioned.Objects
func TestRemoteKVMapFuncs(t *testing.T) {
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
	updateCb := kvsync.RemoteStoreCallback(func(newTx kvsync.Transaction, err error) {
		lck.Lock()
		defer lck.Unlock()
		require.NoError(t, err)
		remoteCallCnt += 1
	})
	txLog := makeTransactionLog("remoteKV_TestMaps", password, t)
	ekv := ekv.MakeMemstore()
	kv, err := kvsync.NewVersionedKV(txLog, ekv, nil, nil, updateCb)
	require.NoError(t, err)
	kv.SyncPrefix("bindings")
	newKV, err := kv.Prefix("bindings")
	rkv := &RemoteKV{
		rkv: newKV.(*kvsync.VersionedKV),
	}
	require.NoError(t, err)

	mapKey := "mapkey"

	// An empty map shouldn't return an error
	_, err = rkv.GetMap(mapKey, 0)
	require.NoError(t, err)

	// A nonexistent map element should
	_, err = rkv.GetMapElement(mapKey, "blah", 0)
	require.Error(t, err)

	// Set & Get first, 1 element at a time
	for k, v := range first {
		vJSON, err := json.Marshal(v)
		require.NoError(t, err)
		err = rkv.StoreMapElement(mapKey, k, vJSON, 0)
		require.NoError(t, err)
		e, err := rkv.GetMapElement(mapKey, k, 0)
		require.NoError(t, err)
		require.Equal(t, vJSON, e)
	}
	newFirstJSON, err := rkv.GetMap(mapKey, 0)
	require.NoError(t, err)
	firstJSON, err := json.Marshal(first)
	require.NoError(t, err)
	require.Equal(t, firstJSON, newFirstJSON)

	// Overwrite with second
	secondJSON, err := json.Marshal(second)
	require.NoError(t, err)
	err = rkv.StoreMap(mapKey, secondJSON, 0)
	require.NoError(t, err)
	for k, v := range second {
		newVJSON, err := rkv.GetMapElement(mapKey, k, 0)
		require.NoError(t, err)
		vJSON, err := json.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, vJSON, newVJSON)
	}
	newSecondJSON, err := rkv.GetMap(mapKey, 0)
	require.NoError(t, err)
	require.Equal(t, secondJSON, newSecondJSON)
}

// makeTransactionLog is a utility function which generates a TransactionLog for
// testing purposes.
func makeTransactionLog(baseDir, password string, t *testing.T) *kvsync.TransactionLog {

	localStore := kvsync.NewKVFilesystem(ekv.MakeMemstore())
	// Construct remote store
	remoteStore := &mockRemote{data: make(map[string][]byte, 0)}

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := kvsync.NewTransactionLog(baseDir+"test.txt", localStore,
		remoteStore, deviceSecret, &CountingReader{count: 0})
	require.NoError(t, err)

	return txLog
}

// CountingReader is a platform-independent deterministic RNG that adheres to
// io.Reader.
type CountingReader struct {
	count uint8
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
}

type mockRemote struct {
	lck  sync.Mutex
	data map[string][]byte
}

func (m *mockRemote) Read(path string) ([]byte, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	return m.data[path], nil
}

func (m *mockRemote) Write(path string, data []byte) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.data[path] = append(m.data[path], data...)
	return nil
}

func (m *mockRemote) ReadDir(path string) ([]string, error) {
	panic("unimplemented")
}

func (m mockRemote) GetLastModified(path string) (time.Time, error) {
	return netTime.Now(), nil
}

func (m mockRemote) GetLastWrite() (time.Time, error) {
	return netTime.Now(), nil
}
