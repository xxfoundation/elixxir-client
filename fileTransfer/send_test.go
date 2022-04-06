////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/stoppable"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/diffieHellman"
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
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, true, nil, nil, nil, t)

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
				case <-time.NewTimer(160 * time.Millisecond).C:
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

	// Create custom RNG function that always returns 11
	getNumParts := func(rng csprng.Source) int {
		return len(queuedParts)
	}

	// Start sending thread
	stop := stoppable.NewSingle("testSendThreadStoppable")
	healthyChan := make(chan bool, 8)
	healthyChan <- true
	go m.sendThread(stop, healthyChan, 0, getNumParts)

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

// Tests that Manager.sendThread successfully sends the parts and reports their
// progress on the callback.
func TestManager_sendThread_NetworkNotHealthy(t *testing.T) {
	m, _, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, true, nil, nil, nil, t)

	sendingChan := make(chan bool, 4)
	getNumParts := func(csprng.Source) int {
		sendingChan <- true
		return 0
	}

	// Start sending thread
	stop := stoppable.NewSingle("testSendThreadStoppable")
	healthyChan := make(chan bool, 8)
	go m.sendThread(stop, healthyChan, 0, getNumParts)

	for i := 0; i < 15; i++ {
		healthyChan <- false
	}
	m.sendQueue <- queuedPart{ftCrypto.TransferID{5}, 0}

	select {
	case <-time.NewTimer(150 * time.Millisecond).C:
		healthyChan <- true
	case r := <-sendingChan:
		t.Errorf("sendThread tried to send even though the network is "+
			"unhealthy. %t", r)
	}

	select {
	case <-time.NewTimer(150 * time.Millisecond).C:
		t.Errorf("Timed out waiting for sending to start.")
	case <-sendingChan:
	}

	err := stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}
}

// Tests that Manager.sendThread successfully sends a partially filled batch
// of the correct length when its times out waiting for messages.
func TestManager_sendThread_Timeout(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, false, nil, nil, nil, t)

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
				case <-time.NewTimer(80*time.Millisecond + pollSleepDuration).C:
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
	go m.sendThread(stop, make(chan bool), 0, getNumParts)

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
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, true, nil, nil, nil, t)

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

	err := m.sendParts(queuedParts, newSentRoundTracker(clearSentRoundsAge))
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
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, true, false, nil, nil, nil, t)
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

	err := m.sendParts(queuedParts, newSentRoundTracker(clearSentRoundsAge))
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

// Error path: tests that Manager.sendParts returns the expected error whe
// getRoundResults returns an error.
func TestManager_sendParts_RoundResultsError(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12}, false, true, nil, nil, nil, t)

	grrErr := errors.New("GetRoundResultsError")
	m.getRoundResults =
		func([]id.Round, time.Duration, network.RoundEventCallback) error {
			return grrErr
		}

	// Add three transfers
	partsToSend := [][]uint16{
		{0, 1, 3, 5, 6, 7},
	}

	// Create queued part list, add parts from each transfer, and shuffle
	queuedParts := make([]queuedPart, 0, 11)
	tIDs := make([]ftCrypto.TransferID, 0, len(sti))
	for i, sendingParts := range partsToSend {
		for _, part := range sendingParts {
			queuedParts = append(queuedParts, queuedPart{sti[i].tid, part})
		}
		tIDs = append(tIDs, sti[i].tid)
	}

	expectedErr := fmt.Sprintf(getRoundResultsErr, 0, tIDs, grrErr)
	err := m.sendParts(queuedParts, newSentRoundTracker(clearSentRoundsAge))
	if err == nil || err.Error() != expectedErr {
		t.Errorf("sendParts did not return the expected error when "+
			"GetRoundResults should have returned an error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Manager.buildMessages returns the expected values for a group
// of 11 file parts from three different transfers.
func TestManager_buildMessages(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, false, nil, nil, nil, t)
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
	messages, transfers, groupedParts, partsToResend, err :=
		m.buildMessages(queuedParts)
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
	m := newTestManager(false, nil, nil, nil, nil, t)

	callbackChan := make(chan sentProgressResults, 10)
	progressCB := func(completed bool, sent, arrived, total uint16,
		tr interfaces.FilePartTracker, err error) {
		callbackChan <- sentProgressResults{
			completed, sent, arrived, total, tr, err}
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
		queuedParts)
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
	m := newTestManager(false, nil, nil, nil, nil, t)

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
	_, _, _, _, err = m.buildMessages(queuedParts)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMessages did not return the expected error when the "+
			"queuedPart part number is invalid.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}

}

// Tests that Manager.newCmixMessage returns a format.Message with the correct
// MAC, fingerprint, and contents.
func TestManager_newCmixMessage(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	prng := NewPrng(42)
	tid, _ := ftCrypto.NewTransferID(prng)
	key, _ := ftCrypto.NewTransferKey(prng)
	recipient := id.NewIdFromString("recipient", id.User, t)
	partSize, _ := m.getPartSize()
	_, parts := newFile(16, partSize, prng, t)
	numFps := calcNumberOfFingerprints(uint16(len(parts)), 1.5)
	kv := versioned.NewKV(make(ekv.Memstore))

	transfer, err := ftStorage.NewSentTransfer(recipient, tid, key, parts,
		numFps, nil, 0, kv)
	if err != nil {
		t.Errorf("Failed to create a new SentTransfer: %+v", err)
	}

	cmixMsg, err := m.newCmixMessage(transfer, 0)
	if err != nil {
		t.Errorf("newCmixMessage returned an error: %+v", err)
	}

	fp := ftCrypto.GenerateFingerprints(key, numFps)[0]

	if cmixMsg.GetKeyFP() != fp {
		t.Errorf("cMix message has wrong fingerprint."+
			"\nexpected: %s\nrecieved: %s", fp, cmixMsg.GetKeyFP())
	}

	decrPart, err := ftCrypto.DecryptPart(key, cmixMsg.GetContents(),
		cmixMsg.GetMac(), 0, cmixMsg.GetKeyFP())
	if err != nil {
		t.Errorf("Failed to decrypt file part: %+v", err)
	}

	partMsg, err := ftStorage.UnmarshalPartMessage(decrPart)
	if err != nil {
		t.Errorf("Failed to unmarshal part message: %+v", err)
	}

	if !bytes.Equal(partMsg.GetPart(), parts[0]) {
		t.Errorf("Decrypted part does not match expected."+
			"\nexpected: %q\nreceived: %q", parts[0], partMsg.GetPart())
	}
}

// Tests that Manager.makeRoundEventCallback returns a callback that calls the
// progress callback when a round succeeds.
func TestManager_makeRoundEventCallback(t *testing.T) {
	sendE2eChan := make(chan message.Receive, 100)
	m := newTestManager(false, nil, sendE2eChan, nil, nil, t)

	callbackChan := make(chan sentProgressResults, 100)
	progressCB := func(completed bool, sent, arrived, total uint16,
		tr interfaces.FilePartTracker, err error) {
		callbackChan <- sentProgressResults{
			completed, sent, arrived, total, tr, err}
	}

	// Add recipient as partner
	recipient := id.NewIdFromString("recipient", id.User, t)
	grp := m.store.E2e().GetGroup()
	dhKey := grp.NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
	p := params.GetDefaultE2ESessionParams()

	rng := csprng.NewSystemRNG()
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA,
		rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhB, rng)

	err := m.store.E2e().AddPartner(recipient, pubKey, dhKey, mySidhPriv,
		theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback (%d).", i)
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

	_, transfers, groupedParts, _, err := m.buildMessages(queuedParts)

	// Set all parts to in-progress
	for tid, transfer := range transfers {
		_, _ = transfer.SetInProgress(rid, groupedParts[tid]...)
	}

	roundEventCB := m.makeRoundEventCallback(
		map[id.Round][]ftCrypto.TransferID{rid: {tid}})

	roundEventCB(true, false, map[id.Round]network.RoundLookupStatus{rid: network.Succeeded})

	<-done1

	select {
	case <-time.NewTimer(50 * time.Millisecond).C:
		t.Errorf("Timed out waiting for end E2E message.")
	case msg := <-sendE2eChan:
		if msg.MessageType != message.EndFileTransfer {
			t.Errorf("E2E message has wrong type.\nexpected: %d\nreceived: %d",
				message.EndFileTransfer, msg.MessageType)
		} else if !msg.RecipientID.Cmp(recipient) {
			t.Errorf("E2E message has wrong recipient."+
				"\nexpected: %d\nreceived: %d", recipient, msg.RecipientID)
		}
	}
}

// Tests that Manager.makeRoundEventCallback returns a callback that calls the
// progress callback with no parts sent on round failure. Also checks that the
// file parts were added back into the queue.
func TestManager_makeRoundEventCallback_RoundFailure(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	rid := id.Round(42)

	callbackChan := make(chan sentProgressResults, 10)
	progressCB := func(completed bool, sent, arrived, total uint16,
		tr interfaces.FilePartTracker, err error) {
		callbackChan <- sentProgressResults{
			completed, sent, arrived, total, tr, err}
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
					expectedResult := sentProgressResults{
						false, 0, 0, uint16(len(partsToSend)), r.tracker, nil}
					if !reflect.DeepEqual(expectedResult, r) {
						t.Errorf("Callback returned unexpected values."+
							"\nexpected: %+v\nreceived: %+v", expectedResult, r)
					}
					done1 <- true
				}
			}
		}
	}()

	// Create queued part list add parts
	partsMap := make(map[uint16]queuedPart, len(partsToSend))
	queuedParts := make([]queuedPart, len(partsToSend))
	for i := range queuedParts {
		queuedParts[i] = queuedPart{tid, uint16(i)}
		partsMap[uint16(i)] = queuedParts[i]
	}

	_, transfers, groupedParts, _, err := m.buildMessages(queuedParts)

	<-done0

	// Set all parts to in-progress
	for tid, transfer := range transfers {
		_, _ = transfer.SetInProgress(rid, groupedParts[tid]...)
	}

	roundEventCB := m.makeRoundEventCallback(
		map[id.Round][]ftCrypto.TransferID{rid: {tid}})

	roundEventCB(false, false, map[id.Round]network.RoundLookupStatus{rid: network.Failed})

	<-done1

	// Check that the parts were added to the queue
	for i := range partsToSend {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Errorf("Timed out waiting for part %d.", i)
		case r := <-m.sendQueue:
			if partsMap[r.partNum] != r {
				t.Errorf("Incorrect part in queue (%d)."+
					"\nexpected: %+v\nreceived: %+v", i, partsMap[r.partNum], r)
			} else {
				delete(partsMap, r.partNum)
			}
		}
	}
}

// Tests that Manager.sendEndE2eMessage sends an E2E message with the expected
// recipient and message type. This does not test round tracking or critical
// messages.
func TestManager_sendEndE2eMessage(t *testing.T) {
	sendE2eChan := make(chan message.Receive, 10)
	m := newTestManager(false, nil, sendE2eChan, nil, nil, t)

	// Add recipient as partner
	recipient := id.NewIdFromString("recipient", id.User, t)
	grp := m.store.E2e().GetGroup()
	dhKey := grp.NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
	p := params.GetDefaultE2ESessionParams()

	rng := csprng.NewSystemRNG()
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)

	err := m.store.E2e().AddPartner(
		recipient, pubKey, dhKey, mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	go func() {
		err = m.sendEndE2eMessage(recipient)
		if err != nil {
			t.Errorf("sendEndE2eMessage returned an error: %+v", err)
		}
	}()

	select {
	case <-time.NewTimer(50 * time.Millisecond).C:
		t.Errorf("Timed out waiting for end E2E message.")
	case msg := <-sendE2eChan:
		if msg.MessageType != message.EndFileTransfer {
			t.Errorf("E2E message has wrong type.\nexpected: %d\nreceived: %d",
				message.EndFileTransfer, msg.MessageType)
		} else if !msg.RecipientID.Cmp(recipient) {
			t.Errorf("E2E message has wrong recipient."+
				"\nexpected: %d\nreceived: %d", recipient, msg.RecipientID)
		}
	}
}

// Tests that Manager.queueParts adds all the expected parts to the sendQueue
// channel.
func TestManager_queueParts(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
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
		partNums := makeListOfPartNums(uint16(len(partList)))
		m.queueParts(tid, partNums)
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

// Tests that makeListOfPartNums returns a list with all the part numbers.
func Test_makeListOfPartNums(t *testing.T) {
	n := uint16(100)
	numList := makeListOfPartNums(n)

	if len(numList) != int(n) {
		t.Errorf("Length of list incorrect.\nexpected: %d\nreceived: %d",
			n, len(numList))
	}

	for i := uint16(0); i < n; i++ {
		if numList[i] != i {
			t.Errorf("Part number at index %d incorrect."+
				"\nexpected: %d\nreceived: %d", i, i, numList[i])
		}
	}
}

// Tests that the part size returned by Manager.GetPartSize matches the manually
// calculated part size.
func TestManager_getPartSize(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	// Calculate the expected part size
	primeByteLen := m.store.Cmix().GetGroup().GetP().ByteLen()
	cmixMsgUsedLen := format.AssociatedDataSize
	filePartMsgUsedLen := ftStorage.FmMinSize
	expected := 2*primeByteLen - cmixMsgUsedLen - filePartMsgUsedLen - 1

	// Get the part size
	partSize, err := m.getPartSize()
	if err != nil {
		t.Errorf("GetPartSize returned an error: %+v", err)
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
