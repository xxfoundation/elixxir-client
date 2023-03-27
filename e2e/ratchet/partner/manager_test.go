////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/v4/storage/utility"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Tests happy path of newManager.
func Test_newManager(t *testing.T) {
	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	// Create new relationship
	newM := NewManager(kv, expectedM.myID, expectedM.partner,
		expectedM.originMyPrivKey, expectedM.originPartnerPubKey,
		expectedM.originMySIDHPrivKey,
		expectedM.originPartnerSIDHPubKey, session.GetDefaultParams(),
		session.GetDefaultParams(),
		expectedM.cyHandler, expectedM.grp, expectedM.rng)

	m := newM.(*manager)

	// Check if the new relationship matches the expected
	if !managersEqual(&expectedM, m, t) {
		t.Errorf("newManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, m)
	}
}

// Tests happy path of LoadManager.
func TestLoadManager(t *testing.T) {
	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	// Attempt to load relationship
	newM, err := LoadManager(kv, expectedM.myID, expectedM.partner,
		expectedM.cyHandler, expectedM.grp, expectedM.rng)
	if err != nil {
		t.Fatalf("LoadManager() returned an error: %v", err)
	}
	// Check if the loaded relationship matches the expected
	if !managersEqual(&expectedM, newM.(*manager), t) {
		t.Errorf("LoadManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, newM.(*manager))
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

	err := expectedM.Delete()
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
		session.GetDefaultParams(), m.cyHandler, m.grp, m.rng)

	se, exists := m.NewReceiveSession(partnerPubKey, partnerSIDHPubKey,
		session.GetDefaultParams(), thisSession)
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
		session.GetDefaultParams(), thisSession)
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
		session.GetDefaultParams(), m.send.sessions[0])
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}

	se, _ = m.NewReceiveSession(m.originPartnerPubKey, m.originPartnerSIDHPubKey,
		session.GetDefaultParams(), m.receive.sessions[0])
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

	pid := m.PartnerId()

	if !m.partner.Cmp(pid) {
		t.Errorf("PartnerId() returned incorrect partner ID."+
			"\n\texpected: %s\n\treceived: %s", m.partner, pid)
	}
}

// Tests happy path of Manager.GetMyID.
func TestManager_GetMyID(t *testing.T) {
	myId := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	m := &manager{myID: myId}

	receivedMyId := m.MyId()

	if !myId.Cmp(receivedMyId) {
		t.Errorf("MyId() returned incorrect partner ID."+
			"\n\texpected: %s\n\treceived: %s", myId, receivedMyId)
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
		session.Sending, session.GetDefaultParams())

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

func TestManager_MakeService(t *testing.T) {
	m, _ := newTestManager(t)
	tag := "hunter2"
	expected := message.Service{
		Identifier: m.ConnectionFingerprint().Bytes(),
		Tag:        tag,
		Metadata:   m.partner[:],
	}

	received := m.MakeService(tag)

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("MakeService returned unexpected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}

}
