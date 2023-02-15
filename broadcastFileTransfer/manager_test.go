////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"bytes"
	_ "embed"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/storage"
	"math/rand"
	"reflect"
	"testing"
)

//go:embed loremIpsum.txt
var loremIpsum string

// Tests that manager adheres to the FileTransfer interface.
var _ FileTransfer = (*manager)(nil)

// Tests that Cmix adheres to the cmix.Client interface.
var _ Cmix = (cmix.Client)(nil)

// Tests that Storage adheres to the storage.Session interface.
var _ Storage = (storage.Session)(nil)

// Tests that partitionFile partitions the given file into the expected parts.
func Test_partitionFile(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	partSize := 96
	fileData, expectedParts := newFile(24, partSize, prng, t)

	receivedParts := partitionFile(fileData, partSize)

	if !reflect.DeepEqual(expectedParts, receivedParts) {
		t.Errorf("File parts do not match expected."+
			"\nexpected: %q\nreceived: %q", expectedParts, receivedParts)
	}

	fullFile := bytes.Join(receivedParts, nil)
	if !bytes.Equal(fileData, fullFile) {
		t.Errorf("Full file does not match expected."+
			"\nexpected: %q\nreceived: %q", fileData, fullFile)
	}
}

// Tests that calcNumberOfFingerprints matches some manually calculated results.
func Test_calcNumberOfFingerprints(t *testing.T) {
	testValues := []struct {
		numParts int
		retry    float32
		result   uint16
	}{
		{12, 0.5, 18},
		{13, 0.6667, 21},
		{1, 0.89, 1},
		{2, 0.75, 3},
		{119, 0.45, 172},
	}

	for i, val := range testValues {
		result := calcNumberOfFingerprints(val.numParts, val.retry)

		if val.result != result {
			t.Errorf("calcNumberOfFingerprints(%3d, %3.2f) result is "+
				"incorrect (%d).\nexpected: %d\nreceived: %d",
				val.numParts, val.retry, i, val.result, result)
		}
	}
}

/*
// Smoke test of the entire file transfer system.
func Test_FileTransfer_Smoke(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelDebug)
	// Set up cMix and E2E message handlers
	cMixHandler := newMockCmixHandler()
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	params := DefaultParams()
	params.ResendWait = 5 * time.Second

	// Set up the first client
	myID1 := id.NewIdFromString("myID1", id.User, t)
	storage1 := newMockStorage()
	cMix1 := newMockCmix(myID1, cMixHandler, storage1)
	user1 := newMockE2e(myID1, cMix1, storage1, rngGen)
	ftm1, err := NewManager(user1, params)
	if err != nil {
		t.Errorf("Failed to create new file transfer manager 1: %+v", err)
	}
	m1 := ftm1.(*manager)

	stop1, err := m1.StartProcesses()
	if err != nil {
		t.Errorf("Failed to start processes for manager 1: %+v", err)
	}

	// Set up the second client
	myID2 := id.NewIdFromString("myID2", id.User, t)
	storage2 := newMockStorage()
	cMix2 := newMockCmix(myID2, cMixHandler, storage2)
	user2 := newMockE2e(myID2, cMix2, storage2, rngGen)
	ftm2, err := NewManager(user2, params)
	if err != nil {
		t.Errorf("Failed to create new file transfer manager 2: %+v", err)
	}
	m2 := ftm2.(*manager)

	stop2, err := m2.StartProcesses()
	if err != nil {
		t.Errorf("Failed to start processes for manager 2: %+v", err)
	}

	// Define details of file to send
	fileName, fileType := "myFile", "txt"
	fileData := []byte(loremIpsum)
	preview := []byte("Lorem ipsum dolor sit amet")
	retry := float32(2.0)

	// Define send complete callback
	fiChan := make(chan FileInfo, 1)
	completeCB := func(fi FileInfo) { fiChan <- fi }

	// Define sent progress callback
	sentProgressCb1 := func(completed bool, sent, received, total uint16,
		st SentTransfer, fpt FilePartTracker, err error) {
	}

	// Send file.
	_, err = m1.Send(fileName, fileType, fileData, retry, preview,
		completeCB, sentProgressCb1, 0)
	if err != nil {
		t.Errorf("Failed to send file: %+v", err)
	}

	var fi FileInfo
	select {
	case fi = <-fiChan:
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for transfer to complete.")
	}

	fileInfo, err := fi.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal FileInfo: %+v", err)
	}

	// Define received progress callback
	receivedCh := make(chan bool)
	receivedProgressCb := func(completed bool, received, total uint16,
		rt ReceivedTransfer, fpt FilePartTracker, err error) {
		if completed {
			receivedCh <- true
		}
	}

	receivedTID, _, err :=
		m2.HandleIncomingTransfer(fileInfo, receivedProgressCb, 0)
	if err != nil {
		t.Errorf("Failed to handle incoming transfer: %+v", err)
	}

	select {
	case <-receivedCh:
	case <-time.After(15 * time.Second):
		t.Fatalf("Timed out waiting for transfer to complete.")
	}

	file, err := m2.Receive(receivedTID)
	if err != nil {
		t.Errorf("Failed to receive file: %+v", err)
	}

	if !bytes.Equal(fileData, file) {
		t.Errorf("Received file does not match original.")
	}

	err = stop1.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 1: %+v", err)
	}

	err = stop2.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 2: %+v", err)
	}
}
*/