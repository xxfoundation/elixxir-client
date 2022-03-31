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
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"reflect"
	"testing"
)

// Tests happy path of newManager.
func Test_newManager(t *testing.T) {
	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	// Create new relationship
	m := NewManager(kv, expectedM.myID, expectedM.partner,
		expectedM.originMyPrivKey, expectedM.originPartnerPubKey,
		expectedM.originMySIDHPrivKey,
		expectedM.originPartnerSIDHPubKey, session.GetDefaultE2ESessionParams(),
		session.GetDefaultE2ESessionParams(),
		expectedM.cyHandler, expectedM.grp, expectedM.rng)

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
	m, err := LoadManager(kv, expectedM.myID, expectedM.partner,
		expectedM.cyHandler, expectedM.grp, expectedM.rng)
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

	err := ClearManager(expectedM, kv)
	if err != nil {
		t.Fatalf("clearManager returned an error: %v", err)
	}

	// Attempt to load relationship
	_, err = LoadManager(kv, expectedM.myID, expectedM.partner,
		expectedM.cyHandler, expectedM.grp, expectedM.rng)
	if err != nil {
		t.Errorf("LoadManager() returned an error: %v", err)
	}
}

// Tests happy path of Manager.NewReceiveSession.
func TestManager_NewReceiveSession(t *testing.T) {
	// Set up test values
	m, kv := newTestManager(t)

	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng.GetStream())
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(m.originMyPrivKey, partnerPubKey, grp,
		m.originMySIDHPrivKey, partnerSIDHPubKey)

	partnerID := id.NewIdFromString("newPartner", id.User, t)

	sid := session.GetSessionIDFromBaseKey(baseKey)
	thisSession := session.NewSession(kv, session.Receive, partnerID,
		m.originMyPrivKey, partnerPrivKey, baseKey,
		m.originMySIDHPrivKey, partnerSIDHPubKey,
		sid, []byte(""), session.Sent,
		session.GetDefaultE2ESessionParams(), m.cyHandler, m.grp, m.rng)

	se, exists := m.NewReceiveSession(partnerPubKey, partnerSIDHPubKey,
		session.GetDefaultE2ESessionParams(), thisSession)
	if exists {
		t.Errorf("NewReceiveSession() incorrect return value."+
			"\n\texpected: %v\n\treceived: %v", false, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(thisSession.GetID().Marshal(),
		se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() incorrect session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), thisSession.GetID(), se.GetID())
	}

	se, exists = m.NewReceiveSession(partnerPubKey, partnerSIDHPubKey,
		session.GetDefaultE2ESessionParams(), thisSession)
	if !exists {
		t.Errorf("NewReceiveSession() incorrect return value."+
			"\n\texpected: %v\n\treceived: %v", true, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(thisSession.GetID().Marshal(),
		se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() incorrect session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), thisSession.GetID(), se.GetID())
	}
}

// Tests happy path of Manager.NewSendSession.
func TestManager_NewSendSession(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)

	se := m.NewSendSession(m.originMyPrivKey, m.originMySIDHPrivKey,
		session.GetDefaultE2ESessionParams(), m.send.sessions[0])
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}

	se, _ = m.NewReceiveSession(m.originPartnerPubKey, m.originPartnerSIDHPubKey,
		session.GetDefaultE2ESessionParams(), m.receive.sessions[0])
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}
}

//Tests happy path of Manager.GetKeyForSending.
func TestManager_GetKeyForSending(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)

	key, err := m.PopSendCypher()
	if err != nil {
		t.Errorf("GetKeyForSending() produced an error: %v", err)
	}

	thisSession := m.send.sessions[0]

	// KeyNum isn't exposable on this layer, so make sure the keynum is 0
	// by checking the fingerprint
	expected := e2eCrypto.DeriveKeyFingerprint(thisSession.GetBaseKey(),
		0, thisSession.GetRelationshipFingerprint())

	received := key.Fingerprint()
	if !bytes.Equal(expected.Bytes(), received.Bytes()) {
		t.Errorf("PopSendCypher() did not return the correct key."+
			"\nexpected fingerprint: %+v"+
			"\nreceived fingerprint: %+v",
			expected.String(), received.String())
	}

	thisSession.SetNegotiationStatus(session.NewSessionTriggered)

	key, err = m.PopSendCypher()
	if err != nil {
		t.Errorf("GetKeyForSending() produced an error: %v", err)
	}

	// KeyNum isn't exposable on this layer, so make sure the keynum is 1
	// by checking the fingerprint
	expected = e2eCrypto.DeriveKeyFingerprint(thisSession.GetBaseKey(),
		1, thisSession.GetRelationshipFingerprint())

	received = key.Fingerprint()

	if !reflect.DeepEqual(expected.Bytes(), received.Bytes()) {
		t.Errorf("PopSendCypher() did not return the correct key."+
			"\nexpected fingerprint: %+v"+
			"\nreceived fingerprint: %+v",
			expected.String(), received.String())
	}
}

// Tests that Manager.GetKeyForSending returns an error for invalid SendType.
func TestManager_GetKeyForSending_Error(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)

	// Create a session that will never get popped
	m.send.sessions[0], _ = session.CreateTestSession(5, 5, 5, session.Unconfirmed, t)

	// Try to pop
	key, err := m.PopSendCypher()
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
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng.GetStream())
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)

	baseKey := session.GenerateE2ESessionBaseKey(m.originMyPrivKey, partnerPubKey, grp,
		m.originMySIDHPrivKey, partnerSIDHPubKey)

	sid := session.GetSessionIDFromBaseKey(baseKey)
	m.send.AddSession(m.originMyPrivKey, partnerPrivKey, baseKey,
		m.originMySIDHPrivKey, partnerSIDHPubKey,
		sid,
		session.Sending, session.GetDefaultE2ESessionParams())

	thisSession := m.send.GetByID(sid)
	thisSession.TriggerNegotiation()

	err := m.Confirm(thisSession.GetID())
	if err != nil {
		t.Errorf("Confirm produced an error: %v", err)
	}
}

// Tests happy path of Manager.TriggerNegotiations.
func TestManager_TriggerNegotiations(t *testing.T) {
	m, _ := newTestManager(t)

	for i := 0; i < 100; i++ {
		m.send.sessions[0].PopReKey()

	}

	sessions := m.TriggerNegotiations()
	if !reflect.DeepEqual(m.send.sessions, sessions) {
		t.Errorf("TriggerNegotiations() returned incorrect sessions."+
			"\n\texpected: %s\n\treceived: %s", m.send.sessions, sessions)
	}
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
