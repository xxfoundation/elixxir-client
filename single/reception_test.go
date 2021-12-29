package single

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	contact2 "gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

// Happy path.
func TestManager_receiveTransmissionHandler(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "Test tag"
	payload := make([]byte, 131)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	msg, _, _, _, err := m.makeTransmitCmixMessage(partner, payload, tag, 8, 32,
		30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	m.callbackMap.registerCallback(tag, callback)

	go m.receiveTransmissionHandler(rawMessages, stoppable.NewSingle("singleStoppable"))
	rawMessages <- message.Receive{
		Payload: msg.Marshal(),
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		if !bytes.Equal(results.payload, payload) {
			t.Errorf("Callback received wrong payload."+
				"\nexpected: %+v\nreceived: %+v", payload, results.payload)
		}
	case <-timer.C:
		t.Errorf("Callback failed to be called.")
	}
}

// Happy path: quit channel.
func TestManager_receiveTransmissionHandler_QuitChan(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	tag := "Test tag"
	payload := make([]byte, 132)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	m.callbackMap.registerCallback(tag, callback)

	go m.receiveTransmissionHandler(rawMessages, stop)

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the message should not have been processed."+
			"\npayload: %+v\ncontact: %+v", results.payload, results.c)
	case <-timer.C:
	}
}

// Error path: CMIX message fingerprint does not match.
func TestManager_receiveTransmissionHandler_FingerPrintError(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(42),
	}
	tag := "Test tag"
	payload := make([]byte, 131)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	msg, _, _, _, err := m.makeTransmitCmixMessage(partner, payload, tag, 8, 32,
		30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	m.callbackMap.registerCallback(tag, callback)

	go m.receiveTransmissionHandler(rawMessages, stop)
	rawMessages <- message.Receive{
		Payload: msg.Marshal(),
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the fingerprints do not match."+
			"\npayload: %+v\ncontact: %+v", results.payload, results.c)
	case <-timer.C:
	}
}

// Error path: cannot process transmission message.
func TestManager_receiveTransmissionHandler_ProcessMessageError(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "Test tag"
	payload := make([]byte, 131)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	msg, _, _, _, err := m.makeTransmitCmixMessage(partner, payload, tag, 8, 32,
		30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	msg.SetMac(make([]byte, format.MacLen))

	m.callbackMap.registerCallback(tag, callback)

	go m.receiveTransmissionHandler(rawMessages, stop)
	rawMessages <- message.Receive{
		Payload: msg.Marshal(),
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the message should not have been processed."+
			"\npayload: %+v\ncontact: %+v", results.payload, results.c)
	case <-timer.C:
	}
}

// Error path: tag fingerprint does not match.
func TestManager_receiveTransmissionHandler_TagFpError(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "Test tag"
	payload := make([]byte, 131)
	rand.New(rand.NewSource(42)).Read(payload)

	msg, _, _, _, err := m.makeTransmitCmixMessage(partner, payload, tag, 8, 32,
		30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	go m.receiveTransmissionHandler(rawMessages, stop)
	rawMessages <- message.Receive{
		Payload: msg.Marshal(),
	}
}

// Happy path.
func TestManager_processTransmission(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("partnerID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "test tag"
	payload := []byte("This is the payload.")
	maxMsgs := uint8(6)
	cmixMsg, dhKey, rid, _, err := m.makeTransmitCmixMessage(partner, payload,
		tag, maxMsgs, 32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to generate expected CMIX message: %+v", err)
	}

	tMsg, err := unmarshalTransmitMessage(cmixMsg.GetContents(), m.store.E2e().GetGroup().GetP().ByteLen())
	if err != nil {
		t.Fatalf("Failed to make transmitMessage: %+v", err)
	}

	expectedC := NewContact(rid, tMsg.GetPubKey(m.store.E2e().GetGroup()),
		dhKey, singleUse.NewTagFP(tag), maxMsgs)

	fp := singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey())
	content, testC, err := m.processTransmission(cmixMsg, fp)
	if err != nil {
		t.Errorf("processTransmission() produced an error: %+v", err)
	}

	if !expectedC.Equal(testC) {
		t.Errorf("processTransmission() did not return the expected values."+
			"\nexpected: %+v\nrecieved: %+v", expectedC, testC)
	}

	if !bytes.Equal(payload, content) {
		t.Errorf("processTransmission() returned the wrong payload."+
			"\nexpected: %+v\nreceived: %+v", payload, content)
	}
}

// Error path: fails to unmarshal transmitMessage.
func TestManager_processTransmission_TransmitMessageUnmarshalError(t *testing.T) {
	m := newTestManager(0, false, t)
	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	fp := singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey())
	_, _, err := m.processTransmission(cmixMsg, fp)
	if !check(err, "failed to unmarshal contents") {
		t.Errorf("processTransmission() did not produce an error when "+
			"the transmitMessage failed to unmarshal: %+v", err)
	}
}

// Error path: MAC fails to verify.
func TestManager_processTransmission_MacVerifyError(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("partnerID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	cmixMsg, _, _, _, err := m.makeTransmitCmixMessage(partner, []byte{}, "", 6,
		32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to generate expected CMIX message: %+v", err)
	}

	cmixMsg.SetMac(make([]byte, 32))

	fp := singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey())
	_, _, err = m.processTransmission(cmixMsg, fp)
	if !check(err, "failed to verify MAC") {
		t.Errorf("processTransmission() did not produce an error when "+
			"the MAC failed to verify: %+v", err)
	}
}
