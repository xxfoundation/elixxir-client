////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

package collective

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

// Smoke test of NewCollector.
func TestNewCollector(t *testing.T) {
	baseDir := ".TestNewCollector/"
	syncPath := "collector/"
	os.RemoveAll(baseDir)

	remoteStore := NewMockRemote()

	// Construct kv
	kv := ekv.MakeMemstore()

	txLog := makeTransactionLog(kv, syncPath, remoteStore,
		NewCountingReader(), t)

	// Create remote kv
	remoteKv := newVersionedKV(txLog, kv, nil)

	myID, err := GetInstanceID(kv)
	require.NoError(t, err)

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)

	crypt := &deviceCrypto{
		secret: []byte("deviceSecret"),
		rngGen: rngGen,
	}

	testcol := newCollector(myID, syncPath, remoteStore, remoteKv.remote,
		crypt, txLog)

	zero := uint32(0)

	expected := &collector{
		syncPath:             syncPath,
		myID:                 myID,
		lastUpdateRead:       make(map[InstanceID]time.Time, 0),
		synchronizationEpoch: synchronizationEpoch,
		txLog:                txLog,
		remote:               remoteStore,
		kv:                   remoteKv.remote,
		encrypt:              crypt,
		keyID:                crypt.KeyID(myID),
		devicePatchTracker:   make(map[InstanceID]*Patch),
		lastMutationRead:     make(map[InstanceID]time.Time),
		connected:            &zero,
		synched:              &zero,
		notifier:             &notifier{},
	}

	require.Equal(t, expected, testcol)
}

func TestNewCollector_CollectChanges(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelTrace)

	baseDir := ".TestNewCollector_CollectChanges/"
	remoteStore := NewMockRemote()
	syncPath := baseDir + "collector/"
	devices := make([]InstanceID, 0)

	timestamps := constructTimestamps(t, 30)
	rngSrc := rand.New(rand.NewSource(42))
	for i := 0; i < 5; i++ {
		kv := ekv.MakeMemstore()
		txLog := makeTransactionLog(kv, baseDir, remoteStore,
			rngSrc, t)
		for j := 0; j < 6; j++ {
			tsIdx := i*5 + j
			key := fmt.Sprintf("Key%d", j)
			value := []byte(fmt.Sprintf("Value%d", i*j))
			mutation := NewMutate(timestamps[tsIdx],
				value, false)
			txLog.state.AddUnsafe(key, mutation)
		}
		serial, err := txLog.state.Serialize()
		encrypted := txLog.encrypt.Encrypt(serial)
		require.NoError(t, err)
		file := buildFile(txLog.header, encrypted)
		err = txLog.io.Write(getTxLogPath(syncPath,
			txLog.encrypt.KeyID(txLog.header.DeviceID),
			txLog.header.DeviceID), file)
		require.NoError(t, err)
		devices = append(devices, txLog.header.DeviceID)
	}

	// Construct kv
	kv := ekv.MakeMemstore()
	txLog := makeTransactionLog(kv, syncPath, remoteStore,
		NewCountingReader(), t)

	// Create remote kv
	remoteKv := newVersionedKV(txLog, kv, nil)

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)
	rng := rngGen.GetStream()
	defer rng.Close()

	crypt := &deviceCrypto{
		secret: []byte("deviceSecret"),
		rngGen: rngGen,
	}

	// Construct collector
	myID, err := GetInstanceID(kv)
	require.NoError(t, err)
	testcol := newCollector(myID, syncPath, remoteStore, remoteKv.remote,
		crypt, txLog)

	changes, err := testcol.collectAllChanges(devices)
	require.NoError(t, err)

	// t.Logf("changes: %+v", changes)
	require.Equal(t, 5, len(changes))

}

func TestCollector_ApplyChanges(t *testing.T) {
	baseDir := ".TestCollector_ApplyChanges/"
	os.RemoveAll(baseDir)

	// workingDir := baseDir + "remoteFsSmoke/"
	remoteStore := NewMockRemote()
	// remoteStore := NewFileSystemRemoteStorage(workingDir)
	syncPath := "collector/"

	devices := make([]InstanceID, 0)

	timestamps := constructTimestamps(t, 30)
	rngSrc := rand.New(rand.NewSource(42))
	for i := 0; i < 5; i++ {
		kv := ekv.MakeMemstore()
		txLog := makeTransactionLog(kv, syncPath, remoteStore,
			rngSrc, t)
		for j := 0; j < 6; j++ {
			tsIdx := i*5 + j
			key := fmt.Sprintf("Key%d", j)
			value := []byte(fmt.Sprintf("Value%d%d", i, j))
			mutation := NewMutate(timestamps[tsIdx],
				value, false)
			// t.Logf("%s: %s -> %s @ ts: %s", txLog.state.myID,
			// 	key, string(value), timestamps[tsIdx])
			txLog.state.AddUnsafe(key, mutation)
		}
		serial, err := txLog.state.Serialize()
		encrypted := txLog.encrypt.Encrypt(serial)
		require.NoError(t, err)
		file := buildFile(txLog.header, encrypted)
		logPath := getTxLogPath(syncPath,
			txLog.encrypt.KeyID(txLog.header.DeviceID),
			txLog.header.DeviceID)
		t.Logf("LogPath: %s", logPath)
		err = txLog.io.Write(logPath, file)
		require.NoError(t, err)
		devices = append(devices, txLog.header.DeviceID)
	}

	// Construct kv
	kv := ekv.MakeMemstore()

	txLog := makeTransactionLog(kv, syncPath, remoteStore,
		NewCountingReader(), t)

	// Create remote kv
	remoteKv := newVersionedKV(txLog, kv, nil)

	myID, err := GetInstanceID(kv)
	require.NoError(t, err)

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)

	crypt := &deviceCrypto{
		secret: []byte("deviceSecret"),
		rngGen: rngGen,
	}

	// Construct collector
	testcol := newCollector(myID, syncPath, remoteStore, remoteKv.remote,
		crypt, txLog)
	_, err = testcol.collectAllChanges(devices)
	require.NoError(t, err)
	require.NoError(t, testcol.applyChanges())

	// These are generated from a previous run, they're always the same due
	// to the entropy source
	expectedVals := []int{20, 11, 12, 33, 14, 15}
	for i := 0; i < 6; i++ {
		key := fmt.Sprintf("Key%d", i)
		val, err := remoteKv.remote.GetBytes(key)
		require.NoError(t, err, key)
		expectedVal := fmt.Sprintf("Value%d", expectedVals[i])
		// t.Logf("Change: %s -> %s", key, string(val))
		require.Equal(t, expectedVal, string(val))
	}

}
