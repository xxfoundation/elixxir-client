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
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that NewSentTransfer creates the expected SentTransfer and that it is
// saved to storage.
func Test_NewSentTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	recipient, _ := id.NewRandomID(prng, id.User)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	kvPrefixed := kv.Prefix(makeSentTransferPrefix(tid))
	parts := [][]byte{
		[]byte("test0"), []byte("test1"), []byte("test2"),
		[]byte("test3"), []byte("test4"), []byte("test5"),
	}
	numParts, numFps := uint16(len(parts)), uint16(float64(len(parts))*1.5)
	fpVector, _ := utility.NewStateVector(
		kvPrefixed, sentFpVectorKey, uint32(numFps))
	partStats, _ := utility.NewMultiStateVector(
		numParts, 3, sentTransferStateMap, sentPartStatsVectorKey, kvPrefixed)

	type cbFields struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	expectedCB := cbFields{
		completed: false,
		sent:      0,
		arrived:   0,
		total:     numParts,
		err:       nil,
	}

	cbChan := make(chan cbFields)
	cb := func(completed bool, sent, arrived, total uint16,
		t interfaces.FilePartTracker, err error) {
		cbChan <- cbFields{
			completed: completed,
			sent:      sent,
			arrived:   arrived,
			total:     total,
			err:       err,
		}
	}

	expectedPeriod := time.Second

	expected := &SentTransfer{
		recipient: recipient,
		key:       key,
		numParts:  numParts,
		numFps:    numFps,
		fpVector:  fpVector,
		sentParts: &partStore{
			parts:    partSliceToMap(parts...),
			numParts: uint16(len(parts)),
			kv:       kvPrefixed,
		},
		inProgressTransfers: &transferredBundle{
			list: make(map[id.Round][]uint16),
			key:  inProgressKey,
			kv:   kvPrefixed,
		},
		finishedTransfers: &transferredBundle{
			list: make(map[id.Round][]uint16),
			key:  finishedKey,
			kv:   kvPrefixed,
		},
		partStats: partStats,
		progressCallbacks: []*sentCallbackTracker{
			newSentCallbackTracker(cb, expectedPeriod),
		},
		status: Running,
		kv:     kvPrefixed,
	}

	// Create new SentTransfer
	st, err := NewSentTransfer(
		recipient, tid, key, parts, numFps, cb, expectedPeriod, kv)
	if err != nil {
		t.Errorf("NewSentTransfer returned an error: %+v", err)
	}

	// Check that the callback is called when added
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting fpr progress callback to be called.")
	case cbResults := <-cbChan:
		if !reflect.DeepEqual(expectedCB, cbResults) {
			t.Errorf("Did not receive correct results from callback."+
				"\nexpected: %+v\nreceived: %+v", expectedCB, cbResults)
		}
	}

	st.progressCallbacks = expected.progressCallbacks

	// Check that the new object matches the expected
	if !reflect.DeepEqual(expected, st) {
		t.Errorf("New SentTransfer does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, st)
	}

	// Make sure it is saved to storage
	_, err = kvPrefixed.Get(sentTransferKey, sentTransferVersion)
	if err != nil {
		t.Errorf("Failed to get new SentTransfer from storage: %+v", err)
	}

	// Check that the fingerprint vector has correct values
	if st.fpVector.GetNumAvailable() != uint32(numFps) {
		t.Errorf("Incorrect number of available keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, st.fpVector.GetNumAvailable())
	}
	if st.fpVector.GetNumKeys() != uint32(numFps) {
		t.Errorf("Incorrect number of keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps, st.fpVector.GetNumKeys())
	}
	if st.fpVector.GetNumUsed() != 0 {
		t.Errorf("Incorrect number of used keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", 0, st.fpVector.GetNumUsed())
	}
}

// Tests that SentTransfer.ReInit overwrites the fingerprint vector, in-progress
// transfer, finished transfers, and progress callbacks with new and empty
// objects.
func TestSentTransfer_ReInit(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	recipient, _ := id.NewRandomID(prng, id.User)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	kvPrefixed := kv.Prefix(makeSentTransferPrefix(tid))
	parts := [][]byte{
		[]byte("test0"), []byte("test1"), []byte("test2"),
		[]byte("test3"), []byte("test4"), []byte("test5"),
	}
	numParts, numFps1 := uint16(len(parts)), uint16(float64(len(parts))*1.5)
	numFps2 := 2 * numFps1
	fpVector, _ := utility.NewStateVector(
		kvPrefixed, sentFpVectorKey, uint32(numFps2))
	partStats, _ := utility.NewMultiStateVector(
		numParts, 3, sentTransferStateMap, sentPartStatsVectorKey, kvPrefixed)

	type cbFields struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	expectedCB := cbFields{
		completed: false,
		sent:      0,
		arrived:   0,
		total:     numParts,
		err:       nil,
	}

	cbChan := make(chan cbFields)
	cb := func(completed bool, sent, arrived, total uint16,
		t interfaces.FilePartTracker, err error) {
		cbChan <- cbFields{
			completed: completed,
			sent:      sent,
			arrived:   arrived,
			total:     total,
			err:       err,
		}
	}

	expectedPeriod := time.Millisecond

	expected := &SentTransfer{
		recipient: recipient,
		key:       key,
		numParts:  numParts,
		numFps:    numFps2,
		fpVector:  fpVector,
		sentParts: &partStore{
			parts:    partSliceToMap(parts...),
			numParts: uint16(len(parts)),
			kv:       kvPrefixed,
		},
		inProgressTransfers: &transferredBundle{
			list: make(map[id.Round][]uint16),
			key:  inProgressKey,
			kv:   kvPrefixed,
		},
		finishedTransfers: &transferredBundle{
			list: make(map[id.Round][]uint16),
			key:  finishedKey,
			kv:   kvPrefixed,
		},
		partStats: partStats,
		progressCallbacks: []*sentCallbackTracker{
			newSentCallbackTracker(cb, expectedPeriod),
		},
		status: Running,
		kv:     kvPrefixed,
	}

	// Create new SentTransfer
	st, err := NewSentTransfer(
		recipient, tid, key, parts, numFps1, nil, 2*expectedPeriod, kv)
	if err != nil {
		t.Errorf("NewSentTransfer returned an error: %+v", err)
	}

	// Re-initialize SentTransfer with new number of fingerprints and callback
	err = st.ReInit(numFps2, cb, expectedPeriod)
	if err != nil {
		t.Errorf("ReInit returned an error: %+v", err)
	}

	// Check that the callback is called when added
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting fpr progress callback to be called.")
	case cbResults := <-cbChan:
		if !reflect.DeepEqual(expectedCB, cbResults) {
			t.Errorf("Did not receive correct results from callback."+
				"\nexpected: %+v\nreceived: %+v", expectedCB, cbResults)
		}
	}

	st.progressCallbacks = expected.progressCallbacks

	// Check that the new object matches the expected
	if !reflect.DeepEqual(expected, st) {
		t.Errorf("New SentTransfer does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, st)
	}

	// Make sure it is saved to storage
	_, err = kvPrefixed.Get(sentTransferKey, sentTransferVersion)
	if err != nil {
		t.Errorf("Failed to get new SentTransfer from storage: %+v", err)
	}

	// Check that the fingerprint vector has correct values
	if st.fpVector.GetNumAvailable() != uint32(numFps2) {
		t.Errorf("Incorrect number of available keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps2, st.fpVector.GetNumAvailable())
	}
	if st.fpVector.GetNumKeys() != uint32(numFps2) {
		t.Errorf("Incorrect number of keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", numFps2, st.fpVector.GetNumKeys())
	}
	if st.fpVector.GetNumUsed() != 0 {
		t.Errorf("Incorrect number of used keys in fingerprint list."+
			"\nexpected: %d\nreceived: %d", 0, st.fpVector.GetNumUsed())
	}
}

// Tests that SentTransfer.GetRecipient returns the expected ID.
func TestSentTransfer_GetRecipient(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedRecipient, _ := id.NewRandomID(prng, id.User)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)

	// Create new SentTransfer
	st, err := NewSentTransfer(
		expectedRecipient, tid, key, [][]byte{}, 5, nil, 0, kv)
	if err != nil {
		t.Errorf("Failed to create new SentTransfer: %+v", err)
	}

	if expectedRecipient != st.GetRecipient() {
		t.Errorf("Failed to get expected transfer key."+
			"\nexpected: %s\nreceived: %s", expectedRecipient, st.GetRecipient())
	}
}

// Tests that SentTransfer.GetTransferKey returns the expected transfer key.
func TestSentTransfer_GetTransferKey(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	recipient, _ := id.NewRandomID(prng, id.User)
	tid, _ := ftCrypto.NewTransferID(prng)
	expectedKey, _ := ftCrypto.NewTransferKey(prng)

	// Create new SentTransfer
	st, err := NewSentTransfer(
		recipient, tid, expectedKey, [][]byte{}, 5, nil, 0, kv)
	if err != nil {
		t.Errorf("Failed to create new SentTransfer: %+v", err)
	}

	if expectedKey != st.GetTransferKey() {
		t.Errorf("Failed to get expected transfer key."+
			"\nexpected: %s\nreceived: %s", expectedKey, st.GetTransferKey())
	}
}

// Tests that SentTransfer.GetNumParts returns the expected number of parts.
func TestSentTransfer_GetNumParts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedNumParts := uint16(16)
	_, st := newRandomSentTransfer(expectedNumParts, 24, kv, t)

	if expectedNumParts != st.GetNumParts() {
		t.Errorf("Failed to get expected number of parts."+
			"\nexpected: %d\nreceived: %d", expectedNumParts, st.GetNumParts())
	}
}

// Tests that SentTransfer.GetNumFps returns the expected number of
// fingerprints.
func TestSentTransfer_GetNumFps(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedNumFps := uint16(24)
	_, st := newRandomSentTransfer(16, expectedNumFps, kv, t)

	if expectedNumFps != st.GetNumFps() {
		t.Errorf("Failed to get expected number of fingerprints."+
			"\nexpected: %d\nreceived: %d", expectedNumFps, st.GetNumFps())
	}
}

// Tests that SentTransfer.GetNumAvailableFps returns the expected number of
// available fingerprints.
func TestSentTransfer_GetNumAvailableFps(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts, numFps := uint16(16), uint16(24)
	_, st := newRandomSentTransfer(numParts, numFps, kv, t)

	if numFps != st.GetNumAvailableFps() {
		t.Errorf("Failed to get expected number of available fingerprints."+
			"\nexpected: %d\nreceived: %d",
			numFps, st.GetNumAvailableFps())
	}

	for i := uint16(0); i < numParts; i++ {
		_, _ = st.fpVector.Next()
	}

	if numFps-numParts != st.GetNumAvailableFps() {
		t.Errorf("Failed to get expected number of available fingerprints."+
			"\nexpected: %d\nreceived: %d",
			numFps-numParts, st.GetNumAvailableFps())
	}
}

// Tests that SentTransfer.GetStatus returns the expected status at each stage
// of the transfer.
func TestSentTransfer_GetStatus(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts, numFps := uint16(2), uint16(4)
	_, st := newRandomSentTransfer(numParts, numFps, kv, t)

	status := st.GetStatus()
	if status != Running {
		t.Errorf("Unexpected transfer status.\nexpected: %s\nreceived: %s",
			Running, status)
	}

	_, _ = st.SetInProgress(0, 0, 1)
	_, _ = st.FinishTransfer(0)

	status = st.GetStatus()
	if status != Stopping {
		t.Errorf("Unexpected transfer status.\nexpected: %s\nreceived: %s",
			Stopping, status)
	}

	st.CallProgressCB(nil)

	status = st.GetStatus()
	if status != Stopped {
		t.Errorf("Unexpected transfer status.\nexpected: %s\nreceived: %s",
			Stopped, status)
	}
}

// Tests that SentTransfer.IsPartInProgress returns false before a part is set
// as in-progress and true after it is set via SentTransfer.SetInProgress. Also
// tests that it returns false after the part has been unset via
// SentTransfer.UnsetInProgress.
func TestSentTransfer_IsPartInProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(0)
	partNum := uint16(7)

	// Test that the part has not been set to in-progress
	inProgress, err := st.IsPartInProgress(partNum)
	if err != nil {
		t.Errorf("IsPartInProgress returned an error: %+v", err)
	}
	if inProgress {
		t.Errorf("Part number %d set as in-progress.", partNum)
	}

	// Set the part number to in-progress
	_, _ = st.SetInProgress(rid, partNum)

	// Test that the part has been set to in-progress
	inProgress, err = st.IsPartInProgress(partNum)
	if err != nil {
		t.Errorf("IsPartInProgress returned an error: %+v", err)
	}
	if !inProgress {
		t.Errorf("Part number %d not set as in-progress.", partNum)
	}

	// Unset the part as in-progress
	_, _ = st.UnsetInProgress(rid)

	// Test that the part has been unset
	inProgress, err = st.IsPartInProgress(partNum)
	if err != nil {
		t.Errorf("IsPartInProgress returned an error: %+v", err)
	}
	if inProgress {
		t.Errorf("Part number %d set as in-progress.", partNum)
	}
}

// Error path: tests that SentTransfer.IsPartInProgress returns the expected
// error when the part number is out of range.
func TestSentTransfer_IsPartInProgress_InvalidPartNumError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	expectedErr := fmt.Sprintf(getStatusErr, st.numParts, "")
	_, err := st.IsPartInProgress(st.numParts)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("IsPartInProgress did not return the expected error when the "+
			"part number is out of range.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that SentTransfer.IsPartFinished returns false before a part is set as
// finished and true after it is set via SentTransfer.FinishTransfer.
func TestSentTransfer_IsPartFinished(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(0)
	partNum := uint16(7)

	// Set the part number to in-progress
	_, _ = st.SetInProgress(rid, partNum)

	// Test that the part has not been set to finished
	isFinished, err := st.IsPartFinished(partNum)
	if err != nil {
		t.Errorf("IsPartFinished returned an error: %+v", err)
	}
	if isFinished {
		t.Errorf("Part number %d set as finished.", partNum)
	}

	// Set the part number to finished
	_, _ = st.FinishTransfer(rid)

	// Test that the part has been set to finished
	isFinished, err = st.IsPartFinished(partNum)
	if err != nil {
		t.Errorf("IsPartFinished returned an error: %+v", err)
	}
	if !isFinished {
		t.Errorf("Part number %d not set as finished.", partNum)
	}
}

// Error path: tests that SentTransfer.IsPartFinished returns the expected
// error when the part number is out of range.
func TestSentTransfer_IsPartFinished_InvalidPartNumError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	expectedErr := fmt.Sprintf(getStatusErr, st.numParts, "")
	_, err := st.IsPartFinished(st.numParts)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("IsPartFinished did not return the expected error when the "+
			"part number is out of range.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that SentTransfer.GetProgress returns the expected progress metrics for
// various transfer states.
func TestSentTransfer_GetProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	_, st := newRandomSentTransfer(16, 24, kv, t)

	completed, sent, arrived, total, track := st.GetProgress()
	err := checkSentProgress(
		completed, sent, arrived, total, false, 0, 0, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, nil, nil, t)

	_, _ = st.SetInProgress(1, 0, 1, 2)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(completed, sent, arrived, total, false, 3, 0, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, []uint16{0, 1, 2}, nil, t)

	_, _ = st.SetInProgress(2, 3, 4, 5)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(completed, sent, arrived, total, false, 6, 0, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, []uint16{0, 1, 2, 3, 4, 5}, nil, t)

	_, _ = st.FinishTransfer(1)
	_, _ = st.UnsetInProgress(2)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(completed, sent, arrived, total, false, 0, 3, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, nil, []uint16{0, 1, 2}, t)

	_, _ = st.SetInProgress(3, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(
		completed, sent, arrived, total, false, 10, 3, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts,
		[]uint16{6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, []uint16{0, 1, 2}, t)

	_, _ = st.FinishTransfer(3)
	_, _ = st.SetInProgress(4, 3, 4, 5)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(
		completed, sent, arrived, total, false, 3, 13, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, []uint16{3, 4, 5},
		[]uint16{0, 1, 2, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, t)

	_, _ = st.FinishTransfer(4)

	completed, sent, arrived, total, track = st.GetProgress()
	err = checkSentProgress(completed, sent, arrived, total, true, 0, 16, numParts)
	if err != nil {
		t.Error(err)
	}
	checkSentTracker(track, st.numParts, nil,
		[]uint16{0, 1, 2, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 3, 4, 5}, t)
}

// Tests that 5 different callbacks all receive the expected data when
// SentTransfer.CallProgressCB is called at different stages of transfer.
func TestSentTransfer_CallProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	type progressResults struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	period := time.Millisecond

	wg := sync.WaitGroup{}
	var step0, step1, step2, step3 uint64
	numCallbacks := 5

	for i := 0; i < numCallbacks; i++ {
		progressChan := make(chan progressResults)

		cbFunc := func(completed bool, sent, arrived, total uint16,
			t interfaces.FilePartTracker, err error) {
			progressChan <- progressResults{completed, sent, arrived, total, err}
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
						if err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, 0, 0, st.numParts); err != nil {
							t.Errorf("%2d: %+v", i, err)
						}
						atomic.AddUint64(&step0, 1)
					case 1:
						if err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, 0, 0, st.numParts); err != nil {
							t.Errorf("%2d: %+v", i, err)
						}
						atomic.AddUint64(&step1, 1)
					case 2:
						if err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, 0, 6, st.numParts); err != nil {
							t.Errorf("%2d: %+v", i, err)
						}
						atomic.AddUint64(&step2, 1)
					case 3:
						if err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, true, 0, 16, st.numParts); err != nil {
							t.Errorf("%2d: %+v", i, err)
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

		st.AddProgressCB(cbFunc, period)
	}

	for !atomic.CompareAndSwapUint64(&step0, uint64(numCallbacks), 0) {
	}

	st.CallProgressCB(nil)

	for !atomic.CompareAndSwapUint64(&step1, uint64(numCallbacks), 0) {
	}

	_, _ = st.SetInProgress(0, 0, 1, 2)
	_, _ = st.SetInProgress(1, 3, 4, 5)
	_, _ = st.SetInProgress(2, 6, 7, 8)
	_, _ = st.UnsetInProgress(1)
	_, _ = st.FinishTransfer(0)
	_, _ = st.FinishTransfer(2)

	st.CallProgressCB(nil)

	for !atomic.CompareAndSwapUint64(&step2, uint64(numCallbacks), 0) {
	}

	_, _ = st.SetInProgress(4, 3, 4, 5, 9, 10, 11, 12, 13, 14, 15)
	_, _ = st.FinishTransfer(4)

	st.CallProgressCB(nil)

	wg.Wait()
}

// Tests that SentTransfer.stopScheduledProgressCB stops a scheduled callback
// from being triggered.
func TestSentTransfer_stopScheduledProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	cbChan := make(chan struct{}, 5)
	cbFunc := interfaces.SentProgressCallback(
		func(completed bool, sent, arrived, total uint16,
			t interfaces.FilePartTracker, err error) {
			cbChan <- struct{}{}
		})
	st.AddProgressCB(cbFunc, 150*time.Millisecond)
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback.")
	case <-cbChan:
	}

	st.CallProgressCB(nil)
	st.CallProgressCB(nil)
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback.")
	case <-cbChan:
	}

	err := st.stopScheduledProgressCB()
	if err != nil {
		t.Errorf("stopScheduledProgressCB returned an error: %+v", err)
	}

	select {
	case <-time.NewTimer(200 * time.Millisecond).C:
	case <-cbChan:
		t.Error("Callback called when it should have been stopped.")
	}
}

// Tests that SentTransfer.AddProgressCB adds an item to the progress callback
// list.
func TestSentTransfer_AddProgressCB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	type callbackResults struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}
	cbChan := make(chan callbackResults)
	cbFunc := interfaces.SentProgressCallback(
		func(completed bool, sent, arrived, total uint16,
			t interfaces.FilePartTracker, err error) {
			cbChan <- callbackResults{completed, sent, arrived, total, err}
		})

	done := make(chan bool)
	go func() {
		select {
		case <-time.NewTimer(time.Millisecond).C:
			t.Error("Timed out waiting for progress callback to be called.")
		case r := <-cbChan:
			err := checkSentProgress(
				r.completed, r.sent, r.arrived, r.total, false, 0, 0, 16)
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
	st.AddProgressCB(cbFunc, period)

	if len(st.progressCallbacks) != 1 {
		t.Errorf("Callback list should only have one item."+
			"\nexpected: %d\nreceived: %d", 1, len(st.progressCallbacks))
	}

	if st.progressCallbacks[0].period != period {
		t.Errorf("Callback has wrong lastCall.\nexpected: %s\nreceived: %s",
			period, st.progressCallbacks[0].period)
	}

	if st.progressCallbacks[0].lastCall != (time.Time{}) {
		t.Errorf("Callback has wrong time.\nexpected: %s\nreceived: %s",
			time.Time{}, st.progressCallbacks[0].lastCall)
	}

	if st.progressCallbacks[0].scheduled {
		t.Errorf("Callback has wrong scheduled.\nexpected: %t\nreceived: %t",
			false, st.progressCallbacks[0].scheduled)
	}
	<-done
}

// Loops through each file part encrypting it with SentTransfer.GetEncryptedPart
// and tests that it returns an encrypted part, MAC, and padding (nonce) that
// can be used to successfully decrypt and get the original part. Also tests
// that fingerprints are valid and not used more than once.
// It also tests that
func TestSentTransfer_GetEncryptedPart(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Create and fill fingerprint map used to check fingerprint validity
	// The first item in the uint16 slice is the fingerprint number and the
	// second item is the number of times it has been used
	fpMap := make(map[format.Fingerprint][]uint16, st.numFps)
	for num, fp := range ftCrypto.GenerateFingerprints(st.key, st.numFps) {
		fpMap[fp] = []uint16{uint16(num), 0}
	}

	for i := uint16(0); i < st.numFps; i++ {
		partNum := i % st.numParts

		encPart, mac, fp, err := st.GetEncryptedPart(partNum, 18)
		if err != nil {
			t.Fatalf("GetEncryptedPart returned an error for part number "+
				"%d (%d): %+v", partNum, i, err)
		}

		// Check that the fingerprint is valid
		fpNum, exists := fpMap[fp]
		if !exists {
			t.Errorf("Fingerprint %s invalid for part number %d (%d).",
				fp, partNum, i)
		}

		// Check that the fingerprint has not been used
		if fpNum[1] > 0 {
			t.Errorf("Fingerprint %s for part number %d already used by %d "+
				"other parts (%d).", fp, partNum, fpNum[1], i)
		}

		// Attempt to decrypt the part
		partMarshaled, err := ftCrypto.DecryptPart(st.key, encPart, mac, fpNum[0], fp)
		if err != nil {
			t.Errorf("Failed to decrypt file part number %d (%d): %+v",
				partNum, i, err)
		}

		partMsg, _ := UnmarshalPartMessage(partMarshaled)

		// Make sure the decrypted part matches the original
		expectedPart, _ := st.sentParts.getPart(i % st.numParts)
		if !bytes.Equal(expectedPart, partMsg.GetPart()) {
			t.Errorf("Decyrpted part number %d does not match expected (%d)."+
				"\nexpected: %+v\nreceived: %+v", partNum, i, expectedPart, partMsg.GetPart())
		}

		if partMsg.GetPartNum() != i%st.numParts {
			t.Errorf("Number of part did not match, expected: %d, "+
				"received: %d", i%st.numParts, partMsg.GetPartNum())
		}
	}
}

// Error path: tests that SentTransfer.GetEncryptedPart returns the expected
// error when no part for the given part number exists.
func TestSentTransfer_GetEncryptedPart_NoPartError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	partNum := st.numParts + 1
	expectedErr := fmt.Sprintf(noPartNumErr, partNum)

	_, _, _, err := st.GetEncryptedPart(partNum, 16)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("GetEncryptedPart did not return the expected error for a "+
			"nonexistent part number %d.\nexpected: %s\nreceived: %+v",
			partNum, expectedErr, err)
	}
}

// Error path: tests that SentTransfer.GetEncryptedPart returns the expected
// error when no fingerprints are available.
func TestSentTransfer_GetEncryptedPart_NoFingerprintsError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Use up all the fingerprints
	for i := uint16(0); i < st.numFps; i++ {
		partNum := i % st.numParts
		_, _, _, err := st.GetEncryptedPart(partNum, 18)
		if err != nil {
			t.Errorf("Error when encyrpting part number %d (%d): %+v",
				partNum, i, err)
		}
	}

	// Try to encrypt without any fingerprints
	_, _, _, err := st.GetEncryptedPart(5, 18)
	if err != MaxRetriesErr {
		t.Errorf("GetEncryptedPart did not return MaxRetriesErr when all "+
			"fingerprints have been used.\nexpected: %s\nreceived: %+v",
			MaxRetriesErr, err)
	}
}

// Tests that SentTransfer.SetInProgress correctly adds the part numbers for the
// given round ID to the in-progress map and sets the correct parts as
// in-progress in the state vector.
func TestSentTransfer_SetInProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedPartNums := []uint16{1, 2, 3}

	// Add parts to the in-progress list
	exists, err := st.SetInProgress(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("SetInProgress returned an error: %+v", err)
	}

	// Check that the round does not already exist
	if exists {
		t.Errorf("Round %d already exists.", rid)
	}

	// Check that the round ID is in the map
	partNums, exists := st.inProgressTransfers.list[rid]
	if !exists {
		t.Errorf("Part numbers for round %d not found.", rid)
	}

	// Check that the returned part numbers are correct
	if !reflect.DeepEqual(expectedPartNums, partNums) {
		t.Errorf("Received part numbers do not match expected."+
			"\nexpected: %v\nreceived: %v", expectedPartNums, partNums)
	}

	// Check that only one item was added to the list
	if len(st.inProgressTransfers.list) > 1 {
		t.Errorf("Extra items in in-progress list."+
			"\nexpected: %d\nreceived: %d", 1, len(st.inProgressTransfers.list))
	}

	// Check that the part numbers were set on the in-progress status vector
	for i, partNum := range expectedPartNums {
		if status, _ := st.partStats.Get(partNum); status != inProgress {
			t.Errorf("Part number %d not marked as in-progress in status "+
				"vector (%d).", partNum, i)
		}
	}

	// Check that the correct number of parts were marked as in-progress in the
	// status vector
	count, _ := st.partStats.GetCount(inProgress)
	if int(count) != len(expectedPartNums) {
		t.Errorf("Incorrect number of parts marked as in-progress."+
			"\nexpected: %d\nreceived: %d", len(expectedPartNums), count)
	}

	// Add more parts to the in-progress list
	expectedPartNums2 := []uint16{4, 5, 6}
	exists, err = st.SetInProgress(rid, expectedPartNums2...)
	if err != nil {
		t.Errorf("SetInProgress returned an error: %+v", err)
	}

	// Check that the round already exists
	if !exists {
		t.Errorf("Round %d should already exist.", rid)
	}

	// Check that the number of parts were marked as in-progress is unchanged
	count, _ = st.partStats.GetCount(inProgress)
	if int(count) != len(expectedPartNums2)+len(expectedPartNums) {
		t.Errorf("Incorrect number of parts marked as in-progress."+
			"\nexpected: %d\nreceived: %d",
			len(expectedPartNums2)+len(expectedPartNums), count)
	}
}

// Tests that SentTransfer.GetInProgress returns the correct part numbers for
// the given round ID in the in-progress map.
func TestSentTransfer_GetInProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedPartNums := []uint16{1, 2, 3, 4, 5, 6}

	// Add parts to the in-progress list
	_, err := st.SetInProgress(rid, expectedPartNums[:3]...)
	if err != nil {
		t.Errorf("Failed to set parts %v to in-progress: %+v",
			expectedPartNums[3:], err)
	}

	// Add parts to the in-progress list
	_, err = st.SetInProgress(rid, expectedPartNums[3:]...)
	if err != nil {
		t.Errorf("Failed to set parts %v to in-progress: %+v",
			expectedPartNums[:3], err)
	}

	// get the in-progress parts
	receivedPartNums, exists := st.GetInProgress(rid)
	if !exists {
		t.Errorf("Failed to find parts for round %d that should exist.", rid)
	}

	// Check that the returned part numbers are correct
	if !reflect.DeepEqual(expectedPartNums, receivedPartNums) {
		t.Errorf("Received part numbers do not match expected."+
			"\nexpected: %v\nreceived: %v", expectedPartNums, receivedPartNums)
	}
}

// Tests that SentTransfer.UnsetInProgress correctly removes the part numbers
// for the given round ID from the in-progress map and unsets the correct parts
// as in-progress in the state vector.
func TestSentTransfer_UnsetInProgress(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedPartNums := []uint16{1, 2, 3, 4, 5, 6}

	// Add parts to the in-progress list
	if _, err := st.SetInProgress(rid, expectedPartNums[:3]...); err != nil {
		t.Errorf("Failed to set parts in-progress: %+v", err)
	}
	if _, err := st.SetInProgress(rid, expectedPartNums[3:]...); err != nil {
		t.Errorf("Failed to set parts in-progress: %+v", err)
	}

	// Remove parts from in-progress list
	receivedPartNums, err := st.UnsetInProgress(rid)
	if err != nil {
		t.Errorf("UnsetInProgress returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedPartNums, receivedPartNums) {
		t.Errorf("Received part numbers do not match expected."+
			"\nexpected: %v\nreceived: %v", expectedPartNums, receivedPartNums)
	}

	// Check that the round ID is not the map
	partNums, exists := st.inProgressTransfers.list[rid]
	if exists {
		t.Errorf("Part numbers for round %d found: %v", rid, partNums)
	}

	// Check that the list is empty
	if len(st.inProgressTransfers.list) != 0 {
		t.Errorf("Extra items in in-progress list."+
			"\nexpected: %d\nreceived: %d", 0, len(st.inProgressTransfers.list))
	}

	// Check that there are no set parts in the in-progress status vector
	status, _ := st.partStats.Get(inProgress)
	if status != unsent {
		t.Errorf("Failed to unset all parts in the in-progress vector."+
			"\nexpected: %d\nreceived: %d", unsent, status)
	}
}

// Tests that SentTransfer.FinishTransfer removes the parts from the in-progress
// list and moved them to the finished list and that it unsets the correct parts
// in the in-progress vector in the state vector.
func TestSentTransfer_FinishTransfer(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedPartNums := []uint16{1, 2, 3}

	// Add parts to the in-progress list
	_, err := st.SetInProgress(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress list: %+v", err)
	}

	// Move transfers to the finished list
	complete, err := st.FinishTransfer(rid)
	if err != nil {
		t.Errorf("FinishTransfer returned an error: %+v", err)
	}

	// Ensure the transfer is not reported as complete
	if complete {
		t.Error("FinishTransfer reported transfer as complete.")
	}

	// Check that the round ID is not in the in-progress map
	_, exists := st.inProgressTransfers.list[rid]
	if exists {
		t.Errorf("Found parts for round %d that should not be in map.", rid)
	}

	// Check that the round ID is in the finished map
	partNums, exists := st.finishedTransfers.list[rid]
	if !exists {
		t.Errorf("Part numbers for round %d not found.", rid)
	}

	// Check that the returned part numbers are correct
	if !reflect.DeepEqual(expectedPartNums, partNums) {
		t.Errorf("Received part numbers do not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedPartNums, partNums)
	}

	// Check that only one item was added to the list
	if len(st.finishedTransfers.list) > 1 {
		t.Errorf("Extra items in finished list."+
			"\nexpected: %d\nreceived: %d", 1, len(st.finishedTransfers.list))
	}

	// Check that there are no set parts in the in-progress status vector
	count, _ := st.partStats.GetCount(inProgress)
	if count != 0 {
		t.Errorf("Failed to unset all parts in the in-progress vector."+
			"\nexpected: %d\nreceived: %d", 0, count)
	}

	// Check that the part numbers were set on the finished status vector
	for i, partNum := range expectedPartNums {
		status, _ := st.partStats.Get(inProgress)
		if status != finished {
			t.Errorf("Part number %d not marked as finished in status vector "+
				"(%d).", partNum, i)
		}
	}

	// Check that the correct number of parts were marked as finished in the
	// status vector
	count, _ = st.partStats.GetCount(finished)
	if int(count) != len(expectedPartNums) {
		t.Errorf("Incorrect number of parts marked as finished."+
			"\nexpected: %d\nreceived: %d", len(expectedPartNums), count)
	}
}

// Tests that SentTransfer.FinishTransfer returns true and sets the status to
// stopping when all file parts are marked as complete.
func TestSentTransfer_FinishTransfer_Complete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedPartNums := make([]uint16, st.numParts)
	for i := range expectedPartNums {
		expectedPartNums[i] = uint16(i)
	}

	// Add parts to the in-progress list
	_, err := st.SetInProgress(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress list: %+v", err)
	}

	// Move transfers to the finished list
	complete, err := st.FinishTransfer(rid)
	if err != nil {
		t.Errorf("FinishTransfer returned an error: %+v", err)
	}

	// Ensure the transfer is not reported as complete
	if !complete {
		t.Error("FinishTransfer reported transfer as not complete.")
	}

	// Test that the status is correctly set
	if st.status != Stopping {
		t.Errorf("Status not set to expected value when transfer is complete."+
			"\nexpected: %s\nreceived: %s", Stopping, st.status)
	}
}

// Error path: tests that SentTransfer.FinishTransfer returns the expected error
// when the round ID is found in the in-progress map.
func TestSentTransfer_FinishTransfer_NoRoundErr(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	rid := id.Round(5)
	expectedErr := fmt.Sprintf(noPartsForRoundErr, rid)

	// Move transfers to the finished list
	complete, err := st.FinishTransfer(rid)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Did not get expected error when round ID not in in-progress "+
			"map.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	// Ensure the transfer is not reported as complete
	if complete {
		t.Error("FinishTransfer reported transfer as complete.")
	}
}

// Tests that SentTransfer.GetUnsentPartNums returns only part numbers that are
// not marked as in-progress or finished.
func TestSentTransfer_GetUnsentPartNums(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(18, 27, kv, t)

	expectedPartNums := make([]uint16, 0, st.numParts/3)

	// Loop through each part and set it individually
	for i := uint16(0); i < st.numParts; i++ {
		switch i % 3 {
		case 0:
			// Part is sent (in-progress)
			_, _ = st.SetInProgress(id.Round(i), i)
		case 1:
			// Part is sent and arrived (finished)
			_, _ = st.SetInProgress(id.Round(i), i)
			_, _ = st.FinishTransfer(id.Round(i))
		case 2:
			// Part is unsent (neither in-progress nor arrived)
			expectedPartNums = append(expectedPartNums, i)
		}
	}

	unsentPartNums, err := st.GetUnsentPartNums()
	if err != nil {
		t.Errorf("GetUnsentPartNums returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedPartNums, unsentPartNums) {
		t.Errorf("Unexpected unsent part numbers.\nexpected: %d\nreceived: %d",
			expectedPartNums, unsentPartNums)
	}
}

// Tests that SentTransfer.GetSentRounds returns the expected round IDs when
// every round is either in-progress, finished, or unsent.
func TestSentTransfer_GetSentRounds(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(18, 27, kv, t)

	expectedRounds := make([]id.Round, 0, st.numParts/3)

	// Loop through each part and set it individually
	for i := uint16(0); i < st.numParts; i++ {
		rid := id.Round(i)
		switch i % 3 {
		case 0:
			// Part is sent (in-progress)
			_, _ = st.SetInProgress(rid, i)
			expectedRounds = append(expectedRounds, rid)
		case 1:
			// Part is sent and arrived (finished)
			_, _ = st.SetInProgress(rid, i)
			_, _ = st.FinishTransfer(rid)
		case 2:
			// Part is unsent (neither in-progress nor arrived)
		}
	}

	// get the sent
	sentRounds := st.GetSentRounds()
	sort.SliceStable(sentRounds,
		func(i, j int) bool { return sentRounds[i] < sentRounds[j] })

	if !reflect.DeepEqual(expectedRounds, sentRounds) {
		t.Errorf("Unexpected sent rounds.\nexpected: %d\nreceived: %d",
			expectedRounds, sentRounds)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Function Testing                                                   //
////////////////////////////////////////////////////////////////////////////////

// Tests that loadSentTransfer returns a SentTransfer that matches the original
// object in memory.
func Test_loadSentTransfer(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, expectedST := newRandomSentTransfer(16, 24, kv, t)
	_, err := expectedST.SetInProgress(5, 3, 4, 5)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress transfer: %+v", err)
	}
	_, err = expectedST.SetInProgress(10, 10, 11, 12)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress transfer: %+v", err)
	}

	_, err = expectedST.FinishTransfer(10)
	if err != nil {
		t.Errorf("Failed to move parts to finished transfer: %+v", err)
	}

	loadedST, err := loadSentTransfer(tid, kv)
	if err != nil {
		t.Errorf("loadSentTransfer returned an error: %+v", err)
	}

	// Progress callbacks cannot be compared
	loadedST.progressCallbacks = expectedST.progressCallbacks

	if !reflect.DeepEqual(expectedST, loadedST) {
		t.Errorf("Loaded SentTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedST, loadedST)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when no
// transfer with the given ID exists in storage.
func Test_loadSentTransfer_LoadInfoError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidTransferID"))

	expectedErr := strings.Split(loadSentStoreErr, "%")[0]
	_, err := loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when no "+
			"transfer with the ID %s exists in storage."+
			"\nexpected: %s\nreceived: %+v", tid, expectedErr, err)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when the
// fingerprint state vector was deleted from storage.
func Test_loadSentTransfer_LoadFingerprintStateVectorError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, st := newRandomSentTransfer(16, 24, kv, t)

	// Delete the fingerprint state vector from storage
	err := st.fpVector.Delete()
	if err != nil {
		t.Errorf("Failed to delete the fingerprint vector: %+v", err)
	}

	expectedErr := strings.Split(loadSentFpVectorErr, "%")[0]
	_, err = loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when "+
			"the fingerprint vector was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when the
// part store was deleted from storage.
func Test_loadSentTransfer_LoadPartStoreError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, st := newRandomSentTransfer(16, 24, kv, t)

	// Delete the part store from storage
	err := st.sentParts.delete()
	if err != nil {
		t.Errorf("Failed to delete the part store: %+v", err)
	}

	expectedErr := strings.Split(loadSentPartStoreErr, "%")[0]
	_, err = loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when "+
			"the part store was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when the
// in-progress transfers bundle was deleted from storage.
func Test_loadSentTransfer_LoadInProgressTransfersError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, st := newRandomSentTransfer(16, 24, kv, t)

	// Delete the in-progress transfers bundle from storage
	err := st.inProgressTransfers.delete()
	if err != nil {
		t.Errorf("Failed to delete the in-progress transfers bundle: %+v", err)
	}

	expectedErr := strings.Split(loadInProgressTransfersErr, "%")[0]
	_, err = loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when "+
			"the in-progress transfers bundle was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when the
// finished transfer bundle was deleted from storage.
func Test_loadSentTransfer_LoadFinishedTransfersError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, st := newRandomSentTransfer(16, 24, kv, t)

	// Delete the finished transfers bundle from storage
	err := st.finishedTransfers.delete()
	if err != nil {
		t.Errorf("Failed to delete the finished transfers bundle: %+v", err)
	}

	expectedErr := strings.Split(loadFinishedTransfersErr, "%")[0]
	_, err = loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when "+
			"the finished transfers bundle was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that loadSentTransfer returns the expected error when the
// part statuses multi state vector was deleted from storage.
func Test_loadSentTransfer_LoadPartStatsMultiStateVectorError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	tid, st := newRandomSentTransfer(16, 24, kv, t)

	// Delete the in-progress state vector from storage
	err := st.partStats.Delete()
	if err != nil {
		t.Errorf("Failed to delete the partStats vector: %+v", err)
	}

	expectedErr := strings.Split(loadSentPartStatusVectorErr, "%")[0]
	_, err = loadSentTransfer(tid, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadSentTransfer did not return the expected error when "+
			"the partStats vector was deleted from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that SentTransfer.saveInfo saves the expected data to storage.
func TestSentTransfer_saveInfo(t *testing.T) {
	st := &SentTransfer{
		key:      ftCrypto.UnmarshalTransferKey([]byte("key")),
		numParts: 16,
		kv:       versioned.NewKV(make(ekv.Memstore)),
	}

	err := st.saveInfo()
	if err != nil {
		t.Errorf("saveInfo returned an error: %+v", err)
	}

	vo, err := st.kv.Get(sentTransferKey, sentTransferVersion)
	if err != nil {
		t.Errorf("Failed to load SentTransfer from storage: %+v", err)
	}

	if !bytes.Equal(st.marshal(), vo.Data) {
		t.Errorf("Marshalled data loaded from storage does not match expected."+
			"\nexpected: %+v\nreceived: %+v", st.marshal(), vo.Data)
	}
}

// Tests that SentTransfer.loadInfo loads a saved SentTransfer from storage.
func TestSentTransfer_loadInfo(t *testing.T) {
	st := &SentTransfer{
		recipient: id.NewIdFromString("recipient", id.User, t),
		key:       ftCrypto.UnmarshalTransferKey([]byte("key")),
		numParts:  16,
		kv:        versioned.NewKV(make(ekv.Memstore)),
	}

	err := st.saveInfo()
	if err != nil {
		t.Errorf("failed to save new SentTransfer to storage: %+v", err)
	}

	loadedST := &SentTransfer{kv: st.kv}
	err = loadedST.loadInfo()
	if err != nil {
		t.Errorf("load returned an error: %+v", err)
	}

	if !reflect.DeepEqual(st, loadedST) {
		t.Errorf("Loaded SentTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", st, loadedST)
	}
}

// Error path: tests that SentTransfer.loadInfo returns an error when there is
// no object in storage to load
func TestSentTransfer_loadInfo_Error(t *testing.T) {
	loadedST := &SentTransfer{kv: versioned.NewKV(make(ekv.Memstore))}
	err := loadedST.loadInfo()
	if err == nil {
		t.Errorf("Loaded object that should not be in storage: %+v", err)
	}
}

// Tests that SentTransfer.delete removes all data from storage.
func TestSentTransfer_delete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Add in-progress transfers
	_, err := st.SetInProgress(5, 3, 4, 5)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress transfer: %+v", err)
	}
	_, err = st.SetInProgress(10, 10, 11, 12)
	if err != nil {
		t.Errorf("Failed to add parts to in-progress transfer: %+v", err)
	}

	// Mark last in-progress transfer and finished
	_, err = st.FinishTransfer(10)
	if err != nil {
		t.Errorf("Failed to move parts to finished transfer: %+v", err)
	}

	// Delete everything from storage
	err = st.delete()
	if err != nil {
		t.Errorf("delete returned an error: %+v", err)
	}

	// Check that the SentTransfer info was deleted
	err = st.loadInfo()
	if err == nil {
		t.Error("Successfully loaded SentTransfer info from storage when it " +
			"should have been deleted.")
	}

	// Check that the parts store were deleted
	_, err = loadPartStore(st.kv)
	if err == nil {
		t.Error("Successfully loaded file parts from storage when it should " +
			"have been deleted.")
	}

	// Check that the in-progress transfers were deleted
	_, err = loadTransferredBundle(inProgressKey, st.kv)
	if err == nil {
		t.Error("Successfully loaded in-progress transfers from storage when " +
			"it should have been deleted.")
	}

	// Check that the finished transfers were deleted
	_, err = loadTransferredBundle(finishedKey, st.kv)
	if err == nil {
		t.Error("Successfully loaded finished transfers from storage when " +
			"it should have been deleted.")
	}

	// Check that the fingerprint vector was deleted
	_, err = utility.LoadStateVector(st.kv, sentFpVectorKey)
	if err == nil {
		t.Error("Successfully loaded fingerprint vector from storage when it " +
			"should have been deleted.")
	}

	// Check that the in-progress status vector was deleted
	_, err = utility.LoadStateVector(st.kv, sentInProgressVectorKey)
	if err == nil {
		t.Error("Successfully loaded in-progress vector from storage when it " +
			"should have been deleted.")
	}

	// Check that the finished status vector was deleted
	_, err = utility.LoadStateVector(st.kv, sentFinishedVectorKey)
	if err == nil {
		t.Error("Successfully loaded finished vector from storage when it " +
			"should have been deleted.")
	}

}

// Tests that SentTransfer.deleteInfo removes the saved SentTransfer data from
// storage.
func TestSentTransfer_deleteInfo(t *testing.T) {
	st := &SentTransfer{
		key:      ftCrypto.UnmarshalTransferKey([]byte("key")),
		numParts: 16,
		kv:       versioned.NewKV(make(ekv.Memstore)),
	}

	// Save from storage
	err := st.saveInfo()
	if err != nil {
		t.Errorf("failed to save new SentTransfer to storage: %+v", err)
	}

	// Delete from storage
	err = st.deleteInfo()
	if err != nil {
		t.Errorf("deleteInfo returned an error: %+v", err)
	}

	// Make sure deleted object cannot be loaded from storage
	_, err = st.kv.Get(sentTransferKey, sentTransferVersion)
	if err == nil {
		t.Error("Loaded object that should be deleted from storage.")
	}
}

// Tests that a SentTransfer marshalled with SentTransfer.marshal and then
// unmarshalled with unmarshalSentTransfer matches the original.
func TestSentTransfer_marshal_unmarshalSentTransfer(t *testing.T) {
	st := &SentTransfer{
		recipient: id.NewIdFromString("testRecipient", id.User, t),
		key:       ftCrypto.UnmarshalTransferKey([]byte("key")),
		numParts:  16,
		numFps:    20,
		status:    Stopped,
	}

	marshaledData := st.marshal()

	recipient, key, numParts, numFps, status :=
		unmarshalSentTransfer(marshaledData)

	if !st.recipient.Cmp(recipient) {
		t.Errorf("Failed to get recipient ID.\nexpected: %s\nreceived: %s",
			st.recipient, recipient)
	}

	if st.key != key {
		t.Errorf("Failed to get expected key.\nexpected: %s\nreceived: %s",
			st.key, key)
	}

	if st.numParts != numParts {
		t.Errorf("Failed to get expected number of parts."+
			"\nexpected: %d\nreceived: %d", st.numParts, numParts)
	}

	if st.numFps != numFps {
		t.Errorf("Failed to get expected number of fingerprints."+
			"\nexpected: %d\nreceived: %d", st.numFps, numFps)
	}

	if st.status != status {
		t.Errorf("Failed to get expected transfer status."+
			"\nexpected: %s\nreceived: %s", st.status, status)
	}
}

// Consistency test: tests that makeSentTransferPrefix returns the expected
// prefixes for the provided transfer IDs.
func Test_makeSentTransferPrefix_Consistency(t *testing.T) {
	prng := NewPrng(42)
	expectedPrefixes := []string{
		"FileTransferSentTransferStoreU4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=",
		"FileTransferSentTransferStore39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=",
		"FileTransferSentTransferStoreCD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=",
		"FileTransferSentTransferStoreuoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=",
		"FileTransferSentTransferStoreGwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=",
		"FileTransferSentTransferStorernvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=",
		"FileTransferSentTransferStoreceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=",
		"FileTransferSentTransferStoreSYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=",
		"FileTransferSentTransferStoreNhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=",
		"FileTransferSentTransferStorekM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog=",
	}

	for i, expected := range expectedPrefixes {
		tid, _ := ftCrypto.NewTransferID(prng)
		prefix := makeSentTransferPrefix(tid)

		if expected != prefix {
			t.Errorf("New SentTransfer prefix does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, prefix)
		}
	}
}

// Tests that each of the elements in the uint32 slice returned by
// uint16SliceToUint32Slice matches the elements in the original uint16 slice.
func Test_uint16SliceToUint32Slice(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	uint16Slice := make([]uint16, 100)

	for i := range uint16Slice {
		uint16Slice[i] = uint16(prng.Uint32())
	}

	uint32Slice := uint16SliceToUint32Slice(uint16Slice)

	// Check that each element is correct
	for i, expected := range uint16Slice {
		if uint32(expected) != uint32Slice[i] {
			t.Errorf("Element #%d is incorrect.\nexpected: %d\nreceived: %d",
				i, uint32(expected), uint32Slice[i])
		}
	}
}

// newRandomSentTransfer generates a new SentTransfer with random data.
func newRandomSentTransfer(numParts, numFps uint16, kv *versioned.KV,
	t *testing.T) (ftCrypto.TransferID, *SentTransfer) {
	// Generate new PRNG with the seed generated by multiplying the pointer for
	// numParts with the current UNIX time in nanoseconds
	seed, _ := strconv.ParseInt(fmt.Sprintf("%d", &numParts), 10, 64)
	seed *= netTime.Now().UnixNano()
	prng := NewPrng(seed)

	recipient, _ := id.NewRandomID(prng, id.User)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	parts := make([][]byte, numParts)
	for i := uint16(0); i < numParts; i++ {
		parts[i] = make([]byte, 16)
		_, err := prng.Read(parts[i])
		if err != nil {
			t.Errorf("Failed to generate random part: %+v", err)
		}
	}

	st, err := NewSentTransfer(recipient, tid, key, parts, numFps, nil, 0, kv)
	if err != nil {
		t.Errorf("Failed to create new SentTansfer: %+v", err)
	}

	return tid, st
}

// checkSentProgress compares the output of SentTransfer.GetProgress to expected
// values.
func checkSentProgress(completed bool, sent, arrived, total uint16,
	eCompleted bool, eSent, eArrived, eTotal uint16) error {
	if eCompleted != completed || eSent != sent || eArrived != arrived ||
		eTotal != total {
		return errors.Errorf("Returned progress does not match expected."+
			"\n          completed  sent  arrived  total"+
			"\nexpected:     %5t   %3d      %3d    %3d"+
			"\nreceived:     %5t   %3d      %3d    %3d",
			eCompleted, eSent, eArrived, eTotal,
			completed, sent, arrived, total)
	}

	return nil
}

// checkSentTracker checks that the sentPartTracker is reporting the correct
// values for each part. Also checks that sentPartTracker.GetNumParts returns
// the expected value (make sure numParts comes from a correct source).
func checkSentTracker(track interfaces.FilePartTracker, numParts uint16,
	inProgress, finished []uint16, t *testing.T) {
	if track.GetNumParts() != numParts {
		t.Errorf("Tracker reported incorrect number of parts."+
			"\nexpected: %d\nreceived: %d", numParts, track.GetNumParts())
		return
	}

	for partNum := uint16(0); partNum < numParts; partNum++ {
		var done bool
		for _, inProgressNum := range inProgress {
			if inProgressNum == partNum {
				status := track.GetPartStatus(partNum)
				if status != interfaces.FpSent {
					t.Errorf("Part number %d has unexpected status."+
						"\nexpected: %d\nreceived: %d",
						partNum, interfaces.FpSent, status)
				}
				done = true
				break
			}
		}
		if done {
			continue
		}

		for _, finishedNum := range finished {
			if finishedNum == partNum {
				status := track.GetPartStatus(partNum)
				if status != interfaces.FpArrived {
					t.Errorf("Part number %d has unexpected status."+
						"\nexpected: %d\nreceived: %d",
						partNum, interfaces.FpArrived, status)
				}
				done = true
				break
			}
		}
		if done {
			continue
		}

		status := track.GetPartStatus(partNum)
		if status != interfaces.FpUnsent {
			t.Errorf("Part number %d has incorrect status."+
				"\nexpected: %d\nreceived: %d",
				partNum, interfaces.FpUnsent, status)
		}
	}
}
