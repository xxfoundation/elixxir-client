////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that Manager.oldTransferRecovery adds all unsent parts to the queue.
func TestManager_oldTransferRecovery(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{6, 12, 18}, false, true, nil, nil, kv, t)

	finishedRounds := make(map[id.Round][]ftCrypto.TransferID)
	expectedStatus := make(
		map[ftCrypto.TransferID]map[uint16]interfaces.FpStatus, len(sti))
	numCbCalls := make(map[ftCrypto.TransferID]int, len(sti))
	var numUnsent int

	for i, st := range sti {
		transfer, err := m.sent.GetTransfer(st.tid)
		if err != nil {
			t.Fatalf("Failed to get transfer #%d %s: %+v", i, st.tid, err)
		}

		expectedStatus[st.tid] = make(map[uint16]interfaces.FpStatus, st.numParts)

		// Loop through each part and set it individually
		for j, k := uint16(0), 0; j < transfer.GetNumParts(); j++ {
			rid := id.Round(j)
			switch j % 3 {
			case 0:
				// Part is sent (in-progress)
				_, _ = transfer.SetInProgress(rid, j)
				if k%2 == 0 {
					finishedRounds[rid] = append(finishedRounds[rid], st.tid)
					expectedStatus[st.tid][j] = 2
				} else {
					expectedStatus[st.tid][j] = 0
					numUnsent++
				}
				numCbCalls[st.tid]++
				k++
			case 1:
				// Part is sent and arrived (finished)
				_, _ = transfer.SetInProgress(rid, j)
				_, _ = transfer.FinishTransfer(rid)
				finishedRounds[rid] = append(finishedRounds[rid], st.tid)
				expectedStatus[st.tid][j] = 2
			case 2:
				// Part is unsent (neither in-progress nor arrived)
				expectedStatus[st.tid][j] = 0
				numUnsent++
			}
		}
	}

	// Returns an error on function and round failure on callback if sendErr is
	// set; otherwise, it reports round successes and returns nil
	rr := func(rIDs []id.Round, _ time.Duration, cb cmix.RoundEventCallback) error {
		rounds := make(map[id.Round]cmix.RoundLookupStatus, len(rIDs))
		for _, rid := range rIDs {
			if finishedRounds[rid] != nil {
				rounds[rid] = cmix.Succeeded
			} else {
				rounds[rid] = cmix.Failed
			}
		}
		cb(true, false, rounds)

		return nil
	}

	// Load new manager from the original manager's storage
	net := newTestNetworkManager(false, nil, nil, t)
	loadedManager, err := newManager(
		nil, nil, nil, net, nil, rr, kv, nil, DefaultParams())
	if err != nil {
		t.Errorf("Failed to create new manager from KV: %+v", err)
	}

	// Create new progress callbacks with channels
	cbChans := make([]chan sentProgressResults, len(sti))
	numCbCalls2 := make(map[ftCrypto.TransferID]int, len(sti))
	for i, st := range sti {
		// Create sent progress callback and channel
		cbChan := make(chan sentProgressResults, 32)
		numCbCalls2[st.tid] = 0
		tid := st.tid
		numCalls, maxNumCalls := int64(0), int64(numCbCalls[tid])
		cb := func(completed bool, sent, arrived, total uint16,
			tr interfaces.FilePartTracker, err error) {
			if atomic.CompareAndSwapInt64(&numCalls, maxNumCalls, maxNumCalls) {
				cbChan <- sentProgressResults{
					completed, sent, arrived, total, tr, err}
			}
			atomic.AddInt64(&numCalls, 1)
		}
		cbChans[i] = cbChan

		err = loadedManager.RegisterSentProgressCallback(st.tid, cb, 0)
		if err != nil {
			t.Errorf("Failed to register SentProgressCallback for transfer "+
				"%d %s: %+v", i, st.tid, err)
		}
	}

	// Wait until callbacks have been called to know the transfers have been
	// recovered
	var wg sync.WaitGroup
	for i, st := range sti {
		wg.Add(1)
		go func(i, callNum int, cbChan chan sentProgressResults, st sentTransferInfo) {
			defer wg.Done()
			select {
			case <-time.NewTimer(150 * time.Millisecond).C:
			case <-cbChan:
			}
		}(i, numCbCalls[st.tid], cbChans[i], st)
	}

	// Create health chan
	healthyRecover := make(chan bool, networkHealthBuffLen)
	chanID := net.GetHealthTracker().AddChannel(healthyRecover)
	healthyRecover <- true

	loadedManager.oldTransferRecovery(healthyRecover, chanID)

	wg.Wait()

	// Check the status of each round in each transfer
	for i, st := range sti {
		transfer, err := loadedManager.sent.GetTransfer(st.tid)
		if err != nil {
			t.Fatalf("Failed to get transfer #%d %s: %+v", i, st.tid, err)
		}

		_, _, _, _, track := transfer.GetProgress()
		for j := uint16(0); j < track.GetNumParts(); j++ {
			if track.GetPartStatus(j) != expectedStatus[st.tid][j] {
				t.Errorf("Unexpected part #%d status for transfer #%d %s."+
					"\nexpected: %d\nreceived: %d", j, i, st.tid,
					expectedStatus[st.tid][j], track.GetPartStatus(j))
			}
		}
	}

	// Check that each item in the queue is unsent
	var queueCount int
	for done := false; !done; {
		select {
		case <-time.NewTimer(5 * time.Millisecond).C:
			done = true
		case p := <-loadedManager.sendQueue:
			queueCount++
			if expectedStatus[p.tid][p.partNum] != 0 {
				t.Errorf("Part #%d for transfer %s not expected in qeueu."+
					"\nexpected: %d\nreceived: %d", p.partNum, p.tid,
					expectedStatus[p.tid][p.partNum], 0)
			}
		}
	}

	// Check that the number of items in the queue is correct
	if queueCount != numUnsent {
		t.Errorf("Number of items incorrect.\nexpected: %d\nreceived: %d",
			numUnsent, queueCount)
	}
}

// Tests that Manager.updateSentRounds updates the status of each round
// correctly by using the part tracker and checks that all the correct parts
// were added to the queue.
func TestManager_updateSentRounds(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{6, 12, 18}, false, true, nil, nil, kv, t)

	finishedRounds := make(map[id.Round][]ftCrypto.TransferID)
	expectedStatus := make(
		map[ftCrypto.TransferID]map[uint16]interfaces.FpStatus, len(sti))
	var numUnsent int

	for i, st := range sti {
		transfer, err := m.sent.GetTransfer(st.tid)
		if err != nil {
			t.Fatalf("Failed to get transfer #%d %s: %+v", i, st.tid, err)
		}

		expectedStatus[st.tid] = make(
			map[uint16]interfaces.FpStatus, st.numParts)

		// Loop through each part and set it individually
		for j, k := uint16(0), 0; j < transfer.GetNumParts(); j++ {
			rid := id.Round(j)
			switch j % 3 {
			case 0:
				// Part is sent (in-progress)
				_, _ = transfer.SetInProgress(rid, j)
				if k%2 == 0 {
					finishedRounds[rid] = append(finishedRounds[rid], st.tid)
					expectedStatus[st.tid][j] = 2
				} else {
					expectedStatus[st.tid][j] = 0
					numUnsent++
				}
				k++
			case 1:
				// Part is sent and arrived (finished)
				_, _ = transfer.SetInProgress(rid, j)
				_, _ = transfer.FinishTransfer(rid)
				finishedRounds[rid] = append(finishedRounds[rid], st.tid)
				expectedStatus[st.tid][j] = 2
			case 2:
				// Part is unsent (neither in-progress nor arrived)
				expectedStatus[st.tid][j] = 0
			}
		}
	}

	// Returns an error on function and round failure on callback if sendErr is
	// set; otherwise, it reports round successes and returns nil
	rr := func(rIDs []id.Round, _ time.Duration, cb cmix.RoundEventCallback) error {
		rounds := make(map[id.Round]cmix.RoundLookupStatus, len(rIDs))
		for _, rid := range rIDs {
			if finishedRounds[rid] != nil {
				rounds[rid] = cmix.Succeeded
			} else {
				rounds[rid] = cmix.Failed
			}
		}
		cb(true, false, rounds)

		return nil
	}

	loadedManager, err := newManager(
		nil, nil, nil, nil, nil, rr, kv, nil, DefaultParams())
	if err != nil {
		t.Errorf("Failed to create new manager from KV: %+v", err)
	}

	// Create health chan
	healthyRecover := make(chan bool, networkHealthBuffLen)
	healthyRecover <- true

	// Get list of rounds that parts were sent on
	_, loadedSentRounds, _ := m.sent.GetUnsentPartsAndSentRounds()

	err = loadedManager.updateSentRounds(healthyRecover, loadedSentRounds)
	if err != nil {
		t.Errorf("updateSentRounds returned an error: %+v", err)
	}

	// Check the status of each round in each transfer
	for i, st := range sti {
		transfer, err := loadedManager.sent.GetTransfer(st.tid)
		if err != nil {
			t.Fatalf("Failed to get transfer #%d %s: %+v", i, st.tid, err)
		}

		_, _, _, _, track := transfer.GetProgress()
		for j := uint16(0); j < track.GetNumParts(); j++ {
			if track.GetPartStatus(j) != expectedStatus[st.tid][j] {
				t.Errorf("Unexpected part #%d status for transfer #%d %s."+
					"\nexpected: %d\nreceived: %d", j, i, st.tid,
					expectedStatus[st.tid][j], track.GetPartStatus(j))
			}
		}
	}

	// Check that each item in the queue is unsent
	var queueCount int
	for done := false; !done; {
		select {
		case <-time.NewTimer(5 * time.Millisecond).C:
			done = true
		case p := <-loadedManager.sendQueue:
			queueCount++
			if expectedStatus[p.tid][p.partNum] != 0 {
				t.Errorf("Part #%d for transfer %s not expected in qeueu."+
					"\nexpected: %d\nreceived: %d", p.partNum, p.tid,
					expectedStatus[p.tid][p.partNum], 0)
			}
		}
	}

	// Check that the number of items in the queue is correct
	if queueCount != numUnsent {
		t.Errorf("Number of items incorrect.\nexpected: %d\nreceived: %d",
			numUnsent, queueCount)
	}
}

// Error path: tests that Manager.updateSentRounds returns the expected error
// when getRoundResults returns only errors.
func TestManager_updateSentRounds_Error(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	m, _, _ := newTestManagerWithTransfers(
		[]uint16{6, 12, 18}, false, true, nil, nil, kv, t)

	// Returns an error on function and round failure on callback if sendErr is
	// set; otherwise, it reports round successes and returns nil
	m.getRoundResults = func(
		[]id.Round, time.Duration, cmix.RoundEventCallback) error {
		return errors.Errorf("GetRoundResults error")
	}

	// Create health chan
	healthyRecover := make(chan bool, roundResultsMaxAttempts)
	for i := 0; i < roundResultsMaxAttempts; i++ {
		healthyRecover <- true
	}

	sentRounds := map[id.Round][]ftCrypto.TransferID{
		0: {{1}, {2}, {3}},
		5: {{4}, {2}, {6}},
		9: {{3}, {9}, {8}},
	}

	expectedErr := fmt.Sprintf(
		oldTransfersRoundResultsErr, len(sentRounds), roundResultsMaxAttempts)
	err := m.updateSentRounds(healthyRecover, sentRounds)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("updateSentRounds did not return the expected error when "+
			"getRoundResults returns only errors.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}

}

// Tests that roundIdMapToList returns all the round IDs in the map.
func Test_roundIdMapToList(t *testing.T) {
	n := 10
	roundMap := make(map[id.Round][]ftCrypto.TransferID, n)
	expectedRoundList := make([]id.Round, n)

	csPrng := NewPrng(42)
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < n; i++ {
		rid := id.Round(i)
		roundMap[rid] = make([]ftCrypto.TransferID, prng.Intn(10))
		for j := range roundMap[rid] {
			roundMap[rid][j], _ = ftCrypto.NewTransferID(csPrng)
		}

		expectedRoundList[i] = rid
	}

	receivedRoundList := roundIdMapToList(roundMap)

	sort.SliceStable(receivedRoundList, func(i, j int) bool {
		return receivedRoundList[i] < receivedRoundList[j]
	})

	if !reflect.DeepEqual(expectedRoundList, receivedRoundList) {
		t.Errorf("Round list does not match expected."+
			"\nexpected: %v\nreceived: %v",
			expectedRoundList, receivedRoundList)
	}
}
