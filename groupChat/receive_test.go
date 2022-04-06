///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests that State.receive returns the correct message on the callback.
func TestManager_receive(t *testing.T) {
	// Setup callback
	msgChan := make(chan MessageReceive)
	receiveFunc := func(msg MessageReceive) { msgChan <- msg }

	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, receiveFunc, t)

	// Create test parameters
	contents := []byte("Test group message.")
	timestamp := netTime.Now()
	sender := m.gs.GetUser()

	expectedMsg := MessageReceive{
		GroupID:        g.ID,
		ID:             group.MessageID{0, 1, 2, 3},
		Payload:        contents,
		SenderID:       sender.ID,
		Timestamp:      timestamp.Local(),
		RoundTimestamp: timestamp,
	}

	// Create cMix message and get public message
	cMixMsg, err := m.newCmixMsg(g, contents, timestamp, g.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}

	intlMsg, _ := newInternalMsg(cMixMsg.ContentsSize() - publicMinLen)
	intlMsg.SetTimestamp(timestamp)
	intlMsg.SetSenderID(m.gs.GetUser().ID)
	intlMsg.SetPayload(contents)
	expectedMsg.ID = group.NewMessageID(g.ID, intlMsg.Marshal())

	receiveChan := make(chan message.Receive, 1)
	stop := stoppable.NewSingle("singleStoppable")

	m.gs.SetUser(g.Members[4], t)
	go m.receive(receiveChan, stop)

	receiveChan <- message.Receive{
		Payload:        cMixMsg.Marshal(),
		RoundTimestamp: timestamp,
	}

	select {
	case msg := <-msgChan:
		if !reflect.DeepEqual(expectedMsg, msg) {
			t.Errorf("Failed to received expected message."+
				"\nexpected: %+v\nreceived: %+v", expectedMsg, msg)
		}
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Errorf("Timed out waiting to receive group message.")
	}
}

// Tests that the callback is not called when the message cannot be read.
func TestManager_receive_ReadMessageError(t *testing.T) {
	// Setup callback
	msgChan := make(chan MessageReceive)
	receiveFunc := func(msg MessageReceive) { msgChan <- msg }

	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, receiveFunc, t)

	receiveChan := make(chan message.Receive, 1)
	stop := stoppable.NewSingle("singleStoppable")

	go m.receive(receiveChan, stop)

	receiveChan <- message.Receive{
		Payload: make([]byte, format.MinimumPrimeSize*2),
	}

	select {
	case <-msgChan:
		t.Error("Callback called when message should have errored.")
	case <-time.NewTimer(5 * time.Millisecond).C:
	}
}

// Tests that the quit channel exits the function.
func TestManager_receive_QuitChan(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, nil, t)

	receiveChan := make(chan message.Receive, 1)
	stop := stoppable.NewSingle("singleStoppable")
	doneChan := make(chan struct{})

	go func() {
		m.receive(receiveChan, stop)
		doneChan <- struct{}{}
	}()

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	select {
	case <-doneChan:
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Errorf("Timed out waiting for thread to quit.")
	}
}

// Tests that State.readMessage returns the message data for the correct
// group.
func TestManager_readMessage(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, expectedGrp := newTestManagerWithStore(prng, 10, 0, nil, nil, t)

	// Create test parameters
	expectedContents := []byte("Test group message.")
	expectedTimestamp := netTime.Now()
	sender := m.gs.GetUser()

	// Create cMix message and get public message
	cMixMsg, err := m.newCmixMsg(expectedGrp, expectedContents,
		expectedTimestamp, expectedGrp.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}

	internalMsg, _ := newInternalMsg(cMixMsg.ContentsSize() - publicMinLen)
	internalMsg.SetTimestamp(expectedTimestamp)
	internalMsg.SetSenderID(sender.ID)
	internalMsg.SetPayload(expectedContents)
	expectedMsgID := group.NewMessageID(expectedGrp.ID, internalMsg.Marshal())

	// Build message.Receive
	receiveMsg := message.Receive{
		ID:             e2e.MessageID{},
		Payload:        cMixMsg.Marshal(),
		RoundTimestamp: expectedTimestamp,
	}

	m.gs.SetUser(expectedGrp.Members[4], t)
	g, messageID, timestamp, senderID, contents, noFpMatch, err :=
		m.readMessage(receiveMsg)
	if err != nil {
		t.Errorf("readMessage() returned an error: %+v", err)
	}

	if noFpMatch {
		t.Error("Fingerprint did not match when it should have.")
	}

	if !reflect.DeepEqual(expectedGrp, g) {
		t.Errorf("readMessage() returned incorrect group."+
			"\nexpected: %#v\nreceived: %#v", expectedGrp, g)
	}

	if expectedMsgID != messageID {
		t.Errorf("readMessage() returned incorrect message ID."+
			"\nexpected: %s\nreceived: %s", expectedMsgID, messageID)
	}

	if !expectedTimestamp.Equal(timestamp) {
		t.Errorf("readMessage() returned incorrect timestamp."+
			"\nexpected: %s\nreceived: %s", expectedTimestamp, timestamp)
	}

	if !sender.ID.Cmp(senderID) {
		t.Errorf("readMessage() returned incorrect sender ID."+
			"\nexpected: %s\nreceived: %s", sender.ID, senderID)
	}

	if !bytes.Equal(expectedContents, contents) {
		t.Errorf("readMessage() returned incorrect message."+
			"\nexpected: %s\nreceived: %s", expectedContents, contents)
	}
}

// Error path: an error is returned when a group with a matching group
// fingerprint cannot be found.
func TestManager_readMessage_FindGroupKpError(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, nil, t)

	// Create test parameters
	expectedContents := []byte("Test group message.")
	expectedTimestamp := netTime.Now()

	// Create cMix message and get public message
	cMixMsg, err := m.newCmixMsg(
		g, expectedContents, expectedTimestamp, g.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}

	cMixMsg.SetKeyFP(format.NewFingerprint([]byte("invalid Fingerprint")))

	// Build message.Receive
	receiveMsg := message.Receive{
		ID:             e2e.MessageID{},
		Payload:        cMixMsg.Marshal(),
		RoundTimestamp: expectedTimestamp,
	}

	expectedErr := strings.SplitN(findGroupKeyFpErr, "%", 2)[0]

	m.gs.SetUser(g.Members[4], t)
	_, _, _, _, _, _, err = m.readMessage(receiveMsg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readMessage() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that a cMix message created by State.newCmixMsg can be read by
// State.readMessage.
func TestManager_decryptMessage(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManager(prng, t)

	// Create test parameters
	expectedContents := []byte("Test group message.")
	expectedTimestamp := netTime.Now()

	// Create cMix message and get public message
	msg, err := m.newCmixMsg(
		g, expectedContents, expectedTimestamp, g.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	publicMsg, err := unmarshalPublicMsg(msg.GetContents())
	if err != nil {
		t.Errorf("Failed to unmarshal publicMsg: %+v", err)
	}

	internalMsg, _ := newInternalMsg(publicMsg.GetPayloadSize())
	internalMsg.SetTimestamp(expectedTimestamp)
	internalMsg.SetSenderID(m.gs.GetUser().ID)
	internalMsg.SetPayload(expectedContents)
	expectedMsgID := group.NewMessageID(g.ID, internalMsg.Marshal())

	// Read message and check if the outputs are correct
	messageID, timestamp, senderID, contents, err := m.decryptMessage(g, msg,
		publicMsg, expectedTimestamp)
	if err != nil {
		t.Errorf("decryptMessage() returned an error: %+v", err)
	}

	if expectedMsgID != messageID {
		t.Errorf("decryptMessage() returned incorrect message ID."+
			"\nexpected: %s\nreceived: %s", expectedMsgID, messageID)
	}

	if !expectedTimestamp.Equal(timestamp) {
		t.Errorf("decryptMessage() returned incorrect timestamp."+
			"\nexpected: %s\nreceived: %s", expectedTimestamp, timestamp)
	}

	if !m.gs.GetUser().ID.Cmp(senderID) {
		t.Errorf("decryptMessage() returned incorrect sender ID."+
			"\nexpected: %s\nreceived: %s", m.gs.GetUser().ID, senderID)
	}

	if !bytes.Equal(expectedContents, contents) {
		t.Errorf("decryptMessage() returned incorrect message."+
			"\nexpected: %s\nreceived: %s", expectedContents, contents)
	}
}

// Error path: an error is returned when the wrong timestamp is passed in and
// the decryption key cannot be generated because of the wrong epoch.
func TestManager_decryptMessage_GetCryptKeyError(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManager(prng, t)

	// Create test parameters
	contents := []byte("Test group message.")
	timestamp := netTime.Now()

	// Create cMix message and get public message
	msg, err := m.newCmixMsg(g, contents, timestamp, g.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	publicMsg, err := unmarshalPublicMsg(msg.GetContents())
	if err != nil {
		t.Errorf("Failed to unmarshal publicMsg: %+v", err)
	}

	// Check if error is correct
	expectedErr := strings.SplitN(genCryptKeyMacErr, "%", 2)[0]
	_, _, _, _, err = m.decryptMessage(g, msg, publicMsg, timestamp.Add(time.Hour))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("decryptMessage() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned when the decrypted payload cannot be
// unmarshalled.
func TestManager_decryptMessage_UnmarshalInternalMsgError(t *testing.T) {
	// Create new test State and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManager(prng, t)

	// Create test parameters
	contents := []byte("Test group message.")
	timestamp := netTime.Now()

	// Create cMix message and get public message
	msg, err := m.newCmixMsg(g, contents, timestamp, g.Members[4], prng)
	if err != nil {
		t.Errorf("Failed to create new cMix message: %+v", err)
	}
	publicMsg, err := unmarshalPublicMsg(msg.GetContents())
	if err != nil {
		t.Errorf("Failed to unmarshal publicMsg: %+v", err)
	}

	// Modify publicMsg to have invalid payload
	publicMsg = mapPublicMsg(publicMsg.Marshal()[:33])
	key, err := group.NewKdfKey(
		g.Key, group.ComputeEpoch(timestamp), publicMsg.GetSalt())
	if err != nil {
		t.Errorf("failed to create new key: %+v", err)
	}
	msg.SetMac(
		group.NewMAC(key, publicMsg.GetPayload(), g.DhKeys[*g.Members[4].ID]))

	// Check if error is correct
	expectedErr := strings.SplitN(unmarshalInternalMsgErr, "%", 2)[0]
	_, _, _, _, err = m.decryptMessage(g, msg, publicMsg, timestamp)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("decryptMessage() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of getCryptKey.
func Test_getCryptKey(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	g := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
	salt, err := newSalt(prng)
	if err != nil {
		t.Errorf("failed to create new salt: %+v", err)
	}
	payload := []byte("payload")
	ts := netTime.Now()

	expectedKey, err := group.NewKdfKey(
		g.Key, group.ComputeEpoch(ts.Add(5*time.Minute)), salt)
	if err != nil {
		t.Errorf("failed to create new key: %+v", err)
	}
	mac := group.NewMAC(expectedKey, payload, g.DhKeys[*g.Members[4].ID])

	key, err := getCryptKey(g.Key, salt, mac, payload, g.DhKeys, ts)
	if err != nil {
		t.Errorf("getCryptKey() returned an error: %+v", err)
	}

	if expectedKey != key {
		t.Errorf("getCryptKey() did not return the expected key."+
			"\nexpected: %v\nreceived: %v", expectedKey, key)
	}
}

// Error path: return an error when the MAC cannot be verified because the
// timestamp is incorrect and generates the wrong epoch.
func Test_getCryptKey_EpochError(t *testing.T) {
	expectedErr := strings.SplitN(genCryptKeyMacErr, "%", 2)[0]

	prng := rand.New(rand.NewSource(42))
	g := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
	salt, err := newSalt(prng)
	if err != nil {
		t.Errorf("failed to create new salt: %+v", err)
	}
	payload := []byte("payload")
	ts := netTime.Now()

	key, err := group.NewKdfKey(g.Key, group.ComputeEpoch(ts), salt)
	if err != nil {
		t.Errorf("getCryptKey() returned an error: %+v", err)
	}
	mac := group.NewMAC(key, payload, g.Members[4].DhKey)

	_, err = getCryptKey(g.Key, salt, mac, payload, g.DhKeys, ts.Add(time.Hour))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("getCryptKey() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
