////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
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

// Unit test of KV.Set.
func TestKV_Set(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelInfo)
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
	rkv.col.synchronizationEpoch = 50 * time.Millisecond

	kv2 := ekv.MakeMemstore()
	txLog2 := makeTransactionLog(kv2, baseDir, remoteStore,
		csprng.NewSystemRNG(), t)
	rkv2 := newKV(txLog2, kv2)

	rkv2.col = newCollector(txLog2.header.DeviceID,
		baseDir, remoteStore, rkv,
		txLog2.encrypt, txLog2)
	rkv2.col.synchronizationEpoch = 50 * time.Millisecond

	mStopper := stoppable.NewMulti("SetTest")
	stop1, err := rkv.StartProcesses()
	require.NoError(t, err)
	stop2, err := rkv2.StartProcesses()
	require.NoError(t, err)
	mStopper.Add(stop1)
	mStopper.Add(stop2)

	// Construct mock update callback
	txChan := make(chan string, numTests)
	updateCb := KeyUpdateCallback(func(key string, oldVal, newVal []byte,
		op versioned.KeyOperation) {
		require.Nil(t, oldVal)
		txChan <- key
	})

	for i := 0; i < numTests; i++ {
		// NOTE: we're deliberately abusing the start/stop here
		// to stress test it.
		err = mStopper.Close()
		require.NoError(t, err)
		err = stoppable.WaitForStopped(mStopper, 5*time.Second)
		if err != nil {
			pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		}
		require.NoError(t, err, mStopper.GetStatus())

		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		rkv2.ListenOnRemoteKey(key, updateCb)
		mStopper = stoppable.NewMulti(fmt.Sprintf("SetTest %d", i))
		stop1, err := rkv.StartProcesses()
		require.NoError(t, err)
		stop2, err := rkv2.StartProcesses()
		require.NoError(t, err)
		mStopper.Add(stop1)
		mStopper.Add(stop2)

		require.NoError(t, rkv.SetRemote(key, val))

		select {
		case <-time.After(5 * time.Second):
			t.Fatalf("Failed to receive from callback")
		case txKey := <-txChan:
			require.Equal(t, txKey, key)
		}
	}

	err = mStopper.Close()
	require.NoError(t, err)
	err = stoppable.WaitForStopped(mStopper, 5*time.Second)
	if err != nil {
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	}
	require.NoError(t, err)
}

// Unit test of KV.Get.
func TestKV_Get(t *testing.T) {
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()
	remoteStore := newMockRemote()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "workingDir", remoteStore,
		csprng.NewSystemRNG(), t)

	// Create remote kv
	rkv := newKV(txLog, kv)

	// Construct mock update callback
	txChan := make(chan string, numTests)
	updateCb := KeyUpdateCallback(func(key string, oldVal, newVal []byte,
		op versioned.KeyOperation) {
		require.Nil(t, oldVal)
		txChan <- key
	})

	// Add intents to remote KV
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), []byte("val"+strconv.Itoa(i))
		rkv.ListenOnRemoteKey(key, updateCb)
		require.NoError(t, rkv.SetRemote(key, val))

		// Ensure Write has completed
		select {
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Failed to recieve from callback")
		case <-txChan:
		}

		received, err := rkv.GetBytes(key)
		require.NoError(t, err)

		require.Equal(t, val, received)
	}
}
