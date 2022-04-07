package single

import (
	"bytes"
	pb "gitlab.com/elixxir/comms/mixmessages"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	contact2 "gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Happy path.
func TestManager_GetMaxTransmissionPayloadSize(t *testing.T) {
	m := newTestManager(0, false, t)
	cmixPrimeSize := m.store.Cmix().GetGroup().GetP().ByteLen()
	e2ePrimeSize := m.store.E2e().GetGroup().GetP().ByteLen()
	expectedSize := 2*cmixPrimeSize - e2ePrimeSize - format.KeyFPLen - format.MacLen - format.RecipientIDLen - transmitPlMinSize - transmitMessageVersionSize - 1
	testSize := m.GetMaxTransmissionPayloadSize()

	if expectedSize != testSize {
		t.Errorf("GetMaxTransmissionPayloadSize() failed to return the expected size."+
			"\nexpected: %d\nreceived: %d", expectedSize, testSize)
	}
}

// Happy path.
func TestManager_transmitSingleUse(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := rand.New(rand.NewSource(42))
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	payload := make([]byte, 95)
	rand.New(rand.NewSource(42)).Read(payload)
	tag := "testTag"
	maxMsgs := uint8(8)
	callback, callbackChan := createReplyComm()
	timeout := 15 * time.Millisecond

	err := m.transmitSingleUse(partner, payload, tag, maxMsgs, prng, callback, timeout)
	if err != nil {
		t.Errorf("transmitSingleUse() returned an error: %+v", err)
	}

	for _, state := range m.p.singleUse {
		state.quitChan <- struct{}{}
	}

	expectedMsg, _, _, _, err := m.makeTransmitCmixMessage(partner, payload,
		tag, maxMsgs, 32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to make expected message: %+v", err)
	}

	if !reflect.DeepEqual(expectedMsg, m.net.(*testNetworkManager).GetMsg(0)) {
		t.Errorf("transmitSingleUse() failed to send the correct CMIX message."+
			"\nexpected: %+v\nreceived: %+v",
			expectedMsg, m.net.(*testNetworkManager).GetMsg(0))
	}

	timer := time.NewTimer(timeout * 2)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the thread should have quit."+
			"\npayload: %+v\nerror:   %+v", results.payload, results.err)
	case <-timer.C:
	}
}

// Error path: function quits early if the timoutHandler quit.
func TestManager_transmitSingleUse_QuitChanError(t *testing.T) {
	m := newTestManager(10*time.Millisecond, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	callback, callbackChan := createReplyComm()
	timeout := 15 * time.Millisecond

	err := m.transmitSingleUse(partner, []byte{}, "testTag", 9,
		rand.New(rand.NewSource(42)), callback, timeout)
	if err != nil {
		t.Errorf("transmitSingleUse() returned an error: %+v", err)
	}

	for _, state := range m.p.singleUse {
		state.quitChan <- struct{}{}
	}

	timer := time.NewTimer(2 * timeout)

	select {
	case results := <-callbackChan:
		if results.payload != nil || results.err != nil {
			t.Errorf("Callback called when the timeout thread should have quit."+
				"\npayload: %+v\nerror:   %+v", results.payload, results.err)
		}
	case <-timer.C:
	}
}

// Error path: fails to add a new identity.
func TestManager_transmitSingleUse_AddIdentityError(t *testing.T) {
	timeout := 15 * time.Millisecond
	m := newTestManager(timeout, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	callback, callbackChan := createReplyComm()

	err := m.transmitSingleUse(partner, []byte{}, "testTag", 9,
		rand.New(rand.NewSource(42)), callback, timeout)
	if err != nil {
		t.Errorf("transmitSingleUse() returned an error: %+v", err)
	}

	for _, state := range m.p.singleUse {
		state.quitChan <- struct{}{}
	}

	timer := time.NewTimer(2 * timeout)

	select {
	case results := <-callbackChan:
		if results.payload != nil || !check(results.err, "Failed to add new identity") {
			t.Errorf("Callback did not return the correct error when the "+
				"routine quit early.\npayload: %+v\nerror:   %+v",
				results.payload, results.err)
		}
	case <-timer.C:
	}
}

// Error path: Send fails to send message.
func TestManager_transmitSingleUse_SendCMIXError(t *testing.T) {
	m := newTestManager(0, true, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	callback, callbackChan := createReplyComm()
	timeout := 15 * time.Millisecond

	err := m.transmitSingleUse(partner, []byte{}, "testTag", 9,
		rand.New(rand.NewSource(42)), callback, timeout)
	if err != nil {
		t.Errorf("transmitSingleUse() returned an error: %+v", err)
	}

	timer := time.NewTimer(timeout * 2)

	select {
	case results := <-callbackChan:
		if results.payload != nil || !check(results.err, "failed to send single-use transmission CMIX message") {
			t.Errorf("Callback did not return the correct error when the "+
				"routine quit early.\npayload: %+v\nerror:   %+v",
				results.payload, results.err)
		}
	case <-timer.C:
	}
}

// Error path: failed to create CMIX message because the payload is too large.
func TestManager_transmitSingleUse_MakeTransmitCmixMessageError(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, m.store.Cmix().GetGroup().GetP().ByteLen())

	err := m.transmitSingleUse(contact2.Contact{}, payload, "", 0, prng, nil, 0)
	if err == nil {
		t.Error("transmitSingleUse() did not return an error when the payload " +
			"is too large.")
	}
}

// Error path: failed to add pending state because is already exists.
func TestManager_transmitSingleUse_AddStateError(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	payload := make([]byte, 95)
	rand.New(rand.NewSource(42)).Read(payload)
	tag := "testTag"
	maxMsgs := uint8(8)
	callback, _ := createReplyComm()
	timeout := 15 * time.Millisecond

	// Create new CMIX and add a state
	_, dhKey, rid, _, err := m.makeTransmitCmixMessage(partner, payload, tag,
		maxMsgs, 32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create new CMIX message: %+v", err)
	}
	m.p.singleUse[*rid] = newState(dhKey, maxMsgs, nil)

	err = m.transmitSingleUse(partner, payload, tag, maxMsgs,
		rand.New(rand.NewSource(42)), callback, timeout)
	if !check(err, "failed to add pending state") {
		t.Errorf("transmitSingleUse() failed to error when on adding state "+
			"when the state already exists: %+v", err)
	}
}

// Error path: timeout occurs on tracking results of round.
func TestManager_transmitSingleUse_RoundTimeoutError(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := rand.New(rand.NewSource(42))
	partner := contact2.Contact{
		ID:       id.NewIdFromString("Contact ID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(5),
	}
	payload := make([]byte, 95)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReplyComm()
	timeout := 15 * time.Millisecond

	err := m.transmitSingleUse(partner, payload, "testTag", 8, prng, callback, timeout)
	if err != nil {
		t.Errorf("transmitSingleUse() returned an error: %+v", err)
	}

	timer := time.NewTimer(timeout * 2)

	select {
	case results := <-callbackChan:
		if results.payload != nil || !check(results.err, "timed out") {
			t.Errorf("Callback did not return the correct error when it "+
				"should have timed out.\npayload: %+v\nerror:   %+v",
				results.payload, results.err)
		}
	case <-timer.C:
	}
}

// Happy path
func TestManager_makeTransmitCmixMessage(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := rand.New(rand.NewSource(42))
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(42),
	}
	tag := "Test tag"
	payload := make([]byte, 130)
	rand.New(rand.NewSource(42)).Read(payload)
	maxMsgs := uint8(8)
	timeNow := netTime.Now()

	msg, dhKey, rid, _, err := m.makeTransmitCmixMessage(partner, payload,
		tag, maxMsgs, 32, 30*time.Second, timeNow, prng)

	if err != nil {
		t.Errorf("makeTransmitCmixMessage() produced an error: %+v", err)
	}

	fp := singleUse.NewTransmitFingerprint(partner.DhPubKey)
	key := singleUse.NewTransmitKey(dhKey)

	encPayload, err := unmarshalTransmitMessage(msg.GetContents(),
		m.store.E2e().GetGroup().GetP().ByteLen())
	if err != nil {
		t.Errorf("Failed to unmarshal contents: %+v", err)
	}

	decryptedPayload, err := unmarshalTransmitMessagePayload(auth.Crypt(key,
		fp[:24], encPayload.GetPayload()))
	if err != nil {
		t.Errorf("Failed to unmarshal payload: %+v", err)
	}

	if !bytes.Equal(payload, decryptedPayload.GetContents()) {
		t.Errorf("Failed to decrypt payload.\nexpected: %+v\nreceived: %+v",
			payload, decryptedPayload.GetContents())
	}

	if !singleUse.VerifyMAC(key, encPayload.GetPayload(), msg.GetMac()) {
		t.Error("Failed to verify the message MAC.")
	}

	if fp != msg.GetKeyFP() {
		t.Errorf("Failed to verify the CMIX message fingperprint."+
			"\nexpected: %s\nreceived: %s", fp, msg.GetKeyFP())
	}

	if maxMsgs != decryptedPayload.GetMaxParts() {
		t.Errorf("Incorrect maxMsgs.\nexpected: %d\nreceived: %d",
			maxMsgs, decryptedPayload.GetMaxParts())
	}

	expectedTagFP := singleUse.NewTagFP(tag)
	if decryptedPayload.GetTagFP() != expectedTagFP {
		t.Errorf("Incorrect TagFP.\nexpected: %s\nreceived: %s",
			expectedTagFP, decryptedPayload.GetTagFP())
	}

	if !rid.Cmp(decryptedPayload.GetRID(encPayload.GetPubKey(m.store.E2e().GetGroup()))) {
		t.Errorf("Returned incorrect recipient ID.\nexpected: %s\nreceived: %s",
			decryptedPayload.GetRID(encPayload.GetPubKey(m.store.E2e().GetGroup())), rid)
	}
}

// Error path: supplied payload to large for message.
func TestManager_makeTransmitCmixMessage_PayloadTooLargeError(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, 1000)
	rand.New(rand.NewSource(42)).Read(payload)

	_, _, _, _, err := m.makeTransmitCmixMessage(contact2.Contact{}, payload, "", 8, 32,
		30*time.Second, netTime.Now(), prng)

	if !check(err, "too long for message payload capacity") {
		t.Errorf("makeTransmitCmixMessage() failed to error when the payload is too "+
			"large: %+v", err)
	}
}

// Error path: key generation fails.
func TestManager_makeTransmitCmixMessage_KeyGenerationError(t *testing.T) {
	m := newTestManager(0, false, t)
	prng := strings.NewReader("a")
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetGroup().NewInt(42),
	}

	_, _, _, _, err := m.makeTransmitCmixMessage(partner, nil, "", 8, 32,
		30*time.Second, netTime.Now(), prng)

	if !check(err, "failed to generate key in group") {
		t.Errorf("makeTransmitCmixMessage() failed to error when key "+
			"generation failed: %+v", err)
	}
}

// Happy path: test for consistency.
func Test_makeIDs_Consistency(t *testing.T) {
	m := newTestManager(0, false, t)
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	transmitMsg := newTransmitMessage(cmixMsg.ContentsSize(), m.store.E2e().GetGroup().GetP().ByteLen())
	msgPayload := newTransmitMessagePayload(transmitMsg.GetPayloadSize())
	msgPayload.SetTagFP(singleUse.NewTagFP("tag"))
	msgPayload.SetMaxParts(8)
	msgPayload.SetContents([]byte("payload"))
	_, publicKey, err := generateDhKeys(m.store.E2e().GetGroup(),
		m.store.E2e().GetGroup().NewInt(42), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to generate public key: %+v", err)
	}
	addressSize := uint8(32)

	expectedPayload, err := unmarshalTransmitMessagePayload(msgPayload.Marshal())
	if err != nil {
		t.Fatalf("Failed to copy payload: %+v", err)
	}

	err = expectedPayload.SetNonce(rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to set nonce: %+v", err)
	}

	timeNow := netTime.Now()

	rid, ephID, err := makeIDs(&msgPayload, publicKey, addressSize,
		30*time.Second, timeNow, rand.New(rand.NewSource(42)))
	if err != nil {
		t.Errorf("makeIDs() returned an error: %+v", err)
	}

	if expectedPayload.GetNonce() != msgPayload.GetNonce() {
		t.Errorf("makeIDs() failed to set the expected nonce."+
			"\nexpected: %d\nreceived: %d", expectedPayload.GetNonce(), msgPayload.GetNonce())
	}

	if !expectedPayload.GetRID(publicKey).Cmp(rid) {
		t.Errorf("makeIDs() did not return the expected ID."+
			"\nexpected: %s\nreceived: %s", expectedPayload.GetRID(publicKey), rid)
	}

	expectedEphID, _, _, err := ephemeral.GetId(expectedPayload.GetRID(publicKey),
		uint(addressSize), timeNow.UnixNano())
	if err != nil {
		t.Fatalf("Failed to generate expected address ID: %+v", err)
	}

	if expectedEphID != ephID {
		t.Errorf("makeIDs() did not return the expected address ID."+
			"\nexpected: %d\nreceived: %d", expectedEphID.Int64(), ephID.Int64())
	}
}

// Error path: failed to generate nonce.
func Test_makeIDs_NonceError(t *testing.T) {
	msgPayload := newTransmitMessagePayload(transmitPlMinSize)

	_, _, err := makeIDs(&msgPayload, &cyclic.Int{}, 32, 30*time.Second,
		netTime.Now(), strings.NewReader(""))
	if !check(err, "failed to generate nonce") {
		t.Errorf("makeIDs() did not return an error when failing to make nonce: %+v", err)
	}
}

type testRoundEvents struct {
	callbacks    map[id.Round][states.NUM_STATES]map[*ds.EventCallback]*ds.EventCallback
	timeoutError bool
	mux          sync.RWMutex
}

func newTestRoundEvents(timeoutError bool) *testRoundEvents {
	return &testRoundEvents{
		callbacks:    make(map[id.Round][states.NUM_STATES]map[*ds.EventCallback]*ds.EventCallback),
		timeoutError: timeoutError,
	}
}

func (r *testRoundEvents) AddRoundEventChan(_ id.Round,
	eventChan chan ds.EventReturn, _ time.Duration, _ ...states.Round) *ds.EventCallback {

	eventChan <- struct {
		RoundInfo *pb.RoundInfo
		TimedOut  bool
	}{
		RoundInfo: &pb.RoundInfo{State: uint32(states.COMPLETED)},
		TimedOut:  r.timeoutError,
	}

	return nil
}
