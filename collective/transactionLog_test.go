////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/base64"
	"os"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

// Smoke test for NewOrLoadTransactionLog.
//
// Intentionally constructs remoteWriter manually for testing purposes.
func TestNewOrLoadTransactionLog(t *testing.T) {
	baseDir := ".testDir"
	logFile := baseDir + "/test.txt"
	os.RemoveAll(baseDir)
	password := "password"
	fs, err := ekv.NewFilestore(baseDir, password)
	require.NoError(t, err)

	// Construct remote store
	remoteStore := NewMockRemote()

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rngGen,
	}

	rng := rngGen.GetStream()
	defer rng.Close()
	deviceID, err := InitInstanceID(fs, rng)
	require.NoError(t, err)

	// Construct mutate log
	txLog, err := newRemoteWriter(logFile, deviceID,
		remoteStore, crypt, fs)
	require.NoError(t, err)

	zero := uint32(0)

	logPath := getTxLogPath(logFile, crypt.KeyID(deviceID), deviceID)
	// Construct expected mutate log object
	expected := &remoteWriter{
		path:           logPath,
		header:         newHeader(deviceID),
		state:          newPatch(deviceID),
		adds:           txLog.adds, // hack, but new chan won't work
		io:             remoteStore,
		encrypt:        crypt,
		kv:             fs,
		localWriteKey:  makeLocalWriteKey(logFile),
		remoteUpToDate: &zero,
		notifier:       &notifier{},
		uploadPeriod:   defaultUploadPeriod,
	}

	// Ensure constructor generates expected object
	require.Equal(t, expected, txLog)

}

// Unit test for NewOrLoadTransactionLog. Tests whether this will load from
// disk and deserialize the data into the remoteWriter file.
//
// Intentionally constructs remoteWriter manually for testing purposes.
func TestNewOrLoadTransactionLog_Loading(t *testing.T) {
	baseDir := ".testDir_TransactionLog_Loading"
	os.RemoveAll(baseDir)
	logFile := baseDir + "/test.txt"
	password := "password"
	fs, err := ekv.NewFilestore(baseDir, password)
	require.NoError(t, err)

	// Construct remote store
	remoteStore := NewMockRemote()

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rngGen,
	}

	rng := rngGen.GetStream()
	defer rng.Close()
	deviceID, err := InitInstanceID(fs, rng)
	require.NoError(t, err)

	// Construct mutate log
	txLog, err := newRemoteWriter(logFile, deviceID,
		remoteStore, crypt, fs)
	txLog.uploadPeriod = 250 * time.Millisecond
	require.NoError(t, err)

	ntfyCh := make(chan bool)
	ntfy := func(state bool) {
		ntfyCh <- state
	}
	txLog.Register(ntfy)

	stopper := stoppable.NewSingle("txLogRunner")
	go txLog.Runner(stopper)

	// Insert timestamps
	for cnt := 0; cnt < 10; cnt++ {
		// Construct mutate
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		_, _, err := txLog.Write(key, []byte(val))
		require.NoError(t, err)
	}

	done := false
	for !done {
		select {
		case <-time.After(5 * time.Second):
			t.Errorf("threads failed to stop")
			pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
			done = true
		case x := <-ntfyCh:
			done = x
		}
	}

	err = stopper.Close()
	require.NoError(t, err)
	err = stoppable.WaitForStopped(stopper, 5*time.Second)
	require.NoError(t, err)
	require.True(t, stopper.IsStopped())

	newTxLog, err := newRemoteWriter(logFile, deviceID,
		remoteStore, crypt, fs)
	require.NoError(t, err)

	require.NoError(t, err)

	// Hacks for comparison
	newTxLog.adds = txLog.adds
	newTxLog.notifier = txLog.notifier
	newTxLog.remoteUpToDate = txLog.remoteUpToDate
	newTxLog.uploadPeriod = txLog.uploadPeriod

	// Ensure loaded log matches original log
	require.Equal(t, txLog, newTxLog)
}

// Unit test for Serialize. Ensures the that function returns the serialized
// internal state. Checks against a hardcoded base64 string.
func TestTransactionLog_Serialize(t *testing.T) {

	kv := ekv.MakeMemstore()
	remoteStore := NewMockRemote()

	txLog := makeTransactionLog(kv, ".baseDir", remoteStore,
		NewCountingReader(), t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert mock data into mutate log
	for cnt, curTs := range mockTimestamps {
		// Construct mutate
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewMutate(curTs, []byte(val), false)

		txLog.state.AddUnsafe(key, newTx)
	}

	// Serialize data
	data, err := txLog.state.Serialize()
	require.NoError(t, err)

	// Encode data to bas64
	data64 := base64.RawStdEncoding.EncodeToString(data)

	expected := "eyJrZXkwIjp7IlRpbWVzdGFtcCI6MTQ1MDczNTcyMSwiVmFsdWUiOiJkbUZzTUE9PSIsIkRlbGV0aW9uIjpmYWxzZX0sImtleTEiOnsiVGltZXN0YW1wIjoxMzg3NjYzNzIxLCJWYWx1ZSI6ImRtRnNNUT09IiwiRGVsZXRpb24iOmZhbHNlfSwia2V5MiI6eyJUaW1lc3RhbXAiOjEwNzIwNDQ1MjEsIlZhbHVlIjoiZG1Gc01nPT0iLCJEZWxldGlvbiI6ZmFsc2V9LCJrZXkzIjp7IlRpbWVzdGFtcCI6MTM1NjEyNzcyMSwiVmFsdWUiOiJkbUZzTXc9PSIsIkRlbGV0aW9uIjpmYWxzZX0sImtleTQiOnsiVGltZXN0YW1wIjoxNDE5MTk5NzIxLCJWYWx1ZSI6ImRtRnNOQT09IiwiRGVsZXRpb24iOmZhbHNlfSwia2V5NSI6eyJUaW1lc3RhbXAiOjEwMDg5NzI1MjEsIlZhbHVlIjoiZG1Gc05RPT0iLCJEZWxldGlvbiI6ZmFsc2V9fQ"

	// Ensure encoded data using mock values matches hardcoded data.
	require.Equal(t, expected, data64)
}

// Unit test for Deserialize. Ensures that deserialize will construct the same
// remoteWriter that was serialized using remoteWriter.serialize.
//
// Intentionally constructs remoteWriter manually for testing purposes.
func TestTransactionLog_Deserialize(t *testing.T) {
	kv := ekv.MakeMemstore()
	remoteStore := NewMockRemote()

	txLog := makeTransactionLog(kv, ".baseDir", remoteStore,
		NewCountingReader(), t)

	// Construct timestamps
	mockTimestamps := constructTimestamps(t, 0)

	// Insert mock data into mutate log
	for cnt, curTs := range mockTimestamps {
		// Construct mutate
		key, val := "key"+strconv.Itoa(cnt), "val"+strconv.Itoa(cnt)
		newTx := NewMutate(curTs, []byte(val), false)

		txLog.state.AddUnsafe(key, newTx)
	}

	// Serialize data
	data, err := txLog.state.Serialize()
	require.NoError(t, err)

	kv2 := ekv.MakeMemstore()
	newTxLog := makeTransactionLog(kv2, ".baseDir2", remoteStore,
		NewCountingReader(), t)

	// Deserialize the mutate log
	require.NoError(t, newTxLog.state.Deserialize(data))

	// Ensure deserialized object matches original object
	require.Equal(t, txLog.state, newTxLog.state)
}
