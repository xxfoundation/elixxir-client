///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Subtest: unmarshal/marshal with one session in the buff
func TestRelationship_MarshalUnmarshal(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())

	// Serialization should include session slice only
	serialized, err := sb.marshal()
	if err != nil {
		t.Fatal(err)
	}

	sb2 := &relationship{
		manager:     mgr,
		t:           0,
		kv:          sb.kv,
		sessions:    make([]*Session, 0),
		sessionByID: make(map[SessionID]*Session),
	}

	err = sb2.unmarshal(serialized)
	if err != nil {
		t.Fatal(err)
	}

	// compare sb2 sesh list and map
	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffs not equal")
	}
}

// Shows that Relationship returns an equivalent session buff to the one that was saved
func TestLoadRelationship(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())

	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	sb2, err := LoadRelationship(mgr, Send)
	if err != nil {
		t.Fatal(err)
	}

	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffers not equal")
	}
}

// Shows that Relationship returns a valid session buff
func TestNewRelationshipBuff(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	if mgr != sb.manager {
		t.Error("managers should be identical")
	}
	if sb.sessionByID == nil || len(sb.sessionByID) != 1 {
		t.Error("session map should not be nil, and should have one " +
			"element")
	}
	if sb.sessions == nil || len(sb.sessions) != 1 {
		t.Error("session list should not be nil, and should have one " +
			"element")
	}
}

// Shows that AddSession adds one session to the relationship
func TestRelationship_AddSession(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	if len(sb.sessions) != 1 {
		t.Error("starting session slice length should be 1")
	}
	if len(sb.sessionByID) != 1 {
		t.Error("starting session map length should be 1")
	}
	session, _ := makeTestSession()
	// Note: AddSession doesn't change the session relationship or set anything else up
	// to match the session to the session buffer. To work properly, the session
	// should have been created using the same relationship (which is not the case in
	// this test.)
	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.partnerSource, session.params)
	if len(sb.sessions) != 2 {
		t.Error("ending session slice length should be 2")
	}
	if len(sb.sessionByID) != 2 {
		t.Error("ending session map length should be 2")
	}
	if session.GetID() != sb.sessions[0].GetID() {
		t.Error("session added should have same ID")
	}
}

// GetNewest should get the session that was most recently added to the buff
func TestRelationship_GetNewest(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	// The newest session should be nil upon session buffer creation
	nilSession := sb.GetNewest()
	if nilSession == nil {
		t.Error("should not have gotten a nil session from a buffer " +
			"with one session")
	}

	session, _ := makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.partnerSource, session.params)
	if session.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

	session2, _ := makeTestSession()
	sb.AddSession(session2.myPrivKey, session2.partnerPubKey, nil,
		session2.partnerSource, session2.params)
	if session2.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

}

// Shows that Confirm confirms the specified session in the buff
func TestRelationship_Confirm(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	session, _ := makeTestSession()

	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.partnerSource, session.params)
	sb.sessions[1].negotiationStatus = Sent

	if sb.sessions[1].IsConfirmed() {
		t.Error("session should not be confirmed before confirmation")
	}

	err := sb.Confirm(sb.sessions[1].GetID())
	if err != nil {
		t.Fatal(err)
	}

	if !sb.sessions[1].IsConfirmed() {
		t.Error("session should be confirmed after confirmation")
	}
}

// Shows that the session buff returns an error when the session doesn't exist
func TestRelationship_Confirm_Err(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	session, _ := makeTestSession()

	err := sb.Confirm(session.GetID())
	if err == nil {
		t.Error("Confirming a session not in the buff should result in an error")
	}
}

// Shows that a session can get got by ID from the buff
func TestRelationship_GetByID(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	session, _ := makeTestSession()
	session = sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.partnerSource, session.params)
	session2 := sb.GetByID(session.GetID())
	if !reflect.DeepEqual(session, session2) {
		t.Error("gotten session should be the same")
	}
}

// Shows that GetNewestRekeyableSession acts as expected:
// returning sessions that are confirmed and past ttl
func TestRelationship_GetNewestRekeyableSession(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	sb.sessions[0].negotiationStatus = Unconfirmed
	// no available rekeyable sessions: nil
	session2 := sb.getNewestRekeyableSession()
	if session2 != nil {
		t.Error("newest rekeyable session should be nil")
	}

	// add a rekeyable session: that session
	session, _ := makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey, session.baseKey,
		session.partnerSource, session.params)
	sb.sessions[0].negotiationStatus = Confirmed
	session3 := sb.getNewestRekeyableSession()

	if session3 == nil {
		t.Error("no session returned")
	} else if session3.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// add another rekeyable session: that session
	// show the newest session is selected
	additionalSession, _ := makeTestSession()
	sb.AddSession(additionalSession.myPrivKey, additionalSession.partnerPubKey,
		additionalSession.partnerPubKey, additionalSession.partnerSource,
		additionalSession.params)

	sb.sessions[0].negotiationStatus = Confirmed

	session4 := sb.getNewestRekeyableSession()
	if session4 == nil {
		t.Error("no session returned")
	} else if session4.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// make the very newest session unrekeyable: the previous session
	//sb.sessions[1].negotiationStatus = Confirmed
	sb.sessions[0].negotiationStatus = Unconfirmed

	session5 := sb.getNewestRekeyableSession()
	if session5 == nil {
		t.Error("no session returned")
	} else if session5.GetID() != sb.sessions[1].GetID() {
		t.Error("didn't get the expected session")
	}
}

// Shows that GetSessionForSending follows the hierarchy of sessions correctly
func TestRelationship_GetSessionForSending(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())

	sb.sessions = make([]*Session, 0)
	sb.sessionByID = make(map[SessionID]*Session)

	none := sb.getSessionForSending()
	if none != nil {
		t.Error("getSessionForSending should return nil if there aren't any sendable sessions")
	}

	// First case: unconfirmed rekey
	unconfirmedRekey, _ := makeTestSession()

	sb.AddSession(unconfirmedRekey.myPrivKey, unconfirmedRekey.partnerPubKey,
		unconfirmedRekey.partnerPubKey, unconfirmedRekey.partnerSource,
		unconfirmedRekey.params)
	sb.sessions[0].negotiationStatus = Unconfirmed
	sb.sessions[0].keyState.numAvailable = 600
	sending := sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}

	// Second case: unconfirmed active
	unconfirmedActive, _ := makeTestSession()

	sb.AddSession(unconfirmedActive.myPrivKey, unconfirmedActive.partnerPubKey,
		unconfirmedActive.partnerPubKey, unconfirmedActive.partnerSource,
		unconfirmedActive.params)
	sb.sessions[0].negotiationStatus = Unconfirmed
	sb.sessions[0].keyState.numAvailable = 2000
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}

	// Third case: confirmed rekey
	confirmedRekey, _ := makeTestSession()

	sb.AddSession(confirmedRekey.myPrivKey, confirmedRekey.partnerPubKey,
		confirmedRekey.partnerPubKey, confirmedRekey.partnerSource,
		confirmedRekey.params)
	sb.sessions[0].negotiationStatus = Confirmed
	sb.sessions[0].keyState.numAvailable = 600
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}

	// Fourth case: confirmed active
	confirmedActive, _ := makeTestSession()
	sb.AddSession(confirmedActive.myPrivKey, confirmedActive.partnerPubKey,
		confirmedActive.partnerPubKey, confirmedActive.partnerSource,
		confirmedActive.params)

	sb.sessions[0].negotiationStatus = Confirmed
	sb.sessions[0].keyState.numAvailable = 2000
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}
}

// Shows that GetKeyForRekey returns a key if there's an appropriate session for rekeying
func TestSessionBuff_GetKeyForRekey(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())

	sb.sessions = make([]*Session, 0)
	sb.sessionByID = make(map[SessionID]*Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForRekey()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey, session.partnerSource,
		session.params)
	sb.sessions[0].negotiationStatus = Confirmed
	key, err = sb.getKeyForRekey()
	if err != nil {
		t.Error(err)
	}
	if key == nil {
		t.Error("should have returned a valid key with a rekeyable session available")
	}
}

// Shows that GetKeyForSending returns a key if there's an appropriate session for sending
func TestSessionBuff_GetKeyForSending(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())

	sb.sessions = make([]*Session, 0)
	sb.sessionByID = make(map[SessionID]*Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForSending()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey, session.partnerSource,
		session.params)
	key, err = sb.getKeyForSending()
	if err != nil {
		t.Error(err)
	}
	if key == nil {
		t.Error("should have returned a valid key with a sendable session available")
	}
}

// Shows that TriggerNegotiation sets up for negotiation correctly
func TestSessionBuff_TriggerNegotiation(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, Send, GetDefaultSessionParams())
	sb.sessions = make([]*Session, 0)
	sb.sessionByID = make(map[SessionID]*Session)

	session, _ := makeTestSession()
	session = sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey, session.partnerSource,
		session.params)
	session.negotiationStatus = Confirmed
	// The added session isn't ready for rekey so it's not returned here
	negotiations := sb.TriggerNegotiation()
	if len(negotiations) != 0 {
		t.Errorf("should have had zero negotiations: %+v", negotiations)
	}
	session2, _ := makeTestSession()
	// Make only a few keys available to trigger the ttl
	session2 = sb.AddSession(session2.myPrivKey, session2.partnerPubKey,
		session2.partnerPubKey, session2.partnerSource,
		session2.params)
	session2.keyState.numAvailable = 4
	session2.negotiationStatus = Confirmed
	negotiations = sb.TriggerNegotiation()
	if len(negotiations) != 1 {
		t.Fatal("should have had one negotiation")
	}
	if negotiations[0].GetID() != session2.GetID() {
		t.Error("negotiated sessions should include the rekeyable " +
			"session")
	}
	if session2.negotiationStatus != NewSessionTriggered {
		t.Errorf("Trigger negotiations should have set status to "+
			"triggered: %s", session2.negotiationStatus)
	}

	// Unconfirmed sessions should also be included in the list
	// as the client should attempt to confirm them
	session3, _ := makeTestSession()

	session3 = sb.AddSession(session3.myPrivKey, session3.partnerPubKey,
		session3.partnerPubKey, session3.partnerSource,
		session3.params)
	session3.negotiationStatus = Unconfirmed

	// Set session 2 status back to Confirmed to show that more than one session can be returned
	session2.negotiationStatus = Confirmed
	// Trigger negotiations
	negotiations = sb.TriggerNegotiation()

	if len(negotiations) != 2 {
		t.Fatal("num of negotiated sessions here should be 2")
	}
	found := false
	for i := range negotiations {
		if negotiations[i].GetID() == session3.GetID() {
			found = true
			if negotiations[i].negotiationStatus != Sending {
				t.Error("triggering negotiation should change session3 to sending")
			}
		}
	}
	if !found {
		t.Error("session3 not found")
	}

	found = false
	for i := range negotiations {
		if negotiations[i].GetID() == session2.GetID() {
			found = true
		}
	}
	if !found {
		t.Error("session2 not found")
	}
}

func makeTestRelationshipManager(t *testing.T) *Manager {
	fps := newFingerprints()
	g := getGroup()
	return &Manager{
		ctx: &context{
			fa:   &fps,
			grp:  g,
			myID: &id.ID{},
		},
		kv:                  versioned.NewKV(make(ekv.Memstore)),
		partner:             id.NewIdFromUInt(8, id.User, t),
		originMyPrivKey:     g.NewInt(2),
		originPartnerPubKey: g.NewInt(3),
	}
}

// Revises a session to fit a sessionbuff and saves it to the sessionbuff's kv store
func adaptToBuff(session *Session, buff *relationship, t *testing.T) {
	session.relationship = buff
	session.keyState.kv = buff.manager.kv
	err := session.keyState.save()
	if err != nil {
		t.Fatal(err)
	}
	err = session.save()
	if err != nil {
		t.Fatal(err)
	}
}

// Compare certain fields of two session buffs for equality
func relationshipsEqual(buff *relationship, buff2 *relationship) bool {
	if len(buff.sessionByID) != len(buff2.sessionByID) {
		return false
	}
	if len(buff.sessions) != len(buff2.sessions) {
		return false
	}

	if !bytes.Equal(buff.fingerprint, buff2.fingerprint) {
		return false
	}
	// Make sure all sessions are present
	for k := range buff.sessionByID {
		_, ok := buff2.sessionByID[k]
		if !ok {
			// key not present in other map
			return false
		}
	}
	// Comparing base key only for now
	// This should ensure that the session buffers have the same sessions in the same order
	for i := range buff.sessions {
		if buff.sessions[i].baseKey.Cmp(buff2.sessions[i].baseKey) != 0 {
			return false
		}
	}
	return true
}
