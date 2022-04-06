////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that State.receive returns the correct progress on the callback when
// receiving a single message.
func TestManager_receive(t *testing.T) {
	// Build a manager for sending and a manger for receiving
	m1 := newTestManager(false, nil, nil, nil, nil, t)
	m2 := newTestManager(false, nil, nil, nil, nil, t)

	// Create transfer components
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	numParts := uint16(16)
	numFps := calcNumberOfFingerprints(numParts, 0.5)
	partSize, _ := m1.getPartSize()
	file, parts := newFile(numParts, partSize, prng, t)
	fileSize := uint32(len(file))
	mac := ftCrypto.CreateTransferMAC(file, key)

	// Add transfer to sending manager
	stID, err := m1.sent.AddTransfer(
		recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("Failed to add new sent transfer: %+v", err)
	}

	// Add transfer to receiving manager
	rtID, err := m2.received.AddTransfer(
		key, mac, fileSize, numParts, numFps, prng)
	if err != nil {
		t.Errorf("Failed to add new received transfer: %+v", err)
	}

	// Generate receive callback that should be called when a message is read
	cbChan := make(chan receivedProgressResults)
	cb := func(completed bool, received, total uint16,
		tr interfaces.FilePartTracker, err error) {
		cbChan <- receivedProgressResults{completed, received, total, tr, err}
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Error("Timed out waiting for callback to be called.")
			case r := <-cbChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					err = checkReceivedProgress(r.completed, r.received, r.total, false,
						1, numParts)
					if r.err != nil {
						t.Errorf("Callback returned an error: %+v", err)
					}
					if err != nil {
						t.Error(err)
					}
					done1 <- true
				}
			}
		}
	}()

	rt, err := m2.received.GetTransfer(rtID)
	if err != nil {
		t.Errorf("Failed to get received transfer %s: %+v", rtID, err)
	}
	rt.AddProgressCB(cb, time.Millisecond)

	<-done0

	// Build message.Receive with cMix message that has file part
	st, err := m1.sent.GetTransfer(stID)
	if err != nil {
		t.Errorf("Failed to get sent transfer %s: %+v", stID, err)
	}
	cMixMsg, err := m1.newCmixMessage(st, 0)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	receiveMsg := message.Receive{Payload: cMixMsg.Marshal()}

	// Start reception thread
	rawMsgs := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle(filePartStoppableName)
	go m2.receive(rawMsgs, stop)

	// Send message on channel;
	rawMsgs <- receiveMsg

	<-done1

	// Create cMix message with wrong fingerprint
	cMixMsg.SetKeyFP(format.NewFingerprint([]byte("invalidFP")))
	receiveMsg = message.Receive{Payload: cMixMsg.Marshal()}

	done := make(chan bool)
	go func() {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
		case r := <-cbChan:
			t.Errorf("Callback should not be called for invalid message: %+v", r)
		}
		done <- true
	}()

	// Send message on channel;
	rawMsgs <- receiveMsg

	<-done
}

// Tests that State.receive the progress callback is not called when the
// stoppable is triggered.
func TestManager_receive_Stop(t *testing.T) {
	// Build a manager for sending and a manger for receiving
	m1 := newTestManager(false, nil, nil, nil, nil, t)
	m2 := newTestManager(false, nil, nil, nil, nil, t)

	// Create transfer components
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	numParts := uint16(16)
	numFps := calcNumberOfFingerprints(numParts, 0.5)
	partSize, _ := m1.getPartSize()
	file, parts := newFile(numParts, partSize, prng, t)
	fileSize := uint32(len(file))
	mac := ftCrypto.CreateTransferMAC(file, key)

	// Add transfer to sending manager
	stID, err := m1.sent.AddTransfer(
		recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("Failed to add new sent transfer: %+v", err)
	}

	// Add transfer to receiving manager
	rtID, err := m2.received.AddTransfer(
		key, mac, fileSize, numParts, numFps, prng)
	if err != nil {
		t.Errorf("Failed to add new received transfer: %+v", err)
	}

	// Generate receive callback that should be called when a message is read
	cbChan := make(chan receivedProgressResults)
	cb := func(completed bool, received, total uint16,
		tr interfaces.FilePartTracker, err error) {
		cbChan <- receivedProgressResults{completed, received, total, tr, err}
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(20 * time.Millisecond).C:
				done1 <- true
			case r := <-cbChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					t.Errorf("Callback should not have been called: %+v", r)
					done1 <- true
				}
			}
		}
	}()

	rt, err := m2.received.GetTransfer(rtID)
	if err != nil {
		t.Errorf("Failed to get received transfer %s: %+v", rtID, err)
	}
	rt.AddProgressCB(cb, time.Millisecond)

	<-done0

	// Build message.Receive with cMix message that has file part
	st, err := m1.sent.GetTransfer(stID)
	if err != nil {
		t.Errorf("Failed to get sent transfer %s: %+v", stID, err)
	}
	cMixMsg, err := m1.newCmixMessage(st, 0)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	receiveMsg := message.Receive{
		Payload: cMixMsg.Marshal(),
	}

	// Start reception thread
	rawMsgs := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle(filePartStoppableName)
	go m2.receive(rawMsgs, stop)

	// Trigger stoppable
	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}

	for stop.IsStopping() {

	}

	// Send message on channel;
	rawMsgs <- receiveMsg

	<-done1
}

// Tests that State.readMessage reads the message without errors and that it
// reports the correct progress on the callback. It also gets the file and
// checks that the part is where it should be.
func TestManager_readMessage(t *testing.T) {

	// Build a manager for sending and a manger for receiving
	m1 := newTestManager(false, nil, nil, nil, nil, t)
	m2 := newTestManager(false, nil, nil, nil, nil, t)

	// Create transfer components
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	numParts := uint16(16)
	numFps := calcNumberOfFingerprints(numParts, 0.5)
	partSize, _ := m1.getPartSize()
	file, parts := newFile(numParts, partSize, prng, t)
	fileSize := uint32(len(file))
	mac := ftCrypto.CreateTransferMAC(file, key)

	// Add transfer to sending manager
	stID, err := m1.sent.AddTransfer(
		recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("Failed to add new sent transfer: %+v", err)
	}

	// Add transfer to receiving manager
	rtID, err := m2.received.AddTransfer(
		key, mac, fileSize, numParts, numFps, prng)
	if err != nil {
		t.Errorf("Failed to add new received transfer: %+v", err)
	}

	// Generate receive callback that should be called when a message is read
	cbChan := make(chan receivedProgressResults, 2)
	cb := func(completed bool, received, total uint16,
		tr interfaces.FilePartTracker, err error) {
		cbChan <- receivedProgressResults{completed, received, total, tr, err}
	}

	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Error("Timed out waiting for callback to be called.")
			case r := <-cbChan:
				switch i {
				case 0:
					done0 <- true
				case 1:
					err := checkReceivedProgress(r.completed, r.received, r.total, false,
						1, numParts)
					if r.err != nil {
						t.Errorf("Callback returned an error: %+v", err)
					}
					if err != nil {
						t.Error(err)
					}
					done1 <- true
				}
			}
		}
	}()

	rt, err := m2.received.GetTransfer(rtID)
	if err != nil {
		t.Errorf("Failed to get received transfer %s: %+v", rtID, err)
	}
	rt.AddProgressCB(cb, time.Millisecond)

	<-done0

	// Build message.Receive with cMix message that has file part
	st, err := m1.sent.GetTransfer(stID)
	if err != nil {
		t.Errorf("Failed to get sent transfer %s: %+v", stID, err)
	}
	cMixMsg, err := m1.newCmixMessage(st, 0)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	receiveMsg := message.Receive{
		Payload: cMixMsg.Marshal(),
	}

	// Read receive message
	receivedCMixMsg, err := m2.readMessage(receiveMsg)
	if err != nil {
		t.Errorf("readMessage returned an error: %+v", err)
	}

	if !reflect.DeepEqual(cMixMsg, receivedCMixMsg) {
		t.Errorf("Received cMix message does not match sent."+
			"\nexpected: %+v\nreceived: %+v", cMixMsg, receivedCMixMsg)
	}

	<-done1

	// Get the file and check that the part was added to it
	fileData, err := rt.GetFile()
	if err == nil {
		t.Error("GetFile did not return an error when parts are missing.")
	}

	if !bytes.Equal(parts[0], fileData[:partSize]) {
		t.Errorf("Part failed to be added to part store."+
			"\nexpected: %q\nreceived: %q", parts[0], fileData[:partSize])
	}
}
