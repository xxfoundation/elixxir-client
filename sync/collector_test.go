////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"os"
	"testing"
)

// Smoke test of NewCollector.
func TestNewCollector(t *testing.T) {
	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	remoteKv, err := NewOrLoadRemoteKv(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId := "testingMyId"

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	collector := NewCollector(syncPath, myId, txLog, fsRemote, remoteKv)

	expected := &Collector{
		syncPath:             syncPath,
		myID:                 myId,
		lastUpdates:          make(changeLogger, 0),
		SynchronizationEpoch: synchronizationEpoch,
		txLog:                txLog,
		remote:               fsRemote,
		kv:                   remoteKv,
	}

	require.Equal(t, expected, collector)

}

func TestCollector_collectChanges(t *testing.T) {

}

func makeCollector(t *testing.T) *Collector {
	syncPath := baseDir + "collector/"
	txLog := makeTransactionLog(syncPath, password, t)

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create remote kv
	remoteKv, err := NewOrLoadRemoteKv(txLog, kv, nil, nil, nil)
	require.NoError(t, err)

	myId := "testingMyId"

	workingDir := baseDir + "remoteFsSmoke/"
	// Delete the test file at the end
	defer os.RemoveAll(baseDir)

	fsRemote := NewFileSystemRemoteStorage(workingDir)

	return NewCollector(syncPath, myId, txLog, fsRemote, remoteKv)
}
