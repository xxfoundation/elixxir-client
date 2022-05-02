////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"bytes"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

// Tests that manager adheres to the FileTransfer interface.
var _ FileTransfer = (*manager)(nil)

// Tests that Cmix adheres to the cmix.Client interface.
var _ Cmix = (cmix.Client)(nil)

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
	params.MaxThroughput = math.MaxInt

	// Set up the first client
	myID1 := id.NewIdFromString("myID1", id.User, t)
	kv1 := versioned.NewKV(ekv.MakeMemstore())
	sendNewCbChan1 := make(chan *TransferInfo)
	sendNewCb1 := func(recipient *id.ID, info *TransferInfo) error {
		sendNewCbChan1 <- info
		return nil
	}
	sendEndCbChan1 := make(chan *id.ID)
	sendEndCb1 := func(recipient *id.ID) { sendEndCbChan1 <- recipient }
	ftm1, err := NewManager(sendNewCb1, sendEndCb1, params, myID1,
		newMockCmix(myID1, cMixHandler), kv1, rngGen)
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
	kv2 := versioned.NewKV(ekv.MakeMemstore())
	ftm2, err := NewManager(nil, nil, params, myID2,
		newMockCmix(myID2, cMixHandler), kv2, rngGen)
	if err != nil {
		t.Errorf("Failed to create new file transfer manager 2: %+v", err)
	}
	m2 := ftm2.(*manager)

	stop2, err := m2.StartProcesses()
	if err != nil {
		t.Errorf("Failed to start processes for manager 2: %+v", err)
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
	var called bool
	timeReceived := make(chan time.Time)
	go func() {
		select {
		case r := <-sendNewCbChan1:
			tid, err := m2.AddNew(
				r.FileName, &r.Key, r.Mac, r.NumParts, r.Size, r.Retry, nil, 0)
			if err != nil {
				t.Errorf("Failed to add transfer: %+v", err)
			}
			receiveProgressCB := func(completed bool, received, total uint16,
				fpt FilePartTracker, err error) {
				if completed && !called {
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
					"Failed to Rregister received progress callback: %+v", err3)
			}
		case <-time.After(2100 * time.Millisecond):
			t.Errorf("Timed out waiting to receive new file transfer.")
			wg.Done()
		}
	}()

	// Define sent progress callback
	wg.Add(1)
	sentProgressCb1 := func(completed bool, arrived, total uint16,
		fpt FilePartTracker, err error) {
		if completed {
			wg.Done()
		}
	}

	// Send file.
	sendStart := netTime.Now()
	tid1, err := m1.Send(
		fileName, fileType, fileData, myID2, retry, preview, sentProgressCb1, 0)
	if err != nil {
		t.Errorf("Failed to send file: %+v", err)
	}

	go func() {
		select {
		case tr := <-timeReceived:
			fileSize := len(fileData)
			sendTime := tr.Sub(sendStart)
			fileSizeKb := float32(fileSize) * .001
			speed := fileSizeKb * float32(time.Second) / (float32(sendTime))
			t.Logf("Completed receiving file %q in %s (%.2f kb @ %.2f kb/s).",
				fileName, sendTime, fileSizeKb, speed)
		}
	}()

	// Wait for file to be sent and received
	wg.Wait()

	select {
	case <-sendEndCbChan1:
	case <-time.After(15 * time.Millisecond):
		t.Error("Timed out waiting for end callback to be called.")
	}

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

const loremIpsum = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed sit amet urna venenatis, rutrum magna maximus, tempor orci. Cras sit amet nulla id dolor blandit commodo. Suspendisse potenti. Praesent gravida porttitor metus vel aliquam. Maecenas rutrum velit at lobortis auctor. Mauris porta blandit tempor. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Morbi volutpat posuere maximus. Nunc in augue molestie ante mattis tempor.

Phasellus placerat elit eu fringilla pharetra. Vestibulum consectetur pulvinar nunc, vestibulum tincidunt felis rhoncus sit amet. Duis non dolor eleifend nibh luctus eleifend. Nunc urna odio, euismod sit amet feugiat ut, dapibus vel elit. Nulla est mauris, posuere eget enim cursus, vehicula viverra est. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque mattis, nisi quis consectetur semper, neque enim rhoncus dolor, ut aliquam leo orci sed dolor. Integer ullamcorper pulvinar turpis, a sollicitudin nunc posuere et. Nullam orci nibh, facilisis ac massa eu, bibendum bibendum sapien. Sed tincidunt nunc mauris, nec ullamcorper enim lacinia nec. Nulla dapibus sapien ut odio bibendum, tempus ornare sapien lacinia.

Duis ac hendrerit augue. Nullam porttitor feugiat finibus. Nam enim urna, maximus et ligula eu, aliquet convallis turpis. Vestibulum luctus quam in dictum efficitur. Vestibulum ac pulvinar ipsum. Vivamus consectetur augue nec tellus mollis, at iaculis magna efficitur. Nunc dictum convallis sem, at vehicula nulla accumsan non. Nullam blandit orci vel turpis convallis, mollis porttitor felis accumsan. Sed non posuere leo. Proin ultricies varius nulla at ultricies. Phasellus et pharetra justo. Quisque eu orci odio. Pellentesque pharetra tempor tempor. Aliquam ac nulla lorem. Sed dignissim ligula sit amet nibh fermentum facilisis.

Donec facilisis rhoncus ante. Duis nec nisi et dolor congue semper vel id ligula. Mauris non eleifend libero, et sodales urna. Nullam pharetra gravida velit non mollis. Integer vel ultrices libero, at ultrices magna. Duis semper risus a leo vulputate consectetur. Cras sit amet convallis sapien. Sed blandit, felis et porttitor fringilla, urna tellus commodo metus, at pharetra nibh urna sed sem. Nam ex dui, posuere id mi et, egestas tincidunt est. Nullam elementum pulvinar diam in maximus. Maecenas vel augue vitae nunc consectetur vestibulum in aliquet lacus. Nullam nec lectus dapibus, dictum nisi nec, congue quam. Suspendisse mollis vel diam nec dapibus. Mauris neque justo, scelerisque et suscipit non, imperdiet eget leo. Vestibulum leo turpis, dapibus ac lorem a, mollis pulvinar quam.

Sed sed mauris a neque dignissim aliquet. Aliquam congue gravida velit in efficitur. Integer elementum feugiat est, ac lacinia libero bibendum sed. Sed vestibulum suscipit dignissim. Nunc scelerisque, turpis quis varius tristique, enim lacus vehicula lacus, id vestibulum velit erat eu odio. Donec tincidunt nunc sit amet sapien varius ornare. Phasellus semper venenatis ligula eget euismod. Mauris sodales massa tempor, cursus velit a, feugiat neque. Sed odio justo, rhoncus eu fermentum non, tristique a quam. In vehicula in tortor nec iaculis. Cras ligula sem, sollicitudin at nulla eget, placerat lacinia massa. Mauris tempus quam sit amet leo efficitur egestas. Proin iaculis, velit in blandit egestas, felis odio sollicitudin ipsum, eget interdum leo odio tempor nisi. Curabitur sed mauris id turpis tempor finibus ut mollis lectus. Curabitur neque libero, aliquam facilisis lobortis eget, posuere in augue. In sodales urna sit amet elit euismod rhoncus.`
