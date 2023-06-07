////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	password = "password"
)

func TestGetChannelInfo(t *testing.T) {

	pings := make([]ed25519.PublicKey, 3)
	rng := rand.New(rand.NewSource(43535))

	for i := range pings {
		pings[i], _, _ = ed25519.GenerateKey(rng)
	}

	data, _ := json.MarshalIndent(pings, "", "  ")
	fmt.Printf("%s\n", data)
}

// TestRemoteKV uses a RemoteKV and shows that several
// different prefixes that are synched will collective all the keys and any
// not in the collective list will not. A separate test should add a prefix
// mid-way and show that the keys begin to collective after the prefix was
// added.
// func TestRemoteKV(t *testing.T) {
// 	testKeys := []string{"hello", "how", "are", "you", "collective", "sync1",
// 		"1sync"}

// 	// Initialize KV
// 	// Construct mock update callback
// 	remoteCallCnt := 0
// 	txs := make(map[string][]byte)
// 	txLog := makeTransactionLog("versionedKV_TestWorkDir", password, t)
// 	ekv := ekv.MakeMemstore()
// 	kv, err := kvsync.newVersionedKV(txLog, ekv, nil, nil, updateCb)
// 	require.NoError(t, err)
// 	kv.SyncPrefix("bindings")
// 	newKV, err := kv.Prefix("bindings")
// 	rkv := &RemoteKV{
// 		rkv: newKV.(*kvsync.versionedKV),
// 	}
// 	require.NoError(t, err)

// 	// There should be 1 tx per synchronized key
// 	txCnt := 0
// 	expTxs := make(map[string][]byte)
// 	for j := range testKeys {
// 		txCnt = txCnt + 1
// 		obj := &versioned.Object{
// 			Version:   0,
// 			Timestamp: time.Now(),
// 			Data:      []byte("WhatsUpDoc?"),
// 		}
// 		objJSON, err := json.Marshal(obj)
// 		require.NoError(t, err)
// 		rkv.Set(testKeys[j], objJSON)

// 		data, err := rkv.Get(testKeys[j], int64(obj.Version))
// 		require.NoError(t, err)
// 		require.Equal(t, objJSON, data)
// 	}

// 	ok := rkv.rkv.WaitForRemote(60 * time.Second)
// 	if !ok {
// 		t.Errorf("threads failed to stop")
// 		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
// 	}

// 	for k, v := range expTxs {
// 		storedV, ok := txs[k]
// 		require.True(t, ok, k)
// 		require.Equal(t, v, storedV)
// 	}

// 	require.Equal(t, txCnt, remoteCallCnt)
// }

// // TestRemoteKVMapFuncs tests getting and setting a map of versioned.Objects
// func TestRemoteKVMapFuncs(t *testing.T) {
// 	// generate a map of bytes 1->100
// 	first := make(map[string]*versioned.Object)
// 	for i := 0; i < 100; i++ {
// 		k := fmt.Sprintf("%d", i)
// 		v := versioned.Object{
// 			Timestamp: time.Now(),
// 			Version:   0,
// 			Data:      []byte{byte(i)},
// 		}
// 		first[k] = &v
// 	}

// 	// This has half the elements, every other one above
// 	second := make(map[string]*versioned.Object)
// 	for i := 0; i < 100; i += 2 {
// 		k := fmt.Sprintf("%d", i)
// 		second[k] = first[k]
// 	}

// 	// Create KV
// 	// Construct mock update callback
// 	remoteCallCnt := 0
// 	var lck sync.Mutex
// 	updateCb := kvsync.RemoteStoreCallback(func(newTx kvsync.Mutate, err error) {
// 		lck.Lock()
// 		defer lck.Unlock()
// 		require.NoError(t, err)
// 		remoteCallCnt += 1
// 	})
// 	txLog := makeTransactionLog("remoteKV_TestMaps", password, t)
// 	ekv := ekv.MakeMemstore()
// 	kv, err := kvsync.newVersionedKV(txLog, ekv, nil, nil, updateCb)
// 	require.NoError(t, err)
// 	kv.SyncPrefix("bindings")
// 	newKV, err := kv.Prefix("bindings")
// 	rkv := &RemoteKV{
// 		rkv: newKV.(*kvsync.versionedKV),
// 	}
// 	require.NoError(t, err)

// 	mapKey := "mapkey"

// 	// An empty map shouldn't return an error
// 	_, err = rkv.GetMap(mapKey, 0)
// 	require.NoError(t, err)

// 	// A nonexistent map element should
// 	_, err = rkv.GetMapElement(mapKey, "blah", 0)
// 	require.Error(t, err)

// 	// Set & Get first, 1 element at a time
// 	for k, v := range first {
// 		vJSON, err := json.Marshal(v)
// 		require.NoError(t, err)
// 		err = rkv.StoreMapElement(mapKey, k, vJSON, 0)
// 		require.NoError(t, err)
// 		e, err := rkv.GetMapElement(mapKey, k, 0)
// 		require.NoError(t, err)
// 		require.Equal(t, vJSON, e)
// 	}
// 	newFirstJSON, err := rkv.GetMap(mapKey, 0)
// 	require.NoError(t, err)
// 	firstJSON, err := json.Marshal(first)
// 	require.NoError(t, err)
// 	require.Equal(t, firstJSON, newFirstJSON)

// 	// Overwrite with second
// 	secondJSON, err := json.Marshal(second)
// 	require.NoError(t, err)
// 	err = rkv.StoreMap(mapKey, secondJSON, 0)
// 	require.NoError(t, err)
// 	for k, v := range second {
// 		newVJSON, err := rkv.GetMapElement(mapKey, k, 0)
// 		require.NoError(t, err)
// 		vJSON, err := json.Marshal(v)
// 		require.NoError(t, err)
// 		require.Equal(t, vJSON, newVJSON)
// 	}
// 	newSecondJSON, err := rkv.GetMap(mapKey, 0)
// 	require.NoError(t, err)
// 	require.Equal(t, secondJSON, newSecondJSON)
// }

// // makeTransactionLog is a utility function which generates a remoteWriter for
// // testing purposes.
// func makeTransactionLog(baseDir, password string, t *testing.T) *kvsync.remoteWriter {

// 	localStore := kvsync.NewKVFilesystem(ekv.MakeMemstore())
// 	// Construct remote store
// 	remoteStore := &mockRemote{data: make(map[string][]byte, 0)}

// 	// Construct device secret
// 	deviceSecret := []byte("deviceSecret")

// 	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

// 	// Construct transaction log
// 	txLog, err := kvsync.NewTransactionLog(baseDir+"test.txt", localStore,
// 		remoteStore, deviceSecret, rngGen)
// 	require.NoError(t, err)

// 	return txLog
// }

type mockRemote struct {
	lck  sync.Mutex
	data map[string][]byte
	t    *testing.T
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

func (m *mockRemote) ReadDir(path string) ([]byte, error) {
	dirs := []string{
		"hello",
		"these",
		"are",
		"directory",
		"names",
	}

	data, err := json.Marshal(dirs)
	m.t.Logf("Data: %s", data)
	return data, err
}

func (m *mockRemote) GetLastModified(path string) (string, error) {
	return netTime.Now().UTC().Format(time.RFC3339), nil
}

func (m *mockRemote) GetLastWrite() (string, error) {
	return netTime.Now().UTC().Format(time.RFC3339), nil
}

func TestReadDir(t *testing.T) {
	mRemote := newRemoteStoreFileSystemWrapper(
		&mockRemote{t: t, data: make(map[string][]byte)})

	dirs, err := mRemote.ReadDir("test")
	require.NoError(t, err)
	t.Logf("%+v", dirs)
}
