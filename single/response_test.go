///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Happy path.
func TestManager_GetMaxResponsePayloadSize(t *testing.T) {
	m := newTestManager(0, false, t)
	cmixPrimeSize := m.store.Cmix().GetGroup().GetP().ByteLen()
	expectedSize := 2*cmixPrimeSize - format.KeyFPLen - format.MacLen - format.RecipientIDLen - responseMinSize - 1
	testSize := m.GetMaxResponsePayloadSize()

	if expectedSize != testSize {
		t.Errorf("GetMaxResponsePayloadSize() failed to return the expected size."+
			"\nexpected: %d\nreceived: %d", expectedSize, testSize)
	}
}

// Happy path.
func TestManager_respondSingleUse(t *testing.T) {
	m := newTestManager(0, false, t)
	used := int32(0)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      10,
		used:          &used,
	}
	payload := make([]byte, 400*int(partner.maxParts))
	rand.New(rand.NewSource(42)).Read(payload)
	re := newTestRoundEvents(false)

	expectedMsgs, err := m.makeReplyCmixMessages(partner, payload)
	if err != nil {
		t.Fatalf("Failed to created expected messages: %+v", err)
	}

	err = m.respondSingleUse(partner, payload, 10*time.Millisecond, re)
	if err != nil {
		t.Errorf("respondSingleUse() produced an error: %+v", err)
	}

	// Check that all messages are expected and received
	if len(m.net.(*testNetworkManager).msgs) != int(partner.GetMaxParts()) {
		t.Errorf("Recieved incorrect number of messages."+
			"\nexpected: %d\nreceived: %d", int(partner.GetMaxParts()),
			len(m.net.(*testNetworkManager).msgs))
	}

	// Check that all received messages were expected
	var exists bool
	for _, received := range m.net.(*testNetworkManager).msgs {
		exists = false
		for _, msg := range expectedMsgs {
			if reflect.DeepEqual(msg, received) {
				exists = true
			}
		}
		if !exists {
			t.Errorf("Unexpected message: %+v", received)
		}
	}
}

// Error path: response has already been sent.
func TestManager_respondSingleUse_ResponseUsedError(t *testing.T) {
	m := newTestManager(0, false, t)
	used := int32(1)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      10,
		used:          &used,
	}

	err := m.respondSingleUse(partner, []byte{}, 10*time.Millisecond, newTestRoundEvents(false))
	if !check(err, "cannot send to single-use contact that has already been sent to") {
		t.Errorf("respondSingleUse() did not produce the expected error when "+
			"the contact has been used: %+v", err)
	}
}

// Error path: cannot create CMIX message when payload is too large.
func TestManager_respondSingleUse_MakeCmixMessageError(t *testing.T) {
	m := newTestManager(0, false, t)
	used := int32(0)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      10,
		used:          &used,
	}
	payload := make([]byte, 500*int(partner.maxParts))
	rand.New(rand.NewSource(42)).Read(payload)

	err := m.respondSingleUse(partner, payload, 10*time.Millisecond, newTestRoundEvents(false))
	if !check(err, "failed to create new CMIX messages") {
		t.Errorf("respondSingleUse() did not produce the expected error when "+
			"the CMIX message creation failed: %+v", err)
	}
}

// Error path: TrackResults returns an error.
func TestManager_respondSingleUse_TrackResultsError(t *testing.T) {
	m := newTestManager(0, false, t)
	used := int32(0)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      10,
		used:          &used,
	}
	payload := make([]byte, 400*int(partner.maxParts))
	rand.New(rand.NewSource(42)).Read(payload)

	err := m.respondSingleUse(partner, payload, 10*time.Millisecond, newTestRoundEvents(true))
	if !check(err, "tracking results of") {
		t.Errorf("respondSingleUse() did not produce the expected error when "+
			"the CMIX message creation failed: %+v", err)
	}
}

// Happy path.
func TestManager_makeReplyCmixMessages(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      255,
	}
	payload := make([]byte, 400*int(partner.maxParts))
	rand.New(rand.NewSource(42)).Read(payload)

	msgs, err := m.makeReplyCmixMessages(partner, payload)
	if err != nil {
		t.Errorf("makeReplyCmixMessages() returned an error: %+v", err)
	}

	buff := bytes.NewBuffer(payload)
	for i, msg := range msgs {
		checkReplyCmixMessage(partner,
			buff.Next(len(msgs[0].GetContents())-responseMinSize), msg, len(msgs), i, t)
	}
}

// Error path: size of payload too large.
func TestManager_makeReplyCmixMessages_PayloadSizeError(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := Contact{
		partner:       id.NewIdFromString("partner ID:", id.User, t),
		partnerPubKey: m.store.E2e().GetGroup().NewInt(42),
		dhKey:         m.store.E2e().GetGroup().NewInt(99),
		tagFP:         singleUse.NewTagFP("tag"),
		maxParts:      255,
	}
	payload := make([]byte, 500*int(partner.maxParts))
	rand.New(rand.NewSource(42)).Read(payload)

	_, err := m.makeReplyCmixMessages(partner, payload)
	if err == nil {
		t.Error("makeReplyCmixMessages() did not return an error when the " +
			"payload is too large.")
	}
}

func checkReplyCmixMessage(c Contact, payload []byte, msg format.Message, maxParts, i int, t *testing.T) {
	expectedFP := singleUse.NewResponseFingerprint(c.dhKey, uint64(i))
	key := singleUse.NewResponseKey(c.dhKey, uint64(i))
	expectedMac := singleUse.MakeMAC(key, msg.GetContents())

	// Check CMIX message
	if expectedFP != msg.GetKeyFP() {
		t.Errorf("CMIX message #%d had incorrect fingerprint."+
			"\nexpected: %s\nrecieved: %s", i, expectedFP, msg.GetKeyFP())
	}

	if !singleUse.VerifyMAC(key, msg.GetContents(), msg.GetMac()) {
		t.Errorf("CMIX message #%d had incorrect MAC."+
			"\nexpected: %+v\nrecieved: %+v", i, expectedMac, msg.GetMac())
	}

	// Decrypt payload
	decryptedPayload := cAuth.Crypt(key, expectedFP[:24], msg.GetContents())
	responseMsg, err := unmarshalResponseMessage(decryptedPayload)
	if err != nil {
		t.Errorf("Failed to unmarshal pay load of CMIX message #%d: %+v", i, err)
	}

	if !bytes.Equal(payload, responseMsg.GetContents()) {
		t.Errorf("Response message #%d had incorrect contents."+
			"\nexpected: %+v\nrecieved: %+v",
			i, payload, responseMsg.GetContents())
	}

	if uint8(maxParts) != responseMsg.GetMaxParts() {
		t.Errorf("Response message #%d had incorrect max parts."+
			"\nexpected: %+v\nrecieved: %+v",
			i, maxParts, responseMsg.GetMaxParts())
	}

	if i != int(responseMsg.GetPartNum()) {
		t.Errorf("Response message #%d had incorrect part number."+
			"\nexpected: %+v\nrecieved: %+v",
			i, i, responseMsg.GetPartNum())
	}
}

// Happy path.
func TestManager_splitPayload(t *testing.T) {
	m := newTestManager(0, false, t)
	maxSize := 5
	maxParts := 10
	payload := []byte("0123456789012345678901234567890123456789012345678901234" +
		"5678901234567890123456789012345678901234567890123456789")
	expectedParts := [][]byte{
		payload[:maxSize],
		payload[maxSize : 2*maxSize],
		payload[2*maxSize : 3*maxSize],
		payload[3*maxSize : 4*maxSize],
		payload[4*maxSize : 5*maxSize],
		payload[5*maxSize : 6*maxSize],
		payload[6*maxSize : 7*maxSize],
		payload[7*maxSize : 8*maxSize],
		payload[8*maxSize : 9*maxSize],
		payload[9*maxSize : 10*maxSize],
	}

	testParts := m.splitPayload(payload, maxSize, maxParts)

	if !reflect.DeepEqual(expectedParts, testParts) {
		t.Errorf("splitPayload() failed to correctly split the payload."+
			"\nexpected: %s\nreceived: %s", expectedParts, testParts)
	}
}
