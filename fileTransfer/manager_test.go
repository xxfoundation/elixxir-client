////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/elixxir/client/v5/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

// Smoke test of the entire file transfer system.
func Test_FileTransfer_Smoke(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelDebug)
	// Set up cMix and E2E message handlers
	cMixHandler := newMockCmixHandler()
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	params := DefaultParams()

	// Set up the first client
	myID1 := id.NewIdFromString("myID1", id.User, t)
	storage1 := newMockStorage()
	cMix1 := newMockCmix(myID1, cMixHandler, storage1)
	user1 := newMockE2e(myID1, cMix1, storage1, rngGen)
	ftm1, err := NewManager(params, user1)
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
	ftm2, err := NewManager(params, user2)
	if err != nil {
		t.Errorf("Failed to create new file transfer manager 2: %+v", err)
	}
	m2 := ftm2.(*manager)

	stop2, err := m2.StartProcesses()
	if err != nil {
		t.Errorf("Failed to start processes for manager 2: %+v", err)
	}

	sendNewCbChan1 := make(chan []byte)
	sendNewCb1 := func(transferInfo []byte) error {
		sendNewCbChan1 <- transferInfo
		return nil
	}

	// Wait group prevents the test from quiting before the file has completed
	// sending and receiving
	var wg sync.WaitGroup

	// Define details of file to send
	fileName, fileType := "myFile", "txt"
	fileData := []byte(loremIpsum)
	preview := []byte("Lorem ipsum dolor sit amet")
	retry := float32(2.0)

	// Create go func that waits for file transfer to be received to register
	// a progress callback that then checks that the file received is correct
	// when done
	wg.Add(1)
	called := uint32(0)
	timeReceived := make(chan time.Time)
	go func() {
		select {
		case r := <-sendNewCbChan1:
			tid, _, err := m2.HandleIncomingTransfer(r, nil, 0)
			if err != nil {
				t.Errorf("Failed to add transfer: %+v", err)
			}
			receiveProgressCB := func(completed bool, received, total uint16,
				rt ReceivedTransfer, fpt FilePartTracker, err error) {
				if completed && atomic.CompareAndSwapUint32(&called, 0, 1) {
					timeReceived <- netTime.Now()
					receivedFile, err2 := m2.Receive(tid)
					if err2 != nil {
						t.Errorf("Failed to receive file: %+v", err2)
					}

					if !bytes.Equal(fileData, receivedFile) {
						t.Errorf("Received file does not match sent."+
							"\nsent:     %q\nreceived: %q",
							fileData, receivedFile)
					}
					wg.Done()
				}
			}
			err3 := m2.RegisterReceivedProgressCallback(
				tid, receiveProgressCB, 0)
			if err3 != nil {
				t.Errorf(
					"Failed to register received progress callback: %+v", err3)
			}
		case <-time.After(2100 * time.Millisecond):
			t.Errorf("Timed out waiting to receive new file transfer.")
			wg.Done()
		}
	}()

	// Define sent progress callback
	wg.Add(1)
	sentProgressCb1 := func(completed bool, arrived, total uint16,
		st SentTransfer, fpt FilePartTracker, err error) {
		if completed {
			wg.Done()
		}
	}

	// Send file.
	sendStart := netTime.Now()
	tid1, err := m1.Send(myID2, fileName, fileType, fileData, retry, preview,
		sentProgressCb1, 0, sendNewCb1)
	if err != nil {
		t.Errorf("Failed to send file: %+v", err)
	}

	go func() {
		select {
		case tr := <-timeReceived:
			fileSize := len(fileData)
			sendTime := tr.Sub(sendStart)
			fileSizeKb := float64(fileSize) * .001
			throughput := fileSizeKb * float64(time.Second) / (float64(sendTime))
			t.Logf("Completed receiving file %q in %s (%.2f kb @ %.2f kb/s).",
				fileName, sendTime, fileSizeKb, throughput)

			expectedThroughput := float64(params.MaxThroughput) * .001
			delta := (math.Abs(expectedThroughput-throughput) /
				((expectedThroughput + throughput) / 2)) * 100
			t.Logf("Expected bandwidth:   %.2f kb/s", expectedThroughput)
			t.Logf("Bandwidth difference: %.2f kb/s (%.2f%%)",
				expectedThroughput-throughput, delta)
		}
	}()

	// Wait for file to be sent and received
	wg.Wait()

	err = m1.CloseSend(tid1)
	if err != nil {
		t.Errorf("Failed to close transfer: %+v", err)
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
