///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Happy path.
func TestManager_ReceiveResponseHandler(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	partner := NewContact(id.NewIdFromString("recipientID", id.User, t),
		m.store.E2e().GetGroup().NewInt(43), m.store.E2e().GetGroup().NewInt(42),
		singleUse.TagFP{}, 8)
	ephID, _, _, err := ephemeral.GetId(partner.partner, id.ArrIDLen, netTime.Now().UnixNano())
	payload := make([]byte, 2000)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReplyComm()
	rid := id.NewIdFromString("rid", id.User, t)

	m.p.singleUse[*rid] = newState(partner.dhKey, partner.maxParts, callback)

	msgs, err := m.makeReplyCmixMessages(partner, payload)
	if err != nil {
		t.Fatalf("Failed to generate CMIX messages: %+v", err)
	}

	go func() {
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-timer.C:
			t.Errorf("quitChan never set.")
		case <-m.p.singleUse[*rid].quitChan:
		}
	}()

	go m.receiveResponseHandler(rawMessages, stop)

	for _, msg := range msgs {
		rawMessages <- message.Receive{
			Payload:     msg.Marshal(),
			Sender:      partner.partner,
			RecipientID: rid,
			EphemeralID: ephID,
		}
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		if !bytes.Equal(results.payload, payload) {
			t.Errorf("Callback received wrong payload."+
				"\nexpected: %+v\nreceived: %+v", payload, results.payload)
		}
		if results.err != nil {
			t.Errorf("Callback received an error: %+v", results.err)
		}
	case <-timer.C:
		t.Errorf("Callback failed to be called.")
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
}

// Error path: invalid CMIX message.
func TestManager_ReceiveResponseHandler_CmixMessageError(t *testing.T) {
	m := newTestManager(0, false, t)
	rawMessages := make(chan message.Receive, rawMessageBuffSize)
	stop := stoppable.NewSingle("singleStoppable")
	partner := NewContact(id.NewIdFromString("recipientID", id.User, t),
		m.store.E2e().GetGroup().NewInt(43), m.store.E2e().GetGroup().NewInt(42),
		singleUse.TagFP{}, 8)
	ephID, _, _, _ := ephemeral.GetId(partner.partner, id.ArrIDLen, netTime.Now().UnixNano())
	payload := make([]byte, 2000)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReplyComm()
	rid := id.NewIdFromString("rid", id.User, t)

	m.p.singleUse[*rid] = newState(partner.dhKey, partner.maxParts, callback)

	go func() {
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-timer.C:
		case <-m.p.singleUse[*rid].quitChan:
			t.Error("quitChan called on error.")
		}
	}()

	go m.receiveResponseHandler(rawMessages, stop)

	rawMessages <- message.Receive{
		Payload:     make([]byte, format.MinimumPrimeSize*2),
		Sender:      partner.partner,
		RecipientID: rid,
		EphemeralID: ephID,
	}

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when it should not have been."+
			"payload: %+v\nerror: %+v", results.payload, results.err)
	case <-timer.C:
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
}

// Happy path.
func TestManager_processesResponse(t *testing.T) {
	m := newTestManager(0, false, t)
	rid := id.NewIdFromString("test RID", id.User, t)
	ephID, _, _, err := ephemeral.GetId(rid, id.ArrIDLen, netTime.Now().UnixNano())
	if err != nil {
		t.Fatalf("Failed to create address ID: %+v", err)
	}
	dhKey := getGroup().NewInt(5)
	maxMsgs := uint8(6)
	timeout := 5 * time.Millisecond
	callback, callbackChan := createReplyComm()

	m.p.singleUse[*rid] = newState(dhKey, maxMsgs, callback)

	expectedData := []string{"This i", "s the ", "expect", "ed fin", "al dat", "a."}
	var msgs []format.Message
	for fp, i := range m.p.singleUse[*rid].fpMap.fps {
		newMsg := format.NewMessage(format.MinimumPrimeSize)
		part := newResponseMessagePart(newMsg.ContentsSize())
		part.SetContents([]byte(expectedData[i]))
		part.SetPartNum(uint8(i))
		part.SetMaxParts(maxMsgs)

		key := singleUse.NewResponseKey(dhKey, i)
		encryptedPayload := auth.Crypt(key, fp[:24], part.Marshal())

		newMsg.SetKeyFP(fp)
		newMsg.SetMac(singleUse.MakeMAC(key, encryptedPayload))
		newMsg.SetContents(encryptedPayload)
		msgs = append(msgs, newMsg)
	}

	go func() {
		timer := time.NewTimer(timeout)
		select {
		case <-timer.C:
			t.Errorf("quitChan never set.")
		case <-m.p.singleUse[*rid].quitChan:
		}
	}()

	for i, msg := range msgs {
		err := m.processesResponse(rid, ephID, msg.Marshal())
		if err != nil {
			t.Errorf("processesResponse() returned an error (%d): %+v", i, err)
		}
	}

	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		t.Errorf("Callback never called.")
	case results := <-callbackChan:
		if results.err != nil {
			t.Errorf("Callback returned an error: %+v", err)
		}
		if !bytes.Equal([]byte(strings.Join(expectedData, "")), results.payload) {
			t.Errorf("Callback returned incorrect data."+
				"\nexpected: %s\nreceived: %s", expectedData, results.payload)
		}
	}

	if _, exists := m.p.singleUse[*rid]; exists {
		t.Error("Failed to delete the state after collation is complete.")
	}
}

// Error path: no state in map.
func TestManager_processesResponse_NoStateError(t *testing.T) {
	m := newTestManager(0, false, t)
	rid := id.NewIdFromString("test RID", id.User, t)

	err := m.processesResponse(rid, ephemeral.Id{}, []byte{})
	if !check(err, "no state exists for the reception ID") {
		t.Errorf("processesResponse() did not return an error when the state "+
			"is not in the map: %+v", err)
	}
}

// Error path: failed to verify MAC.
func TestManager_processesResponse_MacVerificationError(t *testing.T) {
	m := newTestManager(0, false, t)
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	timeout := 5 * time.Millisecond
	callback := func(payload []byte, err error) {}

	quitChan, _, err := m.p.addState(rid, dhKey, 1, callback, timeout)
	if err != nil {
		t.Fatalf("Failed to add state: %+v", err)
	}
	quitChan <- struct{}{}

	newMsg := format.NewMessage(format.MinimumPrimeSize)
	part := newResponseMessagePart(newMsg.ContentsSize())
	part.SetContents([]byte("payload data"))
	part.SetPartNum(0)
	part.SetMaxParts(1)
	newMsg.SetMac(singleUse.MakeMAC(dhKey.Bytes(), []byte("some data")))
	newMsg.SetContents([]byte("payload data"))

	var fp format.Fingerprint
	for fpt, i := range m.p.singleUse[*rid].fpMap.fps {
		if i == 0 {
			fp = fpt
		}
	}
	newMsg.SetKeyFP(fp)

	err = m.processesResponse(rid, ephemeral.Id{}, newMsg.Marshal())
	if !check(err, "MAC") {
		t.Errorf("processesResponse() did not return an error when MAC "+
			"verification should have failed: %+v", err)
	}
}

// Error path: CMIX message fingerprint does not match fingerprints in map.
func TestManager_processesResponse_FingerprintError(t *testing.T) {
	m := newTestManager(0, false, t)
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	timeout := 5 * time.Millisecond
	callback := func(payload []byte, err error) {}

	quitChan, _, err := m.p.addState(rid, dhKey, 1, callback, timeout)
	if err != nil {
		t.Fatalf("Failed to add state: %+v", err)
	}
	quitChan <- struct{}{}

	var fp format.Fingerprint
	for fpt, i := range m.p.singleUse[*rid].fpMap.fps {
		if i == 0 {
			fp = fpt
		}
	}
	newMsg := format.NewMessage(format.MinimumPrimeSize)
	part := newResponseMessagePart(newMsg.ContentsSize())
	part.SetContents([]byte("payload data"))
	part.SetPartNum(0)
	part.SetMaxParts(1)

	key := singleUse.NewResponseKey(dhKey, 0)
	encryptedPayload := auth.Crypt(key, fp[:24], part.Marshal())

	newMsg.SetKeyFP(format.NewFingerprint([]byte("Invalid Fingerprint")))
	newMsg.SetMac(singleUse.MakeMAC(key, encryptedPayload))
	newMsg.SetContents(encryptedPayload)

	err = m.processesResponse(rid, ephemeral.Id{}, newMsg.Marshal())
	if !check(err, "fingerprint") {
		t.Errorf("processesResponse() did not return an error when "+
			"fingerprint was wrong: %+v", err)
	}
}

// Error path: collator fails because part number is wrong.
func TestManager_processesResponse_CollatorError(t *testing.T) {
	m := newTestManager(0, false, t)
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	timeout := 5 * time.Millisecond
	callback := func(payload []byte, err error) {}

	quitChan, _, err := m.p.addState(rid, dhKey, 1, callback, timeout)
	if err != nil {
		t.Fatalf("Failed to add state: %+v", err)
	}
	quitChan <- struct{}{}

	var fp format.Fingerprint
	for fpt, i := range m.p.singleUse[*rid].fpMap.fps {
		if i == 0 {
			fp = fpt
		}
	}
	newMsg := format.NewMessage(format.MinimumPrimeSize)
	part := newResponseMessagePart(newMsg.ContentsSize())
	part.SetContents([]byte("payload data"))
	part.SetPartNum(5)
	part.SetMaxParts(1)

	key := singleUse.NewResponseKey(dhKey, 0)
	encryptedPayload := auth.Crypt(key, fp[:24], part.Marshal())

	newMsg.SetKeyFP(fp)
	newMsg.SetMac(singleUse.MakeMAC(key, encryptedPayload))
	newMsg.SetContents(encryptedPayload)

	err = m.processesResponse(rid, ephemeral.Id{}, newMsg.Marshal())
	if !check(err, "collate") {
		t.Errorf("processesResponse() did not return an error when "+
			"collation should have failed: %+v", err)
	}
}
