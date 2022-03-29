///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partner

import (
	"bytes"
	"encoding/base64"
	"fmt"
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/ratchet/session"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"reflect"
	"testing"
)

// Tests happy path of newManager.
func Test_newManager(t *testing.T) {
	// Set up expected and test values
	s, ctx := session.makeTestSession()
	kv := versioned.NewKV(make(ekv.Memstore))
	partnerID := id.NewIdFromUInt(100, id.User, t)
	expectedM := &Manager{
		ctx:                     ctx,
		kv:                      kv.Prefix(fmt.Sprintf(managerPrefix, partnerID)),
		partner:                 partnerID,
		originPartnerPubKey:     s.partnerPubKey,
		originMyPrivKey:         s.myPrivKey,
		originPartnerSIDHPubKey: s.partnerSIDHPubKey,
		originMySIDHPrivKey:     s.mySIDHPrivKey,
	}
	expectedM.send = NewRelationship(expectedM, session2.Send,
		params.GetDefaultE2ESessionParams())
	expectedM.receive = NewRelationship(expectedM, session2.Receive,
		params.GetDefaultE2ESessionParams())

	// Create new relationship
	m := newManager(ctx, kv, partnerID, s.myPrivKey, s.partnerPubKey,
		s.mySIDHPrivKey, s.partnerSIDHPubKey,
		s.e2eParams,
		s.e2eParams)

	// Check if the new relationship matches the expected
	if !managersEqual(expectedM, m, t) {
		t.Errorf("newManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, m)
	}
}

// Tests happy path of LoadManager.
func TestLoadManager(t *testing.T) {
	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	// Attempt to load relationship
	m, err := LoadManager(expectedM.ctx, kv, expectedM.partner)
	if err != nil {
		t.Errorf("LoadManager() returned an error: %v", err)
	}

	// Check if the loaded relationship matches the expected
	if !managersEqual(expectedM, m, t) {
		t.Errorf("LoadManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, m)
	}
}

// Unit test for clearManager
func TestManager_ClearManager(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("clearManager error: " +
				"Did not panic when loading deleted manager")
		}
	}()

	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	err := clearManager(expectedM, kv)
	if err != nil {
		t.Fatalf("clearManager returned an error: %v", err)
	}

	// Attempt to load relationship
	_, err = LoadManager(expectedM.ctx, kv, expectedM.partner)
	if err != nil {
		t.Errorf("LoadManager() returned an error: %v", err)
	}
}

// Tests happy path of Manager.NewReceiveSession.
func TestManager_NewReceiveSession(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	s, _ := session.makeTestSession()

	se, exists := m.NewReceiveSession(s.partnerPubKey, s.partnerSIDHPubKey,
		s.e2eParams, s)
	if exists {
		t.Errorf("NewReceiveSession() incorrect return value."+
			"\n\texpected: %v\n\treceived: %v", false, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(s.GetID().Marshal(),
		se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() incorrect session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), s.GetID(), se.GetID())
	}

	se, exists = m.NewReceiveSession(s.partnerPubKey, s.partnerSIDHPubKey,
		s.e2eParams, s)
	if !exists {
		t.Errorf("NewReceiveSession() incorrect return value."+
			"\n\texpected: %v\n\treceived: %v", true, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(s.GetID().Marshal(),
		se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() incorrect session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), s.GetID(), se.GetID())
	}
}

// Tests happy path of Manager.NewSendSession.
func TestManager_NewSendSession(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	s, _ := session.makeTestSession()

	se := m.NewSendSession(s.myPrivKey, s.mySIDHPrivKey, s.e2eParams)
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}

	se, _ = m.NewReceiveSession(s.partnerPubKey, s.partnerSIDHPubKey,
		s.e2eParams, s)
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}
}

// Tests happy path of Manager.GetKeyForSending.
func TestManager_GetKeyForSending(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	p := params.GetDefaultE2E()
	expectedKey := &session2.Cypher{
		session: m.send.sessions[0],
	}

	key, err := m.GetKeyForSending(p.Type)
	if err != nil {
		t.Errorf("GetKeyForSending() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedKey, key) {
		t.Errorf("GetKeyForSending() did not return the correct key."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedKey, key)
	}

	p.Type = params.KeyExchange
	m.send.sessions[0].negotiationStatus = session2.NewSessionTriggered
	expectedKey.keyNum++

	key, err = m.GetKeyForSending(p.Type)
	if err != nil {
		t.Errorf("GetKeyForSending() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedKey, key) {
		t.Errorf("GetKeyForSending() did not return the correct key."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedKey, key)
	}
}

// Tests that Manager.GetKeyForSending returns an error for invalid SendType.
func TestManager_GetKeyForSending_Error(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	p := params.GetDefaultE2E()
	p.Type = 2

	key, err := m.GetKeyForSending(p.Type)
	if err == nil {
		t.Errorf("GetKeyForSending() did not produce an error for invalid SendType.")
	}

	if key != nil {
		t.Errorf("GetKeyForSending() did not return the correct key."+
			"\n\texpected: %+v\n\treceived: %+v",
			nil, key)
	}
}

// Tests happy path of Manager.GetPartnerID.
func TestManager_GetPartnerID(t *testing.T) {
	m, _ := newTestManager(t)

	pid := m.GetPartnerID()

	if !m.partner.Cmp(pid) {
		t.Errorf("GetPartnerID() returned incorrect partner ID."+
			"\n\texpected: %s\n\treceived: %s", m.partner, pid)
	}
}

// Tests happy path of Manager.GetSendSession.
func TestManager_GetSendSession(t *testing.T) {
	m, _ := newTestManager(t)

	s := m.GetSendSession(m.send.sessions[0].GetID())

	if !reflect.DeepEqual(m.send.sessions[0], s) {
		t.Errorf("GetSendSession() returned incorrect session."+
			"\n\texpected: %s\n\treceived: %s", m.send.sessions[0], s)
	}
}

// Tests happy path of Manager.GetReceiveSession.
func TestManager_GetReceiveSession(t *testing.T) {
	m, _ := newTestManager(t)

	s := m.GetReceiveSession(m.receive.sessions[0].GetID())

	if !reflect.DeepEqual(m.receive.sessions[0], s) {
		t.Errorf("GetReceiveSession() returned incorrect session."+
			"\n\texpected: %s\n\treceived: %s", m.receive.sessions[0], s)
	}
}

// Tests happy path of Manager.Confirm.
func TestManager_Confirm(t *testing.T) {
	m, _ := newTestManager(t)
	m.send.sessions[0].negotiationStatus = session2.Sent
	err := m.Confirm(m.send.sessions[0].GetID())
	if err != nil {
		t.Errorf("Confirm produced an error: %v", err)
	}
}

// Tests happy path of Manager.TriggerNegotiations.
func TestManager_TriggerNegotiations(t *testing.T) {
	m, _ := newTestManager(t)
	m.send.sessions[0].negotiationStatus = session2.Unconfirmed
	sessions := m.TriggerNegotiations()
	if !reflect.DeepEqual(m.send.sessions, sessions) {
		t.Errorf("TriggerNegotiations() returned incorrect sessions."+
			"\n\texpected: %s\n\treceived: %s", m.send.sessions, sessions)
	}
}

// newTestManager returns a new relationship for testing.
func newTestManager(t *testing.T) (*Manager, *versioned.KV) {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	s, ctx := session.makeTestSession()
	kv := versioned.NewKV(make(ekv.Memstore))
	partnerID := id.NewIdFromUInts([4]uint64{prng.Uint64(), prng.Uint64(),
		prng.Uint64(), prng.Uint64()}, id.User, t)

	// Create new relationship
	m := newManager(ctx, kv, partnerID, s.myPrivKey, s.partnerPubKey,
		s.mySIDHPrivKey, s.partnerSIDHPubKey,
		s.e2eParams,
		s.e2eParams)

	return m, kv
}

func managersEqual(expected, received *Manager, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.ctx, received.ctx) {
		t.Errorf("Did not Receive expected Manager.ctx."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.ctx, received.ctx)
		equal = false
	}
	if !reflect.DeepEqual(expected.kv, received.kv) {
		t.Errorf("Did not Receive expected Manager.kv."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.kv, received.kv)
		equal = false
	}
	if !expected.partner.Cmp(received.partner) {
		t.Errorf("Did not Receive expected Manager.partner."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.partner, received.partner)
		equal = false
	}
	if !relationshipsEqual(expected.receive, received.receive) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.receive, received.receive)
		equal = false
	}
	if !relationshipsEqual(expected.send, received.send) {
		t.Errorf("Did not Receive expected Manager.Send."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.send, received.send)
		equal = false
	}

	return equal
}

// Unit test of Manager.GetRelationshipFingerprint.
func TestManager_GetRelationshipFingerprint(t *testing.T) {
	m, _ := newTestManager(t)
	m.receive.fingerprint = []byte{5}
	m.send.fingerprint = []byte{10}
	h, _ := blake2b.New256(nil)
	h.Write(append(m.receive.fingerprint, m.send.fingerprint...))
	expected := base64.StdEncoding.EncodeToString(h.Sum(nil))[:relationshipFpLength]

	fp := m.GetRelationshipFingerprint()
	if fp != expected {
		t.Errorf("GetRelationshipFingerprint did not return the expected "+
			"fingerprint.\nexpected: %s\nreceived: %s", expected, fp)
	}

	// Flip the order and show that the output is the same.
	m.receive.fingerprint, m.send.fingerprint = m.send.fingerprint, m.receive.fingerprint

	fp = m.GetRelationshipFingerprint()
	if fp != expected {
		t.Errorf("GetRelationshipFingerprint did not return the expected "+
			"fingerprint.\nexpected: %s\nreceived: %s", expected, fp)
	}
}

// Tests the consistency of the output of Manager.GetRelationshipFingerprint.
func TestManager_GetRelationshipFingerprint_Consistency(t *testing.T) {
	m, _ := newTestManager(t)
	prng := rand.New(rand.NewSource(42))
	expectedFps := []string{
		"GmeTCfxGOqRqeID", "gbpJjHd3tIe8BKy", "2/ZdG+WNzODJBiF",
		"+V1ySeDLQfQNSkv", "23OMC+rBmCk+gsu", "qHu5MUVs83oMqy8",
		"kuXqxsezI0kS9Bc", "SlEhsoZ4BzAMTtr", "yG8m6SPQfV/sbTR",
		"j01ZSSm762TH7mj", "SKFDbFvsPcohKPw", "6JB5HK8DHGwS4uX",
		"dU3mS1ujduGD+VY", "BDXAy3trbs8P4mu", "I4HoXW45EwWR0oD",
		"661YH2l2jfOkHbA", "cSS9ZyTOQKVx67a", "ojfubzDIsMNYc/t",
		"2WrEw83Yz6Rhq9I", "TQILxBIUWMiQS2j", "rEqdieDTXJfCQ6I",
	}

	for i, expected := range expectedFps {
		prng.Read(m.receive.fingerprint)
		prng.Read(m.send.fingerprint)

		fp := m.GetRelationshipFingerprint()
		if fp != expected {
			t.Errorf("GetRelationshipFingerprint did not return the expected "+
				"fingerprint (%d).\nexpected: %s\nreceived: %s", i, expected, fp)
		}

		// Flip the order and show that the output is the same.
		m.receive.fingerprint, m.send.fingerprint = m.send.fingerprint, m.receive.fingerprint

		fp = m.GetRelationshipFingerprint()
		if fp != expected {
			t.Errorf("GetRelationshipFingerprint did not return the expected "+
				"fingerprint (%d).\nexpected: %s\nreceived: %s", i, expected, fp)
		}

		// fmt.Printf("\"%s\",\n", fp) // Uncomment to reprint expected values
	}
}
