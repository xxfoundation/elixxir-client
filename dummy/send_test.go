////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// Tests that Manager.sendThread sends multiple sets of messages.
func TestManager_sendThread(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, t)

	stop := stoppable.NewSingle("sendThreadTest")
	go m.sendThread(stop)

	if stat := atomic.LoadUint32(&m.status); stat != notStarted {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			notStarted, stat)
	}

	err := m.Start()
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

// Tests that sendMessage generates random message data using pseudo-RNGs.
func TestManager_sendMessage(t *testing.T) {
	m := newTestManager(100, 0, 0, t)

	// Generate two identical RNGs, one for generating expected data (newRandomCmixMessage)
	// and one for received data (sendMessage)
	prngOne := NewPrng(42)
	prngTwo := NewPrng(42)

	// Generate map of recipients and messages
	msgs := make(map[id.ID]format.Message, m.maxNumMessages)
	for i := 0; i < m.maxNumMessages; i++ {
		// Generate random data
		recipient, fp, service, payload, mac, err := m.newRandomCmixMessage(prngOne)
		if err != nil {
			t.Fatalf("Failed to generate random cMix message (%d): %+v", i, err)
		}

		payloadSize := m.store.GetCmixGroup().GetP().ByteLen()
		msgs[*recipient] = generateMessage(payloadSize, fp, service, payload, mac)

		// Send the messages
		err = m.sendMessage(i, m.maxNumMessages, prngTwo)
		if err != nil {
			t.Errorf("sendMessages returned an error: %+v", err)
		}

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
			t.Errorf("Failed to receive message from %s: %+v", &recipient, msg.Marshal())
		} else if !reflect.DeepEqual(msg.Marshal(), receivedMsg.Marshal()) {
			// In mockCmix.Send, we map recipientId to the passed fingerprint.
			t.Errorf("Received unexpected message for recipient %s."+
				"\nexpected: %+v\nreceived: %+v", &recipient, msg, receivedMsg)
		}
	}
}

// Tests that newRandomCmixMessage generates cMix message data with
// populated recipient, payload, fingerprint, and MAC.
func TestManager_newRandomCmixMessage(t *testing.T) {
	m := newTestManager(0, 0, 0, t)
	prng := NewPrng(42)

	// Generate data
	recipient, fp, _, payload, mac, err := m.newRandomCmixMessage(prng)
	if err != nil {
		t.Fatalf("newRandomCmixMessage returned an error: %+v", err)
	}

	// Check that recipient is not empty data
	if bytes.Equal(recipient.Bytes(), make([]byte, id.ArrIDLen)) {
		t.Errorf("Recipient ID not set")
	}

	// Check that payload is not empty data
	payloadSize := m.store.GetCmixGroup().GetP().ByteLen()
	if bytes.Equal(payload, make([]byte, payloadSize)) {
		t.Error("cMix message contents not set.")
	}

	// Check that fingerprint is not empty data
	if fp == (format.Fingerprint{}) {
		t.Error("cMix message fingerprint not set.")
	}

	// Check that mac is not empty data
	if bytes.Equal(mac, make([]byte, format.MacLen)) {
		t.Error("cMix message MAC not set.")
	}

}
