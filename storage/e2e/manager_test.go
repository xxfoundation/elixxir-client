package e2e

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests happy path of newManager.
func Test_newManager(t *testing.T) {
	// Set up expected and test values
	s, ctx := makeTestSession()
	kv := versioned.NewKV(make(ekv.Memstore))
	partnerID := id.NewIdFromUInt(100, id.User, t)
	expectedM := &Manager{
		ctx:                 ctx,
		kv:                  kv.Prefix(fmt.Sprintf(managerPrefix, partnerID)),
		partner:             partnerID,
		originPartnerPubKey: s.partnerPubKey,
		originMyPrivKey:     s.myPrivKey,
	}
	expectedM.send = NewRelationship(expectedM, Send, GetDefaultSessionParams())
	expectedM.receive = NewRelationship(expectedM, Receive, GetDefaultSessionParams())

	// Create new relationship
	m := newManager(ctx, kv, partnerID, s.myPrivKey, s.partnerPubKey, s.params,
		s.params)

	// Check if the new relationship matches the expected
	if !managersEqual(expectedM, m, t) {
		t.Errorf("newManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, m)
	}
}

// Tests happy path of loadManager.
func TestLoadManager(t *testing.T) {
	// Set up expected and test values
	expectedM, kv := newTestManager(t)

	// Attempt to load relationship
	m, err := loadManager(expectedM.ctx, kv, expectedM.partner)
	if err != nil {
		t.Errorf("loadManager() returned an error: %v", err)
	}

	// Check if the loaded relationship matches the expected
	if !managersEqual(expectedM, m, t) {
		t.Errorf("loadManager() did not produce the expected Manager."+
			"\n\texpected: %+v\n\treceived: %+v", expectedM, m)
	}
}

// Tests happy path of Manager.NewReceiveSession.
func TestManager_NewReceiveSession(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	s, _ := makeTestSession()

	se, exists := m.NewReceiveSession(s.partnerPubKey, s.params, s)
	if exists {
		t.Errorf("NewReceiveSession() did not return the correct value."+
			"\n\texpected: %v\n\treceived: %v", false, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(s.GetID().Marshal(), se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), s.GetID(), se.GetID())
	}

	se, exists = m.NewReceiveSession(s.partnerPubKey, s.params, s)
	if !exists {
		t.Errorf("NewReceiveSession() did not return the correct value."+
			"\n\texpected: %v\n\treceived: %v", true, exists)
	}
	if !m.partner.Cmp(se.GetPartner()) || !bytes.Equal(s.GetID().Marshal(), se.GetID().Marshal()) {
		t.Errorf("NewReceiveSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v"+
			"\n\texpected ID: %v\n\treceived ID: %v",
			m.partner, se.GetPartner(), s.GetID(), se.GetID())
	}
}

// Tests happy path of Manager.NewSendSession.
func TestManager_NewSendSession(t *testing.T) {
	// Set up test values
	m, _ := newTestManager(t)
	s, _ := makeTestSession()

	se := m.NewSendSession(s.myPrivKey, s.params)
	if !m.partner.Cmp(se.GetPartner()) {
		t.Errorf("NewSendSession() did not return the correct session."+
			"\n\texpected partner: %v\n\treceived partner: %v",
			m.partner, se.GetPartner())
	}

	se = m.NewSendSession(s.partnerPubKey, s.params)
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
	expectedKey := &Key{
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
	m.send.sessions[0].negotiationStatus = NewSessionTriggered
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
	m.send.sessions[0].negotiationStatus = Sent
	err := m.Confirm(m.send.sessions[0].GetID())
	if err != nil {
		t.Errorf("Confirm produced an error: %v", err)
	}
}

// Tests happy path of Manager.TriggerNegotiations.
func TestManager_TriggerNegotiations(t *testing.T) {
	m, _ := newTestManager(t)
	m.send.sessions[0].negotiationStatus = Unconfirmed
	sessions := m.TriggerNegotiations()
	if !reflect.DeepEqual(m.send.sessions, sessions) {
		t.Errorf("TriggerNegotiations() returned incorrect sessions."+
			"\n\texpected: %s\n\treceived: %s", m.send.sessions, sessions)
	}
}

// newTestManager returns a new relationship for testing.
func newTestManager(t *testing.T) (*Manager, *versioned.KV) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	s, ctx := makeTestSession()
	kv := versioned.NewKV(make(ekv.Memstore))
	partnerID := id.NewIdFromUInts([4]uint64{prng.Uint64(), prng.Uint64(),
		prng.Uint64(), prng.Uint64()}, id.User, t)

	// Create new relationship
	m := newManager(ctx, kv, partnerID, s.myPrivKey, s.partnerPubKey, s.params,
		s.params)

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
