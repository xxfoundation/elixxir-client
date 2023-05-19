////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"os"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

///////////////////////////////////////////////////////////////////////////////
// KV Testing
///////////////////////////////////////////////////////////////////////////////

// Smoke test of NewOrLoadKV.
func TestNewOrLoadRemoteKv(t *testing.T) {
	// Construct kv
	kv := ekv.MakeMemstore()
	remoteStore := newMockRemote()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "", remoteStore,
		NewCountingReader(), t)

	// Create remote kv
	received := newKV(txLog, kv)

	isSync := atomic.Bool{}
	isSync.Store(false)

	// Create expected remote kv
	expected := &internalKV{
		local:              kv,
		txLog:              txLog,
		keyUpdateListeners: make(map[string]KeyUpdateCallback),
		mapUpdateListeners: make(map[string]mapChangedByRemoteCallback),
		isSynchronizing:    &isSync,
	}

	// Check equality of created vs expected remote kv
	require.Equal(t, expected, received)
}

// Unit test for NewOrLoadKV. Ensures that it will load if there is data
// on disk.
func TestNewOrLoadRemoteKv_Loading(t *testing.T) {

	// Construct kv
	kv := ekv.MakeMemstore()
	remoteStore := newMockRemote()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "kv_Loading_TestDir", remoteStore,
		NewCountingReader(), t)

	// Create remote kv
	rkv := newKV(txLog, kv)

	instance, err := GetInstanceID(rkv.local)
	require.NoError(t, err)

	require.NotEmpty(t, instance.String())
}

// TestKV_SetGet tests setting and getting between two synchronized
// KVs communicating through a memory-based remote store.
func TestKV_SetGet(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelDebug)
	// jww.SetLogThreshold(jww.LevelDebug)
	const numTests = 100

	baseDir := ".workingDirSet"

	remoteStore := newMockRemote()

	kv := ekv.MakeMemstore()
	txLog := makeTransactionLog(kv, baseDir, remoteStore,
		csprng.NewSystemRNG(), t)
	rkv := newKV(txLog, kv)

	rkv.col = newCollector(txLog.header.DeviceID,
		baseDir, remoteStore, rkv,
		txLog.encrypt, txLog)
	rkv.col.synchronizationEpoch = 500 * time.Millisecond
	txLog.uploadPeriod = 500 * time.Millisecond

	kv2 := ekv.MakeMemstore()
	txLog2 := makeTransactionLog(kv2, baseDir, remoteStore,
		csprng.NewSystemRNG(), t)
	rkv2 := newKV(txLog2, kv2)

	rkv2.col = newCollector(txLog2.header.DeviceID,
		baseDir, remoteStore, rkv2,
		txLog2.encrypt, txLog2)
	rkv2.col.synchronizationEpoch = 500 * time.Millisecond
	txLog2.uploadPeriod = 500 * time.Millisecond

	// t.Logf("Device 1: %s, Device 2: %s", txLog.header.DeviceID,
	// txLog2.header.DeviceID)

	// Construct mock update callback
	txChan := make(chan string, numTests)
	updateCb := KeyUpdateCallback(func(key string, oldVal, newVal []byte,
		op versioned.KeyOperation) {
		// t.Logf("%s: %s -> %s", key, string(oldVal), string(newVal))
		require.Nil(t, oldVal)
		txChan <- key
	})

	for i := 0; i < numTests; i++ {
		key := "key" + strconv.Itoa(i)
		rkv2.ListenOnRemoteKey(key, updateCb)
	}

	mStopper := stoppable.NewMulti("SetTest")
	stop1, err := rkv.StartProcesses()
	require.NoError(t, err)
	stop2, err := rkv2.StartProcesses()
	require.NoError(t, err)
	mStopper.Add(stop1)
	mStopper.Add(stop2)

	expected := make(map[string][]byte)
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		expected[key] = val
		// t.Logf("SetRemote: %s: %s", key, string(val))
		require.NoError(t, rkv.SetRemote(key, val))
	}
	for i := 0; i < numTests; i++ {
		select {
		case <-time.After(1 * time.Second):
			t.Fatalf("Failed to receive from callback %d", i)
		case txKey := <-txChan:
			_, ok := expected[txKey]
			require.True(t, ok, txKey)
		}
	}

	err = mStopper.Close()
	require.NoError(t, err)
	err = stoppable.WaitForStopped(mStopper, 1*time.Second)
	if err != nil {
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	}
	require.NoError(t, err)

	// Verify everything is synchronized between both instances
	for k, v := range expected {
		v1, err := rkv.GetBytes(k)
		require.NoError(t, err)
		require.Equal(t, v, v1, k)
		v2, err := rkv2.GetBytes(k)
		require.NoError(t, err)
		require.Equal(t, v, v2, k)
	}
}
