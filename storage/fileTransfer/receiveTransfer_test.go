////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that NewReceivedTransfer correctly creates a new ReceivedTransfer and
// that it is saved to storage.
func Test_NewReceivedTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")
	kvPrefixed := kv.Prefix(makeReceivedTransferPrefix(tid))
	fileSize := uint32(256)
	numParts, numFps := uint16(16), uint16(24)
	fpVector, _ := utility.NewStateVector(
		kvPrefixed, receivedFpVectorKey, uint32(numFps))
	receivedVector, _ := utility.NewStateVector(
		kvPrefixed, receivedVectorKey, uint32(numParts))

	expected := &ReceivedTransfer{
		key:         key,
		transferMAC: mac,
		fileSize:    fileSize,
		numParts:    numParts,
		numFps:      numFps,
		fpVector:    fpVector,
		receivedParts: &partStore{
			parts:    make(map[uint16][]byte, numParts),
			numParts: numParts,
			kv:       kvPrefixed,
		},
		receivedStatus:    receivedVector,
		progressCallbacks: []*receivedCallbackTracker{},
		mux:               sync.RWMutex{},
		kv:                kvPrefixed,
	}

	// Create new ReceivedTransfer
	rt, err := NewReceivedTransfer(tid, key, mac, fileSize, numParts, numFps, kv)
	if err != nil {
		t.Fatalf("NewReceivedTransfer returned an error: %v", err)
	}

	// Check that the new object matches the expected
	if !reflect.DeepEqual(expected, rt) {
		t.Errorf("New ReceivedTransfer does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, rt)
	}

	// Make sure it is saved to storage
	_, err = kvPrefixed.Get(receivedTransferKey, receivedTransferVersion)
	if err != nil {
		t.Fatalf("Failed to load ReceivedTransfer from storage: %+v", err)
	}

	// Check that the fingerprint vector has correct values
	if rt.fpVector.GetNumAvailable() != uint32(numFps) {
		t.Errorf("Incorrect number of available keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, rt.fpVector.GetNumAvailable())
	}
	if rt.fpVector.GetNumKeys() != uint32(numFps) {
		t.Errorf("Incorrect number of keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, rt.fpVector.GetNumKeys())
	}
	if rt.fpVector.GetNumUsed() != 0 {
		t.Errorf("Incorrect number of used keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", 0, rt.fpVector.GetNumUsed())
	}
}

// Tests that ReceivedTransfer.GetTransferKey returns the expected key.
func TestReceivedTransfer_GetTransferKey(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))

	tid, _ := ftCrypto.NewTransferID(prng)
	mac := []byte("transferMAC")
	expectedKey, _ := ftCrypto.NewTransferKey(prng)

	rt, err := NewReceivedTransfer(tid, expectedKey, mac, 256, 16, 1, kv)
	if err != nil {
		t.Errorf("Failed to create new ReceivedTransfer: %+v", err)
	}

	if expectedKey != rt.GetTransferKey() {
		t.Errorf("Failed to get expected transfer key."+
			"\nexpected: %s\nreceived: %s", expectedKey, rt.GetTransferKey())
	}
}

// Tests that ReceivedTransfer.GetTransferMAC returns the expected bytes.
func TestReceivedTransfer_GetTransferMAC(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))

	tid, _ := ftCrypto.NewTransferID(prng)
	expectedMAC := []byte("transferMAC")
	key, _ := ftCrypto.NewTransferKey(prng)

	rt, err := NewReceivedTransfer(tid, key, expectedMAC, 256, 16, 1, kv)
	if err != nil {
		t.Errorf("Failed to create new ReceivedTransfer: %+v", err)
	}

	if !bytes.Equal(expectedMAC, rt.GetTransferMAC()) {
		t.Errorf("Failed to get expected transfer MAC."+
			"\nexpected: %v\nreceived: %v", expectedMAC, rt.GetTransferMAC())
	}
}

// Tests that ReceivedTransfer.GetNumParts returns the expected number of parts.
func TestReceivedTransfer_GetNumParts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedNumParts := uint16(16)
	_, rt, _ := newRandomReceivedTransfer(expectedNumParts, 20, kv, t)

	if expectedNumParts != rt.GetNumParts() {
		t.Errorf("Failed to get expected number of parts."+
			"\nexpected: %d\nreceived: %d", expectedNumParts, rt.GetNumParts())
	}
}

// Tests that ReceivedTransfer.GetNumFps returns the expected number of
// fingerprints.
func TestReceivedTransfer_GetNumFps(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedNumFps := uint16(20)
	_, rt, _ := newRandomReceivedTransfer(16, expectedNumFps, kv, t)

	if expectedNumFps != rt.GetNumFps() {
		t.Errorf("Failed to get expected number of fingerprints."+
			"\nexpected: %d\nreceived: %d", expectedNumFps, rt.GetNumFps())
	}
}

// Tests that ReceivedTransfer.GetNumAvailableFps returns the expected number of
// available fingerprints.
func TestReceivedTransfer_GetNumAvailableFps(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts, numFps := uint16(16), uint16(24)
	_, rt, _ := newRandomReceivedTransfer(numParts, numFps, kv, t)

	if numFps-numParts != rt.GetNumAvailableFps() {
		t.Errorf("Failed to get expected number of available fingerprints."+
			"\nexpected: %d\nreceived: %d",
			numFps-numParts, rt.GetNumAvailableFps())
	}
}

// Tests that ReceivedTransfer.GetFileSize returns the expected file size.
func TestReceivedTransfer_GetFileSize(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, file := newRandomReceivedTransfer(16, 20, kv, t)
	expectedFileSize := len(file)

	if expectedFileSize != int(rt.GetFileSize()) {
		t.Errorf("Failed to get expected file size."+
			"\nexpected: %d\nreceived: %d", expectedFileSize, rt.GetFileSize())
	}
}

// Tests that ReceivedTransfer.IsPartReceived returns false for unreceived file
// parts and true when the file part has been received.
func TestReceivedTransfer_IsPartReceived(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	partNum := uint16(5)

	if rt.IsPartReceived(partNum) {
		t.Errorf("Part number %d received.", partNum)
	}

	_ = rt.receivedParts.addPart([]byte("part"), partNum)

	if !rt.IsPartReceived(partNum) {
		t.Errorf("Part number %d not received.", partNum)
	}
}

// checkReceivedProgress compares the output of ReceivedTransfer.GetProgress to
// expected values.
func checkReceivedProgress(completed bool, received, total uint16,
	eCompleted bool, eReceived, eTotal uint16) error {
	if eCompleted != completed || eReceived != received || eTotal != total {
		return errors.Errorf("Returned progress does not match expected."+
			"\n          completed  received  total"+
			"\nexpected:     %5t       %3d    %3d"+
			"\nreceived:     %5t       %3d    %3d",
			eCompleted, eReceived, eTotal,
			completed, received, total)
	}

	return nil
}

// checkReceivedTracker checks that the receivedPartTracker is reporting the
// correct values for each part. Also checks that
// receivedPartTracker.GetNumParts returns the expected value (make sure
// numParts comes from a correct source).
func checkReceivedTracker(track interfaces.FilePartTracker, numParts uint16,
	received []uint16, t *testing.T) {
	if track.GetNumParts() != numParts {
		t.Errorf("Tracker reported incorrect number of parts."+
			"\nexpected: %d\nreceived: %d", numParts, track.GetNumParts())
		return
	}

	for partNum := uint16(0); partNum < numParts; partNum++ {
		var done bool
		for _, receivedNum := range received {
			if receivedNum == partNum {
				if track.GetPartStatus(partNum) != interfaces.FpReceived {
					t.Errorf("Part number %d has unexpected status."+
						"\nexpected: %d\nreceived: %d", partNum,
						interfaces.FpReceived, track.GetPartStatus(partNum))
				}
				done = true
				break
			}
		}
		if done {
			continue
		}

		if track.GetPartStatus(partNum) != interfaces.FpUnsent {
			t.Errorf("Part number %d has incorrect status."+
				"\nexpected: %d\nreceived: %d",
				partNum, interfaces.FpUnsent, track.GetPartStatus(partNum))
		}
	}
}

// Tests that ReceivedTransfer.GetProgress returns the expected progress metrics
// for various transfer states.
func TestReceivedTransfer_GetProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	_, rt, _ := newEmptyReceivedTransfer(numParts, 20, kv, t)

	completed, received, total, track := rt.GetProgress()
	err := checkReceivedProgress(completed, received, total, false, 0, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(track, rt.numParts, nil, t)

	_, _ = rt.fpVector.Next()
	_, _ = rt.receivedStatus.Next()

	completed, received, total, track = rt.GetProgress()
	err = checkReceivedProgress(completed, received, total, false, 1, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(track, rt.numParts, []uint16{0}, t)

	for i := 0; i < 4; i++ {
		_, _ = rt.fpVector.Next()
		_, _ = rt.receivedStatus.Next()
	}

	completed, received, total, track = rt.GetProgress()
	err = checkReceivedProgress(completed, received, total, false, 5, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(track, rt.numParts, []uint16{0, 1, 2, 3, 4}, t)

	for i := 0; i < 6; i++ {
		_, _ = rt.fpVector.Next()
		_, _ = rt.receivedStatus.Next()
	}

	completed, received, total, track = rt.GetProgress()
	err = checkReceivedProgress(completed, received, total, false, 11, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(
		track, rt.numParts, []uint16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, t)

	for i := 0; i < 4; i++ {
		_, _ = rt.fpVector.Next()
		_, _ = rt.receivedStatus.Next()
	}

	completed, received, total, track = rt.GetProgress()
	err = checkReceivedProgress(completed, received, total, false, 15, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(track, rt.numParts, []uint16{0, 1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14}, t)

	_, _ = rt.fpVector.Next()
	_, _ = rt.receivedStatus.Next()

	completed, received, total, track = rt.GetProgress()
	err = checkReceivedProgress(completed, received, total, true, 16, numParts)
	if err != nil {
		t.Error(err)
	}
	checkReceivedTracker(track, rt.numParts, []uint16{0, 1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15}, t)
}

// Tests that 5 different callbacks all receive the expected data when
// ReceivedTransfer.CallProgressCB is called at different stages of transfer.
func TestReceivedTransfer_CallProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	type progressResults struct {
		completed       bool
		received, total uint16
		tr              interfaces.FilePartTracker
		err             error
	}

	period := time.Millisecond

	wg := sync.WaitGroup{}
	var step0, step1, step2, step3 uint64
	numCallbacks := 5

	for i := 0; i < numCallbacks; i++ {
		progressChan := make(chan progressResults)

		cbFunc := func(completed bool, received, total uint16,
			tr interfaces.FilePartTracker, err error) {
			progressChan <- progressResults{completed, received, total, tr, err}
		}
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			n := 0
			for {
				select {
				case <-time.NewTimer(time.Second).C:
					t.Errorf("Timed out after %s waiting for callback (%d).",
						period*5, i)
					return
				case r := <-progressChan:
					switch n {
					case 0:
						if err := checkReceivedProgress(r.completed, r.received,
							r.total, false, 0, rt.numParts); err != nil {
							t.Errorf("%2d: %v", i, err)
						}
						atomic.AddUint64(&step0, 1)
					case 1:
						if err := checkReceivedProgress(r.completed, r.received,
							r.total, false, 0, rt.numParts); err != nil {
							t.Errorf("%2d: %v", i, err)
						}
						atomic.AddUint64(&step1, 1)
					case 2:
						if err := checkReceivedProgress(r.completed, r.received,
							r.total, false, 4, rt.numParts); err != nil {
							t.Errorf("%2d: %v", i, err)
						}
						atomic.AddUint64(&step2, 1)
					case 3:
						if err := checkReceivedProgress(r.completed, r.received,
							r.total, true, 16, rt.numParts); err != nil {
							t.Errorf("%2d: %v", i, err)
						}
						atomic.AddUint64(&step3, 1)
						return
					default:
						t.Errorf("n (%d) is great than 3 (%d)", n, i)
						return
					}
					n++
				}
			}
		}(i)

		rt.AddProgressCB(cbFunc, period)
	}

	for !atomic.CompareAndSwapUint64(&step0, uint64(numCallbacks), 0) {
	}

	rt.CallProgressCB(nil)

	for !atomic.CompareAndSwapUint64(&step1, uint64(numCallbacks), 0) {
	}

	for i := 0; i < 4; i++ {
		_, _ = rt.receivedStatus.Next()
	}

	rt.CallProgressCB(nil)

	for !atomic.CompareAndSwapUint64(&step2, uint64(numCallbacks), 0) {
	}

	for i := 0; i < 12; i++ {
		_, _ = rt.receivedStatus.Next()
	}

	rt.CallProgressCB(nil)

	wg.Wait()
}

// Tests that ReceivedTransfer.stopScheduledProgressCB stops a scheduled
// callback from being triggered.
func TestReceivedTransfer_stopScheduledProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	cbChan := make(chan struct{}, 5)
	cbFunc := interfaces.ReceivedProgressCallback(
		func(completed bool, received, total uint16,
			t interfaces.FilePartTracker, err error) {
			cbChan <- struct{}{}
		})
	rt.AddProgressCB(cbFunc, 150*time.Millisecond)
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback.")
	case <-cbChan:
	}

	rt.CallProgressCB(nil)
	rt.CallProgressCB(nil)
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback.")
	case <-cbChan:
	}

	err := rt.stopScheduledProgressCB()
	if err != nil {
		t.Errorf("stopScheduledProgressCB returned an error: %+v", err)
	}

	select {
	case <-time.NewTimer(200 * time.Millisecond).C:
	case <-cbChan:
		t.Error("Callback called when it should have been stopped.")
	}
}

// Tests that ReceivedTransfer.AddProgressCB adds an item to the progress
// callback list.
func TestReceivedTransfer_AddProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	type callbackResults struct {
		completed       bool
		received, total uint16
		err             error
	}
	cbChan := make(chan callbackResults)
	cbFunc := interfaces.ReceivedProgressCallback(
		func(completed bool, received, total uint16,
			t interfaces.FilePartTracker, err error) {
			cbChan <- callbackResults{completed, received, total, err}
		})

	done := make(chan bool)
	go func() {
		select {
		case <-time.NewTimer(time.Millisecond).C:
			t.Error("Timed out waiting for progress callback to be called.")
		case r := <-cbChan:
			err := checkReceivedProgress(
				r.completed, r.received, r.total, false, 0, 16)
			if err != nil {
				t.Error(err)
			}
			if r.err != nil {
				t.Errorf("Callback returned an error: %+v", err)
			}
		}
		done <- true
	}()

	period := time.Millisecond
	rt.AddProgressCB(cbFunc, period)

	if len(rt.progressCallbacks) != 1 {
		t.Errorf("Callback list should only have one item."+
			"\nexpected: %d\nreceived: %d", 1, len(rt.progressCallbacks))
	}

	if rt.progressCallbacks[0].period != period {
		t.Errorf("Callback has wrong lastCall.\nexpected: %s\nreceived: %s",
			period, rt.progressCallbacks[0].period)
	}

	if rt.progressCallbacks[0].lastCall != (time.Time{}) {
		t.Errorf("Callback has wrong time.\nexpected: %s\nreceived: %s",
			time.Time{}, rt.progressCallbacks[0].lastCall)
	}

	if rt.progressCallbacks[0].scheduled {
		t.Errorf("Callback has wrong scheduled.\nexpected: %t\nreceived: %t",
			false, rt.progressCallbacks[0].scheduled)
	}
	<-done
}

// Tests that ReceivedTransfer.AddPart adds a part in the correct place in the
// list of parts.
func TestReceivedTransfer_AddPart(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numFps := uint16(20)
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	expectedData := []byte("test")

	partNum, fpNum := uint16(1), uint16(1)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(expectedData)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	complete, err := rt.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Errorf("AddPart returned an error: %+v", err)
	}

	if complete {
		t.Errorf("Transfer complete when it should not be.")
	}

	receivedData, exists := rt.receivedParts.parts[partNum]
	if !exists {
		t.Errorf("Part #%d not found in part map.", partNum)
	} else if !bytes.Equal(expectedData, receivedData[:len(expectedData)]) {
		t.Fatalf("Part data in list does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedData, receivedData[:len(expectedData)])
	}

	// Check that the fingerprint vector has correct values
	if rt.fpVector.GetNumAvailable() != uint32(numFps-1) {
		t.Errorf("Incorrect number of available keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, rt.fpVector.GetNumAvailable())
	}
	if rt.fpVector.GetNumKeys() != uint32(numFps) {
		t.Errorf("Incorrect number of keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, rt.fpVector.GetNumKeys())
	}
	if rt.fpVector.GetNumUsed() != 1 {
		t.Errorf("Incorrect number of used keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", 1, rt.fpVector.GetNumUsed())
	}

	// Check that the part was properly marked as received on the received
	// status vector
	if !rt.receivedStatus.Used(uint32(partNum)) {
		t.Errorf("Part number %d not marked as used in received status vector.",
			partNum)
	}
	if rt.receivedStatus.GetNumUsed() != 1 {
		t.Errorf("Incorrect number of received parts in vector."+
			"\nexpected: %d\nreceived: %d", 1, rt.receivedStatus.GetNumUsed())
	}
}

// Error path: tests that ReceivedTransfer.AddPart returns the expected error
// when the provided MAC is invalid.
func TestReceivedTransfer_AddPart_DecryptPartError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	expectedData := []byte("test")

	partNum, fpNum := uint16(1), uint16(1)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(expectedData)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, _, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)
	badMac := make([]byte, format.MacLen)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(badMac)

	// Add encrypted part
	expectedErr := "reconstructed MAC from decrypting does not match MAC from sender"
	_, err = rt.AddPart(cmixMsg, fpNum)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("AddPart did not return the expected error when the MAC is "+
			"invalid.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that ReceivedTransfer.GetFile returns the expected file data.
func TestReceivedTransfer_GetFile(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, expectedFile := newRandomReceivedTransfer(16, 20, kv, t)

	receivedFile, err := rt.GetFile()
	if err != nil {
		t.Errorf("GetFile returned an error: %+v", err)
	}

	// Check that the received file is correct
	if !bytes.Equal(expectedFile, receivedFile) {
		t.Errorf("File does not match expected.\nexpected: %+v\nreceived: %+v",
			expectedFile, receivedFile)
	}
}

// Error path: tests that ReceivedTransfer.GetFile returns the expected error
// when not all file parts are present.
func TestReceivedTransfer_GetFile_MissingPartsError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	_, rt, _ := newEmptyReceivedTransfer(numParts, 20, kv, t)
	expectedErr := fmt.Sprintf(getFileErr, numParts, numParts)

	_, err := rt.GetFile()
	if err == nil || err.Error() != expectedErr {
		t.Errorf("GetFile failed to return the expected error when all file "+
			"parts are missing.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that ReceivedTransfer.GetFile returns the expected error
// when the stored transfer MAC does not match the one generated from the file.
func TestReceivedTransfer_GetFile_InvalidMacError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newRandomReceivedTransfer(16, 20, kv, t)

	rt.transferMAC = []byte("invalidMAC")

	_, err := rt.GetFile()
	if err == nil || err.Error() != getTransferMacErr {
		t.Errorf("GetFile failed to return the expected error when the file "+
			"MAC cannot be verified.\nexpected: %s\nreceived: %+v",
			getTransferMacErr, err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Function Testing                                                   //
////////////////////////////////////////////////////////////////////////////////

// Tests that loadReceivedTransfer returns a ReceivedTransfer that matches the
// original object in memory.
func Test_loadReceivedTransfer(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, expectedRT, _ := newRandomReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	expectedData := []byte("test")
	partNum, fpNum := uint16(1), uint16(1)

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(expectedData)

	fp := ftCrypto.GenerateFingerprint(expectedRT.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(expectedRT.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	_, err = expectedRT.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Errorf("Failed to add test part: %+v", err)
	}

	loadedRT, err := loadReceivedTransfer(tid, kv)
	if err != nil {
		t.Errorf("loadReceivedTransfer returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedRT, loadedRT) {
		t.Errorf("Loaded ReceivedTransfer does not match expected"+
			".\nexpected: %+v\nreceived: %+v", expectedRT, loadedRT)
	}
}

// Error path: tests that loadReceivedTransfer returns the expected error when
// no info is in storage to load.
func Test_loadReceivedTransfer_LoadInfoError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidTransferID"))
	expectedErr := strings.Split(loadReceivedStoreErr, "%")[0]

	_, err := loadReceivedTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadReceivedTransfer did not return the expected error when "+
			"trying to load info from storage that does not exist."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadReceivedTransfer returns the expected error when
// the fingerprint state vector has been deleted from storage.
func Test_loadReceivedTransfer_LoadFpVectorError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, rt, _ := newRandomReceivedTransfer(16, 20, kv, t)

	// Create encrypted part

	data := []byte("test")
	partNum, fpNum := uint16(1), uint16(1)

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(data)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	_, err = rt.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Errorf("Failed to add test part: %+v", err)
	}

	// Delete fingerprint state vector from storage
	err = rt.fpVector.Delete()
	if err != nil {
		t.Errorf("Failed to delete fingerprint vector: %+v", err)
	}

	expectedErr := strings.Split(loadReceiveFpVectorErr, "%")[0]
	_, err = loadReceivedTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadReceivedTransfer did not return the expected error when "+
			"the state vector was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadReceivedTransfer returns the expected error when
// the part store has been deleted from storage.
func Test_loadReceivedTransfer_LoadPartStoreError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, rt, _ := newRandomReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	data := []byte("test")
	partNum, fpNum := uint16(1), uint16(1)

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(data)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	_, err = rt.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Errorf("Failed to add test part: %+v", err)
	}

	// Delete fingerprint state vector from storage
	err = rt.receivedParts.delete()
	if err != nil {
		t.Errorf("Failed to delete part store: %+v", err)
	}

	expectedErr := strings.Split(loadReceivePartStoreErr, "%")[0]
	_, err = loadReceivedTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadReceivedTransfer did not return the expected error when "+
			"the part store was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadReceivedTransfer returns the expected error when
// the received status state vector has been deleted from storage.
func Test_loadReceivedTransfer_LoadReceivedVectorError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, rt, _ := newRandomReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	data := []byte("test")
	partNum, fpNum := uint16(1), uint16(1)

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(data)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	_, err = rt.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Errorf("Failed to add test part: %+v", err)
	}

	// Delete fingerprint state vector from storage
	err = rt.receivedStatus.Delete()
	if err != nil {
		t.Errorf("Failed to received status vector: %+v", err)
	}

	expectedErr := strings.Split(loadReceivedVectorErr, "%")[0]
	_, err = loadReceivedTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadReceivedTransfer did not return the expected error when "+
			"the received status vector was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that ReceivedTransfer.saveInfo saves the expected data to storage.
func TestReceivedTransfer_saveInfo(t *testing.T) {
	rt := &ReceivedTransfer{
		key:         ftCrypto.UnmarshalTransferKey([]byte("key")),
		transferMAC: []byte("transferMAC"),
		numParts:    16,
		kv:          versioned.NewKV(make(ekv.Memstore)),
	}

	err := rt.saveInfo()
	if err != nil {
		t.Fatalf("saveInfo returned an error: %v", err)
	}

	vo, err := rt.kv.Get(receivedTransferKey, receivedTransferVersion)
	if err != nil {
		t.Fatalf("Failed to load ReceivedTransfer from storage: %+v", err)
	}

	if !bytes.Equal(rt.marshal(), vo.Data) {
		t.Errorf("Marshalled data loaded from storage does not match expected."+
			"\nexpected: %+v\nreceived: %+v", rt.marshal(), vo.Data)
	}
}

// Tests that ReceivedTransfer.loadInfo loads a saved ReceivedTransfer from
// storage.
func TestReceivedTransfer_loadInfo(t *testing.T) {
	rt := &ReceivedTransfer{
		key:         ftCrypto.UnmarshalTransferKey([]byte("key")),
		transferMAC: []byte("transferMAC"),
		numParts:    16,
		kv:          versioned.NewKV(make(ekv.Memstore)),
	}

	err := rt.saveInfo()
	if err != nil {
		t.Errorf("failed to save new ReceivedTransfer to storage: %+v", err)
	}

	loadedRT := &ReceivedTransfer{kv: rt.kv}
	err = loadedRT.loadInfo()
	if err != nil {
		t.Errorf("load returned an error: %+v", err)
	}

	if !reflect.DeepEqual(rt, loadedRT) {
		t.Errorf("Loaded ReceivedTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", rt, loadedRT)
	}
}

// Error path: tests that ReceivedTransfer.loadInfo returns an error when there
// is no object in storage to load
func TestReceivedTransfer_loadInfo_Error(t *testing.T) {
	loadedRT := &ReceivedTransfer{kv: versioned.NewKV(make(ekv.Memstore))}
	err := loadedRT.loadInfo()
	if err == nil {
		t.Errorf("Loaded object that should not be in storage: %+v", err)
	}
}

// Tests that ReceivedTransfer.delete removes all data from storage.
func TestReceivedTransfer_delete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newRandomReceivedTransfer(16, 20, kv, t)

	// Create encrypted part
	expectedData := []byte("test")
	partNum, fpNum := uint16(1), uint16(1)

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(expectedData)

	fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	_, err = rt.AddPart(cmixMsg, fpNum)
	if err != nil {
		t.Fatalf("Failed to add test part: %+v", err)
	}

	// Delete everything from storage
	err = rt.delete()
	if err != nil {
		t.Errorf("delete returned an error: %+v", err)
	}

	// Check that the SentTransfer info was deleted
	err = rt.loadInfo()
	if err == nil {
		t.Error("Successfully loaded SentTransfer info from storage when it " +
			"should have been deleted.")
	}

	// Check that the parts store were deleted
	_, err = loadPartStore(rt.kv)
	if err == nil {
		t.Error("Successfully loaded file parts from storage when it should " +
			"have been deleted.")
	}

	// Check that the fingerprint vector was deleted
	_, err = utility.LoadStateVector(rt.kv, receivedFpVectorKey)
	if err == nil {
		t.Error("Successfully loaded fingerprint vector from storage when it " +
			"should have been deleted.")
	}

	// Check that the received status vector was deleted
	_, err = utility.LoadStateVector(rt.kv, receivedVectorKey)
	if err == nil {
		t.Error("Successfully loaded received status vector from storage when " +
			"it should have been deleted.")
	}
}

// Tests that ReceivedTransfer.deleteInfo removes the saved ReceivedTransfer
// data from storage.
func TestReceivedTransfer_deleteInfo(t *testing.T) {
	rt := &ReceivedTransfer{
		key:         ftCrypto.UnmarshalTransferKey([]byte("key")),
		transferMAC: []byte("transferMAC"),
		numParts:    16,
		kv:          versioned.NewKV(make(ekv.Memstore)),
	}

	// Save from storage
	err := rt.saveInfo()
	if err != nil {
		t.Errorf("failed to save new ReceivedTransfer to storage: %+v", err)
	}

	// Delete from storage
	err = rt.deleteInfo()
	if err != nil {
		t.Errorf("deleteInfo returned an error: %+v", err)
	}

	// Make sure deleted object cannot be loaded from storage
	_, err = rt.kv.Get(receivedTransferKey, receivedTransferVersion)
	if err == nil {
		t.Error("Loaded object that should be deleted from storage.")
	}
}

// Tests that a ReceivedTransfer marshalled with ReceivedTransfer.marshal and
// then unmarshalled with unmarshalReceivedTransfer matches the original.
func TestReceivedTransfer_marshal_unmarshalReceivedTransfer(t *testing.T) {
	rt := &ReceivedTransfer{
		key:         ftCrypto.UnmarshalTransferKey([]byte("key")),
		transferMAC: []byte("transferMAC"),
		fileSize:    256,
		numParts:    16,
		numFps:      20,
	}

	marshaledData := rt.marshal()
	key, mac, fileSize, numParts, numFps := unmarshalReceivedTransfer(marshaledData)

	if rt.key != key {
		t.Errorf("Failed to get expected key.\nexpected: %s\nreceived: %s",
			rt.key, key)
	}

	if !bytes.Equal(rt.transferMAC, mac) {
		t.Errorf("Failed to get expected transfer MAC."+
			"\nexpected: %v\nreceived: %v", rt.transferMAC, mac)
	}

	if rt.fileSize != fileSize {
		t.Errorf("Failed to get expected file size."+
			"\nexpected: %d\nreceived: %d", rt.fileSize, fileSize)
	}

	if rt.numParts != numParts {
		t.Errorf("Failed to get expected number of parts."+
			"\nexpected: %d\nreceived: %d", rt.numParts, numParts)
	}

	if rt.numFps != numFps {
		t.Errorf("Failed to get expected number of fingerprints."+
			"\nexpected: %d\nreceived: %d", rt.numFps, numFps)
	}
}

// Consistency test: tests that makeReceivedTransferPrefix returns the expected
// prefixes for the provided transfer IDs.
func Test_makeReceivedTransferPrefix_Consistency(t *testing.T) {
	prng := NewPrng(42)
	expectedPrefixes := []string{
		"FileTransferReceivedTransferStoreU4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=",
		"FileTransferReceivedTransferStore39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=",
		"FileTransferReceivedTransferStoreCD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=",
		"FileTransferReceivedTransferStoreuoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=",
		"FileTransferReceivedTransferStoreGwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=",
		"FileTransferReceivedTransferStorernvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=",
		"FileTransferReceivedTransferStoreceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=",
		"FileTransferReceivedTransferStoreSYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=",
		"FileTransferReceivedTransferStoreNhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=",
		"FileTransferReceivedTransferStorekM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog=",
	}

	for i, expected := range expectedPrefixes {
		tid, _ := ftCrypto.NewTransferID(prng)
		prefix := makeReceivedTransferPrefix(tid)

		if expected != prefix {
			t.Errorf("New ReceivedTransfer prefix does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, prefix)
		}
	}
}

// newRandomReceivedTransfer generates a new ReceivedTransfer with random data.
// Returns the generated transfer ID, new ReceivedTransfer, and the full file.
func newRandomReceivedTransfer(numParts, numFps uint16, kv *versioned.KV,
	t *testing.T) (ftCrypto.TransferID, *ReceivedTransfer, []byte) {

	prng := NewPrng(42)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	parts, fileData := newRandomPartStore(numParts, kv, prng, t)
	fileSize := uint32(len(fileData))
	mac := ftCrypto.CreateTransferMAC(fileData, key)

	rt, err := NewReceivedTransfer(tid, key, mac, fileSize, numParts, numFps, kv)
	if err != nil {
		t.Errorf("Failed to create new ReceivedTransfer: %+v", err)
	}

	for partNum, part := range parts.parts {
		cmixMsg := format.NewMessage(format.MinimumPrimeSize)

		partData, _ := NewPartMessage(cmixMsg.ContentsSize())
		partData.SetPartNum(partNum)
		_ = partData.SetPart(part)

		fp := ftCrypto.GenerateFingerprint(rt.key, partNum)
		encryptedPart, mac, err := ftCrypto.EncryptPart(rt.key, partData.Marshal(), partNum, fp)

		cmixMsg.SetKeyFP(fp)
		cmixMsg.SetContents(encryptedPart)
		cmixMsg.SetMac(mac)

		_, err = rt.AddPart(cmixMsg, partNum)
		if err != nil {
			t.Errorf("Failed to add part #%d: %+v", partNum, err)
		}
	}

	return tid, rt, fileData
}

// newRandomReceivedTransfer generates a new empty ReceivedTransfer. Returns the
// generated transfer ID, new ReceivedTransfer, and the full file.
func newEmptyReceivedTransfer(numParts, numFps uint16, kv *versioned.KV,
	t *testing.T) (ftCrypto.TransferID, *ReceivedTransfer, []byte) {

	prng := NewPrng(42)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, fileData := newRandomPartStore(numParts, kv, prng, t)
	fileSize := uint32(len(fileData))

	mac := ftCrypto.CreateTransferMAC(fileData, key)

	rt, err := NewReceivedTransfer(tid, key, mac, fileSize, numParts, numFps, kv)
	if err != nil {
		t.Errorf("Failed to create new ReceivedTransfer: %+v", err)
	}

	return tid, rt, fileData
}
