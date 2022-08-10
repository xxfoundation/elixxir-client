////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"bytes"
	"encoding/base64"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that Manager.sendThread sends multiple sets of messages.
func TestManager_sendThread(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, false, t)

	stop := stoppable.NewSingle("sendThreadTest")
	go m.sendThread(stop)

	if stat := atomic.LoadUint32(&m.status); stat != notStarted {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			notStarted, stat)
	}

	err := m.SetStatus(true)
	if err != nil {
		t.Errorf("Failed to set status to true.")
	}

	msgChan := make(chan bool, 10)
	go func() {
		var numReceived int
		for i := 0; i < 2; i++ {
			for m.net.(*mockCmix).GetMsgListLen() == numReceived {
				time.Sleep(5 * time.Millisecond)
			}
			numReceived = m.net.(*mockCmix).GetMsgListLen()
			msgChan <- true
		}
	}()

	var numReceived int
	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		numReceived += m.net.(*mockCmix).GetMsgListLen()
	}

	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		if m.net.(*mockCmix).GetMsgListLen() <= numReceived {
			t.Errorf("Failed to receive second send."+
				"\nmessages on last receive: %d\nmessages on this receive: %d",
				numReceived, m.net.(*mockCmix).GetMsgListLen())
		}
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if !stop.IsStopped() {
		t.Error("Stoppable never stopped.")
	}

	if stat := atomic.LoadUint32(&m.status); stat != stopped {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			stopped, stat)
	}

}

// Tests that Manager.sendMessages sends all the messages with the correct
// recipient.
func TestManager_sendMessages(t *testing.T) {
	m := newTestManager(100, 0, 0, false, t)
	prng := NewPrng(42)

	// Generate map of recipients and messages
	msgs := make(map[id.ID]format.Message, m.maxNumMessages)
	for i := 0; i < m.maxNumMessages; i++ {
		recipient, err := id.NewRandomID(prng, id.User)
		if err != nil {
			t.Errorf("Failed to generate random recipient ID (%d): %+v", i, err)
		}

		msg, err := m.newRandomCmixMessage(prng)
		if err != nil {
			t.Errorf("Failed to generate random cMix message (%d): %+v", i, err)
		}

		msgs[*recipient] = msg
	}

	// Send the messages
	err := m.sendMessages(msgs, prng)
	if err != nil {
		t.Errorf("sendMessages returned an error: %+v", err)
	}

	// get sent messages
	receivedMsgs := m.net.(*mockCmix).GetMsgList()

	// Test that all messages were received
	if len(receivedMsgs) != len(msgs) {
		t.Errorf("Failed to received all sent messages."+
			"\nexpected: %d\nreceived: %d", len(msgs), len(receivedMsgs))
	}

	// Test that all messages were received for the correct recipient
	for recipient, msg := range msgs {
		receivedMsg, exists := receivedMsgs[recipient]
		if !exists {
			t.Errorf("Failed to receive message from %s: %+v", &recipient, msg)
		} else if !reflect.DeepEqual(msg.GetKeyFP().Bytes(), receivedMsg) {
			// In mockCmix.Send, we map recipientId to the passed fingerprint.
			t.Errorf("Received unexpected message for recipient %s."+
				"\nexpected: %+v\nreceived: %+v", &recipient, msg.GetKeyFP(), receivedMsg)
		}
	}
}

// Tests that Manager.newRandomMessages creates a non-empty map of messages and
// that each message is unique.
func TestManager_newRandomMessages(t *testing.T) {
	m := newTestManager(10, 0, 0, false, t)
	prng := NewPrng(42)

	msgMap, err := m.newRandomMessages(prng)
	if err != nil {
		t.Errorf("newRandomMessages returned an error: %+v", err)
	}

	if len(msgMap) == 0 {
		t.Error("Message map is empty.")
	}

	marshalledMsgs := make(map[string]format.Message, len(msgMap))
	for _, msg := range msgMap {
		msgString := base64.StdEncoding.EncodeToString(msg.Marshal())
		if _, exists := marshalledMsgs[msgString]; exists {
			t.Errorf("Message not unique.")
		} else {
			marshalledMsgs[msgString] = msg
		}
	}
}

// Tests that Manager.newRandomCmixMessage generates a cMix message with
// populated contents, fingerprint, and MAC.
func TestManager_newRandomCmixMessage(t *testing.T) {
	m := newTestManager(0, 0, 0, false, t)
	prng := NewPrng(42)

	cMixMsg, err := m.newRandomCmixMessage(prng)
	if err != nil {
		t.Errorf("newRandomCmixMessage returned an error: %+v", err)
	}

	if bytes.Equal(cMixMsg.GetContents(), make([]byte, len(cMixMsg.GetContents()))) {
		t.Error("cMix message contents not set.")
	}

	if cMixMsg.GetKeyFP() == (format.Fingerprint{}) {
		t.Error("cMix message fingerprint not set.")
	}

	if bytes.Equal(cMixMsg.GetMac(), make([]byte, format.MacLen)) {
		t.Error("cMix message MAC not set.")
	}
}
