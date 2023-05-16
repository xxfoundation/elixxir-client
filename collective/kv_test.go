////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

///////////////////////////////////////////////////////////////////////////////
// KV Testing
///////////////////////////////////////////////////////////////////////////////

// Smoke test of NewOrLoadKV.
func TestNewOrLoadRemoteKv(t *testing.T) {
	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "", t)

	// Create remote kv
	received := newKV(txLog, kv)

	// Create expected remote kv
	expected := &internalKV{
		local:              kv,
		txLog:              txLog,
		keyUpdateListeners: make(map[string]KeyUpdateCallback),
		mapUpdateListeners: make(map[string]mapChangedByRemoteCallback),
	}

	// Check equality of created vs expected remote kv
	require.Equal(t, expected, received)
}

// Unit test for NewOrLoadKV. Ensures that it will load if there is data
// on disk.
func TestNewOrLoadRemoteKv_Loading(t *testing.T) {

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "kv_Loading_TestDir", t)

	// Create remote kv
	rkv := newKV(txLog, kv)

	instance, err := GetInstanceID(rkv.local)
	require.NoError(t, err)

	require.NotEmpty(t, instance.String())
}

// Unit test of KV.Set.
func TestKV_Set(t *testing.T) {
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "workingDirSet", t)

	// Create remote kv
	rkv := newKV(txLog, kv)

	rkv.txLog.io = &mockRemote{
		data: make(map[string][]byte, 0),
	}

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

		select {
		case <-time.After(500 * time.Second):
			t.Fatalf("Failed to recieve from callback")
		case txKey := <-txChan:
			require.Equal(t, txKey, key)
		}
	}
}

// Unit test of KV.Get.
func TestKV_Get(t *testing.T) {
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct mutate log
	txLog := makeTransactionLog(kv, "workingDir", t)

	// Create remote kv
	rkv := newKV(txLog, kv)

	// Overwrite remote w/ non file IO option
	rkv.txLog.io = &mockRemote{
		data: make(map[string][]byte, 0),
	}

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
		case <-time.After(500 * time.Second):
			t.Fatalf("Failed to recieve from callback")
		case <-txChan:
		}

		received, err := rkv.GetBytes(key)
		require.NoError(t, err)

		require.Equal(t, val, received)
	}
}
