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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests that Manager.sendThread successfully sends the parts and reports their
// progress on the callback.
func TestManager_sendThread(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers([]uint16{12, 4, 1}, false, nil, t)

	// Add three transfers
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
		{1, 2, 3, 0},
		{0},
	}

	var wg sync.WaitGroup
	for i, st := range sti {
		wg.Add(1)
		go func(i int, st sentTransferInfo) {
			for j := 0; j < 2; j++ {
				select {
				case <-time.NewTimer(20 * time.Millisecond).C:
					t.Errorf("Timed out waiting for callback #%d", i)
				case r := <-st.cbChan:
					if j > 0 {
						err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, uint16(len(partsToSend[i])), 0,
							st.numParts)
						if err != nil {
							t.Errorf("%d: %+v", i, err)
						}
						if r.err != nil {
							t.Errorf("Callback returned an error (%d): %+v", i, r.err)
						}
					}
				}
			}
			wg.Done()
		}(i, st)
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
	}
	rand.Shuffle(len(queuedParts), func(i, j int) {
		queuedParts[i], queuedParts[j] = queuedParts[j], queuedParts[i]
	})

	// Crate custom getRngNum function that always returns 11
	getNumParts := func(rng csprng.Source) int {
		return len(queuedParts)
	}

	// Start sending thread
	stop := stoppable.NewSingle("testSendThreadStoppable")
	go m.sendThread(stop, getNumParts)

	// Add parts to queue
	for _, part := range queuedParts {
		m.sendQueue <- part
	}

	wg.Wait()

	err := stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}

	if len(m.net.(*testNetworkManager).GetMsgList(0)) != len(queuedParts) {
		t.Errorf("Not all messages were received.\nexpected: %d\nreceived: %d",
			len(queuedParts), len(m.net.(*testNetworkManager).GetMsgList(0)))
	}
}

// Tests that Manager.sendThread successfully sends a partially filled batch
// of the correct length when its times out waiting for messages.
func TestManager_sendThread_Timeout(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers([]uint16{12, 4, 1}, false, nil, t)

	// Add three transfers
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
		{1, 2, 3, 0},
		{0},
	}

	var wg sync.WaitGroup
	for i, st := range sti[:1] {
		wg.Add(1)
		go func(i int, st sentTransferInfo) {
			for j := 0; j < 2; j++ {
				select {
				case <-time.NewTimer(20*time.Millisecond + pollSleepDuration).C:
					t.Errorf("Timed out waiting for callback #%d", i)
				case r := <-st.cbChan:
					if j > 0 {
						err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, 5, 0,
							st.numParts)
						if err != nil {
							t.Errorf("%d: %+v", i, err)
						}
						if r.err != nil {
							t.Errorf("Callback returned an error (%d): %+v", i, r.err)
						}
					}
				}
			}
			wg.Done()
		}(i, st)
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
	}

	// Crate custom getRngNum function that always returns 11
	getNumParts := func(rng csprng.Source) int {
		return len(queuedParts)
	}

	// Start sending thread
	stop := stoppable.NewSingle("testSendThreadStoppable")
	go m.sendThread(stop, getNumParts)

	// Add parts to queue
	for _, part := range queuedParts[:5] {
		m.sendQueue <- part
	}

	time.Sleep(pollSleepDuration)

	wg.Wait()

	err := stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}

	if len(m.net.(*testNetworkManager).GetMsgList(0)) != len(queuedParts[:5]) {
		t.Errorf("Not all messages were received.\nexpected: %d\nreceived: %d",
			len(queuedParts[:5]), len(m.net.(*testNetworkManager).GetMsgList(0)))
	}
}

// Tests that Manager.sendParts sends all the correct cMix messages and calls
// the progress callbacks with the correct values.
func TestManager_sendParts(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers([]uint16{12, 4, 1}, false, nil, t)
	prng := NewPrng(42)

	// Add three transfers
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
		{1, 2, 3, 0},
		{0},
	}

	var wg sync.WaitGroup
	for i, st := range sti {
		wg.Add(1)
		go func(i int, st sentTransferInfo) {
			for j := 0; j < 2; j++ {
				select {
				case <-time.NewTimer(20 * time.Millisecond).C:
					t.Errorf("Timed out waiting for callback #%d", i)
				case r := <-st.cbChan:
					if j > 0 {
						err := checkSentProgress(r.completed, r.sent, r.arrived,
							r.total, false, uint16(len(partsToSend[i])), 0,
							st.numParts)
						if err != nil {
							t.Errorf("%d: %+v", i, err)
						}
						if r.err != nil {
							t.Errorf("Callback returned an error (%d): %+v", i, r.err)
						}
					}
				}
			}
			wg.Done()
		}(i, st)
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
	}
	rand.Shuffle(len(queuedParts), func(i, j int) {
		queuedParts[i], queuedParts[j] = queuedParts[j], queuedParts[i]
	})

	err := m.sendParts(queuedParts, prng)
	if err != nil {
		t.Errorf("sendParts returned an error: %+v", err)
	}

	m.net.(*testNetworkManager).GetMsgList(0)

	// Check that each recipient is connected to the correct transfer and that
	// the fingerprint is correct
	for i, tcm := range m.net.(*testNetworkManager).GetMsgList(0) {
		index := 0
		for ; !sti[index].recipient.Cmp(tcm.Recipient); index++ {

		}
		transfer, err := m.sent.GetTransfer(sti[index].tid)
		if err != nil {
			t.Errorf("Failed to get transfer %s: %+v", sti[index].tid, err)
		}

		var fpFound bool
		for _, fp := range ftCrypto.GenerateFingerprints(
			transfer.GetTransferKey(), 15) {
			if fp == tcm.Message.GetKeyFP() {
				fpFound = true
			}
		}
		if !fpFound {
			t.Errorf("Fingeprint %s not found (%d).", tcm.Message.GetKeyFP(), i)
		}
	}

	wg.Wait()
}

// Error path: tests that, on SendManyCMIX failure, Manager.sendParts adds the
// parts back into the queue, does not call the callback, and does not update
// the progress.
func TestManager_sendParts_SendManyCmixError(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers([]uint16{12, 4, 1}, true, nil, t)
	prng := NewPrng(42)
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
		{1, 2, 3, 0},
		{0},
	}

	var wg sync.WaitGroup
	for i, st := range sti {
		wg.Add(1)
		go func(i int, st sentTransferInfo) {
			for j := 0; j < 2; j++ {
				select {
				case <-time.NewTimer(10 * time.Millisecond).C:
					if j < 1 {
						t.Errorf("Timed out waiting for callback #%d (%d)", i, j)
					}
				case r := <-st.cbChan:
					if j > 0 {
						t.Errorf("Callback called on send failure: %+v", r)
					}
				}
			}
			wg.Done()
		}(i, st)
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
	}

	err := m.sendParts(queuedParts, prng)
	if err != nil {
		t.Errorf("sendParts returned an error: %+v", err)
	}

	if len(m.net.(*testNetworkManager).GetMsgList(0)) > 0 {
		t.Errorf("Sent %d cMix message(s) when sending should have failed.",
			len(m.net.(*testNetworkManager).GetMsgList(0)))
	}

	if len(m.sendQueue) != len(queuedParts) {
		t.Errorf("Failed to add all parts to queue after send failure."+
			"\nexpected: %d\nreceived: %d", len(queuedParts), len(m.sendQueue))
	}

	wg.Wait()
}

// Tests that Manager.buildMessages returns the expected values for a group
// of 11 file parts from three different transfers.
func TestManager_buildMessages(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers([]uint16{12, 4, 1}, false, nil, t)
	prng := NewPrng(42)
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
		{1, 2, 3, 0},
		{0},
	}
	idMap := map[ftCrypto.TransferID]int{
		sti[0].tid: 0,
		sti[1].tid: 1,
		sti[2].tid: 2,
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
	}
	rand.Shuffle(len(queuedParts), func(i, j int) {
		queuedParts[i], queuedParts[j] = queuedParts[j], queuedParts[i]
	})

	// Build the messages
	prng = NewPrng(2)
	messages, transfers, groupedParts, partsToResend, err := m.buildMessages(
		queuedParts, prng)
	if err != nil {
		t.Errorf("buildMessages returned an error: %+v", err)
	}

	// Check that the transfer map has all the transfers
	for i, st := range sti {
		_, exists := transfers[st.tid]
		if !exists {
			t.Errorf("Transfer %s (#%d) not in transfers map.", st.tid, i)
		}
	}

	// Check that partsToResend contains all parts because none should have
	// errored
	if len(partsToResend) != len(queuedParts) {
		sort.SliceStable(partsToResend, func(i, j int) bool {
			return partsToResend[i] < partsToResend[j]
		})
		for i, j := range partsToResend {
			if i != j {
				t.Errorf("Part at index %d not found. Found %d.", i, j)
			}
		}
	}

	// Check that all the parts are present and grouped correctly
	for tid, partNums := range groupedParts {
		sort.SliceStable(partNums, func(i, j int) bool {
			return partNums[i] < partNums[j]
		})
		sort.SliceStable(partsToSend[idMap[tid]], func(i, j int) bool {
			return partsToSend[idMap[tid]][i] < partsToSend[idMap[tid]][j]
		})
		if !reflect.DeepEqual(partsToSend[idMap[tid]], partNums) {
			t.Errorf("Incorrect parts for transfer %s."+
				"\nexpected: %v\nreceived: %v",
				tid, partsToSend[idMap[tid]], partNums)
		}
	}

	// Check that each recipient is connected to the correct transfer and that
	// the fingerprint is correct
	for i, tcm := range messages {
		index := 0
		for ; !sti[index].recipient.Cmp(tcm.Recipient); index++ {

		}
		transfer, err := m.sent.GetTransfer(sti[index].tid)
		if err != nil {
			t.Errorf("Failed to get transfer %s: %+v", sti[index].tid, err)
		}

		var fpFound bool
		for _, fp := range ftCrypto.GenerateFingerprints(
			transfer.GetTransferKey(), 15) {
			if fp == tcm.Message.GetKeyFP() {
				fpFound = true
			}
		}
		if !fpFound {
			t.Errorf("Fingeprint %s not found (%d).", tcm.Message.GetKeyFP(), i)
		}
	}
}

// Tests that Manager.buildMessages skips file parts with deleted transfers or
// transfers that have run out of fingerprints.
func TestManager_buildMessages_MessageBuildFailureError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	// Add transfer

	type callbackResults struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	callbackChan := make(chan callbackResults, 10)
	progressCB := func(completed bool, sent, arrived, total uint16,
		t interfaces.FilePartTracker, err error) {
		callbackChan <- callbackResults{completed, sent, arrived, total, err}
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(20 * time.Millisecond).C:
				t.Error("Timed out waiting for callback.")
			case r := <-callbackChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					expectedErr := fmt.Sprintf(
						maxRetriesErr, ftStorage.MaxRetriesErr)
					if i > 0 && (r.err == nil || r.err.Error() != expectedErr) {
						t.Errorf("Callback received unexpected error when max "+
							"retries should have been reached."+
							"\nexpected: %s\nreceived: %v", expectedErr, r.err)
					}
					done1 <- true
				}
			}
		}
	}()

	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, parts := newFile(4, 64, prng, t)
	tid, err := m.sent.AddTransfer(
		recipient, key, parts, 3, progressCB, time.Millisecond, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	<-done0

	// Create queued part list add parts
	queuedParts := []queuedPart{
		{tid, 0},
		{tid, 1},
		{tid, 2},
		{tid, 3},
		{ftCrypto.UnmarshalTransferID([]byte("invalidID")), 3},
	}

	// Build the messages
	prng = NewPrng(46)
	messages, transfers, groupedParts, partsToResend, err := m.buildMessages(
		queuedParts, prng)
	if err != nil {
		t.Errorf("buildMessages returned an error: %+v", err)
	}

	<-done1

	// Check that partsToResend contains all parts because none should have
	// errored
	if len(partsToResend) != len(queuedParts)-2 {
		sort.SliceStable(partsToResend, func(i, j int) bool {
			return partsToResend[i] < partsToResend[j]
		})
		for i, j := range partsToResend {
			if i != j {
				t.Errorf("Part at index %d not found. Found %d.", i, j)
			}
		}
	}

	if len(messages) != 3 {
		t.Errorf("Length of messages incorrect."+
			"\nexpected: %d\nreceived: %d", 3, len(messages))
	}

	if len(transfers) != 1 {
		t.Errorf("Length of transfers incorrect."+
			"\nexpected: %d\nreceived: %d", 1, len(transfers))
	}

	if len(groupedParts) != 1 {
		t.Errorf("Length of grouped parts incorrect."+
			"\nexpected: %d\nreceived: %d", 1, len(groupedParts))
	}
}

// Tests that Manager.buildMessages returns the expected error when a queued
// part has an invalid part number.
func TestManager_buildMessages_NewCmixMessageError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	// Add transfer
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, parts := newFile(12, 405, prng, t)
	tid, err := m.sent.AddTransfer(recipient, key, parts, 15, nil, 0, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Create queued part list and add a single part with an invalid part number
	queuedParts := []queuedPart{{tid, uint16(len(parts))}}

	// Build the messages
	expectedErr := fmt.Sprintf(
		newCmixMessageErr, queuedParts[0].partNum, tid, "")
	_, _, _, _, err = m.buildMessages(queuedParts, prng)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMessages did not return the expected error when the "+
			"queuedPart part number is invalid.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}

}

// Tests that Manager.newCmixMessage returns a format.Message with the correct
// MAC, fingerprint, and contents.
func TestManager_newCmixMessage(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)
	prng := NewPrng(42)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	recipient := id.NewIdFromString("recipient", id.User, t)
	partSize, _ := m.getPartSize()
	_, parts := newFile(16, uint32(partSize), prng, t)
	numFps := calcNumberOfFingerprints(uint16(len(parts)), 1.5)
	kv := versioned.NewKV(make(ekv.Memstore))

	transfer, err := ftStorage.NewSentTransfer(recipient, tid, key, parts,
		numFps, nil, 0, kv)
	if err != nil {
		t.Errorf("Failed to create a new SentTransfer: %+v", err)
	}

	cmixMsg, err := m.newCmixMessage(transfer, 0, prng)
	if err != nil {
		t.Errorf("newCmixMessage returned an error: %+v", err)
	}

	fp := ftCrypto.GenerateFingerprints(key, numFps)[0]

	if cmixMsg.GetKeyFP() != fp {
		t.Errorf("cMix message has wrong fingerprint."+
			"\nexpected: %s\nrecieved: %s", fp, cmixMsg.GetKeyFP())
	}

	partMsg, err := unmarshalPartMessage(cmixMsg.GetContents())
	if err != nil {
		t.Errorf("Failed to unmarshal part message: %+v", err)
	}

	decrPart, err := ftCrypto.DecryptPart(key, partMsg.getPart(),
		partMsg.getPadding(), cmixMsg.GetMac(), partMsg.getPartNum())
	if err != nil {
		t.Errorf("Failed to decrypt file part: %+v", err)
	}

	if !bytes.Equal(decrPart, parts[0]) {
		t.Errorf("Decrypted part does not match expected."+
			"\nexpected: %q\nreceived: %q", parts[0], decrPart)
	}
}

// Tests that Manager.makeRoundEventCallback returns a callback that calls the
// progress callback when a round succeeds.
func TestManager_makeRoundEventCallback(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	// Add transfer
	type callbackResults struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	callbackChan := make(chan callbackResults, 10)
	progressCB := func(completed bool, sent, arrived, total uint16,
		t interfaces.FilePartTracker, err error) {
		callbackChan <- callbackResults{completed, sent, arrived, total, err}
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Error("Timed out waiting for callback.")
			case r := <-callbackChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					if r.err != nil {
						t.Errorf("Callback received error: %v", r.err)
					}
					if !r.completed {
						t.Error("File not marked as completed.")
					}
					done1 <- true
				}
			}
		}
	}()

	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, parts := newFile(4, 64, prng, t)
	tid, err := m.sent.AddTransfer(
		recipient, key, parts, 6, progressCB, time.Millisecond, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	<-done0

	// Create queued part list add parts
	queuedParts := []queuedPart{
		{tid, 0},
		{tid, 1},
		{tid, 2},
		{tid, 3},
	}

	rid := id.Round(42)

	_, transfers, groupedParts, _, err := m.buildMessages(queuedParts, prng)

	// Set all parts to in-progress
	for tid, transfer := range transfers {
		_, _ = transfer.SetInProgress(rid, groupedParts[tid]...)
	}

	roundEventCB := m.makeRoundEventCallback(rid, tid, transfers[tid])

	roundEventCB(true, false, nil)

	<-done1
}

// Tests that Manager.makeRoundEventCallback returns a callback that calls the
// progress callback with the correct error when a round fails.
func TestManager_makeRoundEventCallback_RoundFailure(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	rid := id.Round(42)

	// Add transfer
	type callbackResults struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	callbackChan := make(chan callbackResults, 10)
	progressCB := func(completed bool, sent, arrived, total uint16,
		t interfaces.FilePartTracker, err error) {
		callbackChan <- callbackResults{completed, sent, arrived, total, err}
	}

	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, parts := newFile(4, 64, prng, t)
	tid, err := m.sent.AddTransfer(
		recipient, key, parts, 6, progressCB, time.Millisecond, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}
	partsToSend := []uint16{0, 1, 2, 3}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Error("Timed out waiting for callback.")
			case r := <-callbackChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					expectedErr := fmt.Sprintf(
						roundFailureCbErr, partsToSend, tid, rid, roundFailErr)
					if r.err == nil || !strings.Contains(r.err.Error(), expectedErr) {
						t.Errorf("Callback received unexpected error when round "+
							"failed.\nexpected: %s\nreceived: %v", expectedErr, r.err)
					}
					done1 <- true
				}
			}
		}
	}()

	// Create queued part list add parts
	queuedParts := make([]queuedPart, len(partsToSend))
	for i := range queuedParts {
		queuedParts[i] = queuedPart{tid, uint16(i)}
	}

	_, transfers, groupedParts, _, err := m.buildMessages(queuedParts, prng)

	<-done0

	// Set all parts to in-progress
	for tid, transfer := range transfers {
		_, _ = transfer.SetInProgress(rid, groupedParts[tid]...)
	}

	roundEventCB := m.makeRoundEventCallback(rid, tid, transfers[tid])

	roundEventCB(false, false, nil)

	<-done1
}

// Panic path: tests that Manager.makeRoundEventCallback panics when
// SentTransfer.FinishTransfer returns an error because the file parts had not
// previously been set to in-progress.
func TestManager_makeRoundEventCallback_FinishTransferPanic(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	_, parts := newFile(4, 64, prng, t)
	tid, err := m.sent.AddTransfer(
		recipient, key, parts, 6, nil, time.Millisecond, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Create queued part list add parts
	queuedParts := []queuedPart{{tid, 0}, {tid, 1}, {tid, 2}, {tid, 3}}
	rid := id.Round(42)

	_, transfers, _, _, err := m.buildMessages(queuedParts, prng)

	expectedErr := strings.Split(finishTransferPanic, "%")[0]
	defer func() {
		err2 := recover()
		if err2 == nil || !strings.Contains(err2.(string), expectedErr) {
			t.Errorf("makeRoundEventCallback failed to panic or returned the "+
				"wrong error.\nexpected: %s\nreceived: %+v", expectedErr, err2)
		}
	}()

	roundEventCB := m.makeRoundEventCallback(rid, tid, transfers[tid])

	roundEventCB(true, false, nil)
}

// Tests that Manager.queueParts adds all the expected parts to the sendQueue
// channel.
func TestManager_queueParts(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)
	csPrng := NewPrng(42)
	prng := rand.New(rand.NewSource(42))

	// Create map of expected parts
	n := 26
	parts := make(map[ftCrypto.TransferID]map[uint16]bool, n)
	for i := 0; i < n; i++ {
		tid, _ := ftCrypto.NewTransferID(csPrng)
		o := uint16(prng.Int31n(64))
		parts[tid] = make(map[uint16]bool, o)
		for j := uint16(0); j < o; j++ {
			parts[tid][j] = true
		}
	}

	// Queue all parts
	for tid, partList := range parts {
		m.queueParts(tid, uint16(len(partList)))
	}

	// Read through all parts in channel ensuring none are missing or duplicate
	for len(parts) > 0 {
		select {
		case part := <-m.sendQueue:
			partList, exists := parts[part.tid]
			if !exists {
				t.Errorf("Could not find transfer %s", part.tid)
			} else {
				_, exists := partList[part.partNum]
				if !exists {
					t.Errorf("Could not find part number %d for transfer %s",
						part.partNum, part.tid)
				}

				delete(parts[part.tid], part.partNum)

				if len(parts[part.tid]) == 0 {
					delete(parts, part.tid)
				}
			}
		case <-time.NewTimer(time.Millisecond).C:
			if len(parts) != 0 {
				t.Errorf("Timed out reading parts from channel. Failed to "+
					"read parts for %d/%d transfers.", len(parts), n)
			}
			return
		default:
			if len(parts) != 0 {
				t.Errorf("Failed to read parts for %d/%d transfers.",
					len(parts), n)
			}
		}
	}
}

// Tests that getShuffledPartNumList returns a list with all the part numbers.
func Test_getShuffledPartNumList(t *testing.T) {
	n := 100
	numList := getShuffledPartNumList(uint16(n))

	if len(numList) != n {
		t.Errorf("Length of shuffled list incorrect."+
			"\nexpected: %d\nreceived: %d", n, len(numList))
	}

	for i := 0; i < n; i++ {
		j := 0
		for ; i != int(numList[j]); j++ {
		}
		if i != int(numList[j]) {
			t.Errorf("Failed to find part number %d in shuffled list.", i)
		}
	}
}

// Tests that the part size returned by Manager.getPartSize matches the manually
// calculated part size.
func TestManager_getPartSize(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, t)

	// Calculate the expected part size
	primeByteLen := m.store.Cmix().GetGroup().GetP().ByteLen()
	cmixMsgUsedLen := format.AssociatedDataSize
	filePartMsgUsedLen := fmMinSize
	expected := 2*primeByteLen - cmixMsgUsedLen - filePartMsgUsedLen

	// Get the part size
	partSize, err := m.getPartSize()
	if err != nil {
		t.Errorf("getPartSize returned an error: %+v", err)
	}

	if expected != partSize {
		t.Errorf("Returned part size does not match expected."+
			"\nexpected: %d\nreceived: %d", expected, partSize)
	}
}

// Tests that partitionFile partitions the given file into the expected parts.
func Test_partitionFile(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	partSize := 96
	fileData, expectedParts := newFile(24, uint32(partSize), prng, t)

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

// Tests that getRandomNumParts returns values between minPartsSendPerRound and
// maxPartsSendPerRound.
func Test_getRandomNumParts(t *testing.T) {
	prng := NewPrng(42)

	for i := 0; i < 100; i++ {
		num := getRandomNumParts(prng)
		if num < minPartsSendPerRound {
			t.Errorf("Number %d is smaller than minimum %d (%d)",
				num, minPartsSendPerRound, i)
		} else if num > maxPartsSendPerRound {
			t.Errorf("Number %d is greater than maximum %d (%d)",
				num, maxPartsSendPerRound, i)
		}
	}
}

// Tests that getRandomNumParts panics for a PRNG that errors.
func Test_getRandomNumParts_PrngPanic(t *testing.T) {
	prng := NewPrngErr()

	defer func() {
		// Error if a panic does not occur
		if err := recover(); err == nil {
			t.Error("Failed to panic for broken PRNG.")
		}
	}()

	_ = getRandomNumParts(prng)
}

// Tests that getRandomNumParts satisfies the getRngNum type.
func Test_getRandomNumParts_GetRngNumType(t *testing.T) {
	var _ getRngNum = getRandomNumParts
}
