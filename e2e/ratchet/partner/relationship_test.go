///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partner

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet"
	session7 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	session6 "gitlab.com/elixxir/client/e2e/ratchet/session"
	"gitlab.com/elixxir/client/interfaces/params"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Subtest: unmarshal/marshal with one session in the buff
func TestRelationship_MarshalUnmarshal(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	// Serialization should include session slice only
	serialized, err := sb.marshal()
	if err != nil {
		t.Fatal(err)
	}

	sb2 := &relationship{
		manager:     mgr,
		t:           0,
		kv:          sb.kv,
		sessions:    make([]*session7.Session, 0),
		sessionByID: make(map[session7.SessionID]*session7.Session),
	}

	err = sb2.unmarshal(serialized)
	if err != nil {
		t.Fatal(err)
	}

	// compare sb2 session list and map
	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffs not equal")
	}
}

// Shows that Relationship returns an equivalent session buff to the one that was saved
func TestLoadRelationship(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	sb2, err := LoadRelationship(mgr, session7.Send)
	if err != nil {
		t.Fatal(err)
	}

	if !relationshipsEqual(sb, sb2) {
		t.Error("session buffers not equal")
	}
}

// Shows that a deleted Relationship can no longer be pulled from store
func TestDeleteRelationship(t *testing.T) {
	mgr := makeTestRelationshipManager(t)

	// Generate send relationship
	mgr.send = NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	if err := mgr.send.save(); err != nil {
		t.Fatal(err)
	}

	// Generate receive relationship
	mgr.receive = NewRelationship(mgr, session7.Receive, params.GetDefaultE2ESessionParams())
	if err := mgr.receive.save(); err != nil {
		t.Fatal(err)
	}

	err := DeleteRelationship(mgr)
	if err != nil {
		t.Fatalf("DeleteRelationship error: Could not delete manager: %v", err)
	}

	_, err = LoadRelationship(mgr, session7.Send)
	if err == nil {
		t.Fatalf("DeleteRelationship error: Should not have loaded deleted relationship: %v", err)
	}

	_, err = LoadRelationship(mgr, session7.Receive)
	if err == nil {
		t.Fatalf("DeleteRelationship error: Should not have loaded deleted relationship: %v", err)
	}
}

// Shows that a deleted relationship fingerprint can no longer be pulled from store
func TestRelationship_deleteRelationshipFingerprint(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("deleteRelationshipFingerprint error: " +
				"Did not panic when loading deleted fingerprint")
		}
	}()

	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	err = deleteRelationshipFingerprint(mgr.kv)
	if err != nil {
		t.Fatalf("deleteRelationshipFingerprint error: "+
			"Could not delete fingerprint: %v", err)
	}

	loadRelationshipFingerprint(mgr.kv)
}

// Shows that Relationship returns a valid session buff
func TestNewRelationshipBuff(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	if len(sb.sessions) != 1 {
		t.Error("starting session slice length should be 1")
	}
	if len(sb.sessionByID) != 1 {
		t.Error("starting session map length should be 1")
	}
	session, _ := session6.makeTestSession()
	// Note: AddSession doesn't change the session relationship or set anything else up
	// to match the session to the session buffer. To work properly, the session
	// should have been created using the same relationship (which is not the case in
	// this test.)
	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource, session7.Sending, session.e2eParams)
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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	// The newest session should be nil upon session buffer creation
	nilSession := sb.GetNewest()
	if nilSession == nil {
		t.Error("should not have gotten a nil session from a buffer " +
			"with one session")
	}

	session, _ := session6.makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource, session7.Sending, session.e2eParams)
	if session.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

	session2, _ := session6.makeTestSession()
	sb.AddSession(session2.myPrivKey, session2.partnerPubKey, nil,
		session2.mySIDHPrivKey, session2.partnerSIDHPubKey,
		session2.partnerSource, session7.Sending, session.e2eParams)
	if session2.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

}

// Shows that Confirm confirms the specified session in the buff
func TestRelationship_Confirm(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	session, _ := session6.makeTestSession()

	sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource, session7.Sending, session.e2eParams)
	sb.sessions[1].negotiationStatus = session7.Sent

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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	session, _ := session6.makeTestSession()

	err := sb.Confirm(session.GetID())
	if err == nil {
		t.Error("Confirming a session not in the buff should result in an error")
	}
}

// Shows that a session can get got by ID from the buff
func TestRelationship_GetByID(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	session, _ := session6.makeTestSession()
	session = sb.AddSession(session.myPrivKey, session.partnerPubKey, nil,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource, session7.Sending, session.e2eParams)
	session2 := sb.GetByID(session.GetID())
	if !reflect.DeepEqual(session, session2) {
		t.Error("gotten session should be the same")
	}
}

// Shows that GetNewestRekeyableSession acts as expected:
// returning sessions that are confirmed and past rekeyThreshold
func TestRelationship_GetNewestRekeyableSession(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	sb.sessions[0].negotiationStatus = session7.Unconfirmed
	// no available rekeyable sessions: nil
	session2 := sb.getNewestRekeyableSession()
	if session2 != sb.sessions[0] {
		t.Error("newest rekeyable session should be the unconfired session")
	}

	// add a rekeyable session: that session
	session, _ := session6.makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey, session.baseKey,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource, session7.Sending, session.e2eParams)
	sb.sessions[0].negotiationStatus = session7.Confirmed
	session3 := sb.getNewestRekeyableSession()

	if session3 == nil {
		t.Error("no session returned")
	} else if session3.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// add another rekeyable session: that session
	// show the newest session is selected
	additionalSession, _ := session6.makeTestSession()
	sb.AddSession(additionalSession.myPrivKey,
		additionalSession.partnerPubKey, nil,
		additionalSession.mySIDHPrivKey,
		additionalSession.partnerSIDHPubKey,
		additionalSession.partnerSource,
		session7.Sending, additionalSession.e2eParams)

	sb.sessions[0].negotiationStatus = session7.Confirmed

	session4 := sb.getNewestRekeyableSession()
	if session4 == nil {
		t.Error("no session returned")
	} else if session4.GetID() != sb.sessions[0].GetID() {
		t.Error("didn't get the expected session")
	}

	// make the very newest session unrekeyable: the previous session
	// sb.sessions[1].negotiationStatus = Confirmed
	sb.sessions[0].negotiationStatus = session7.Unconfirmed

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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	sb.sessions = make([]*session7.Session, 0)
	sb.sessionByID = make(map[session7.SessionID]*session7.Session)

	none := sb.getSessionForSending()
	if none != nil {
		t.Error("getSessionForSending should return nil if there aren't any sendable sessions")
	}

	// First case: unconfirmed rekey
	unconfirmedRekey, _ := session6.makeTestSession()

	sb.AddSession(unconfirmedRekey.myPrivKey, unconfirmedRekey.partnerPubKey,
		unconfirmedRekey.partnerPubKey, // FIXME? Shoudln't this be nil?
		unconfirmedRekey.mySIDHPrivKey,
		unconfirmedRekey.partnerSIDHPubKey,
		unconfirmedRekey.partnerSource,
		session7.Sending, unconfirmedRekey.e2eParams)
	sb.sessions[0].negotiationStatus = session7.Unconfirmed
	sb.sessions[0].keyState.SetNumKeysTEST(2000, t)
	sb.sessions[0].rekeyThreshold = 1000
	sb.sessions[0].keyState.SetNumAvailableTEST(600, t)
	sending := sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}
	if sending.Status() != session7.RekeyNeeded || sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'RekeyNedded' "+
			"'Unconfirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	// Second case: unconfirmed active
	unconfirmedActive, _ := session6.makeTestSession()

	sb.AddSession(unconfirmedActive.myPrivKey,
		unconfirmedActive.partnerPubKey,
		unconfirmedActive.partnerPubKey,
		unconfirmedActive.mySIDHPrivKey,
		unconfirmedActive.partnerSIDHPubKey,
		unconfirmedActive.partnerSource,
		session7.Sending, unconfirmedActive.e2eParams)
	sb.sessions[0].negotiationStatus = session7.Unconfirmed
	sb.sessions[0].keyState.SetNumKeysTEST(2000, t)
	sb.sessions[0].rekeyThreshold = 1000
	sb.sessions[0].keyState.SetNumAvailableTEST(2000, t)
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}

	if sending.Status() != session7.Active || sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'Active' "+
			"'Unconfirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	// Third case: confirmed rekey
	confirmedRekey, _ := session6.makeTestSession()

	sb.AddSession(confirmedRekey.myPrivKey, confirmedRekey.partnerPubKey,
		confirmedRekey.partnerPubKey,
		confirmedRekey.mySIDHPrivKey,
		confirmedRekey.partnerSIDHPubKey,
		confirmedRekey.partnerSource,
		session7.Sending, confirmedRekey.e2eParams)
	sb.sessions[0].negotiationStatus = session7.Confirmed
	sb.sessions[0].keyState.SetNumKeysTEST(2000, t)
	sb.sessions[0].rekeyThreshold = 1000
	sb.sessions[0].keyState.SetNumAvailableTEST(600, t)
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Error("got an unexpected session")
	}
	if sending.Status() != session7.RekeyNeeded || !sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'RekeyNeeded' "+
			"'Confirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}

	// Fourth case: confirmed active
	confirmedActive, _ := session6.makeTestSession()
	sb.AddSession(confirmedActive.myPrivKey, confirmedActive.partnerPubKey,
		confirmedActive.partnerPubKey,
		confirmedActive.mySIDHPrivKey,
		confirmedActive.partnerSIDHPubKey,
		confirmedActive.partnerSource,
		session7.Sending, confirmedActive.e2eParams)

	sb.sessions[0].negotiationStatus = session7.Confirmed
	sb.sessions[0].keyState.SetNumKeysTEST(2000, t)
	sb.sessions[0].keyState.SetNumAvailableTEST(2000, t)
	sb.sessions[0].rekeyThreshold = 1000
	sending = sb.getSessionForSending()
	if sending.GetID() != sb.sessions[0].GetID() {
		t.Errorf("got an unexpected session of state: %s", sending.Status())
	}
	if sending.Status() != session7.Active || !sending.IsConfirmed() {
		t.Errorf("returned session is expected to be 'Active' "+
			"'Confirmed', it is: %s, confirmed: %v", sending.Status(),
			sending.IsConfirmed())
	}
}

// Shows that GetKeyForRekey returns a key if there's an appropriate session for rekeying
func TestSessionBuff_GetKeyForRekey(t *testing.T) {
	mgr := makeTestRelationshipManager(t)
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	sb.sessions = make([]*session7.Session, 0)
	sb.sessionByID = make(map[session7.SessionID]*session7.Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForRekey()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := session6.makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource,
		session7.Sending, session.e2eParams)
	sb.sessions[0].negotiationStatus = session7.Confirmed
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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())

	sb.sessions = make([]*session7.Session, 0)
	sb.sessionByID = make(map[session7.SessionID]*session7.Session)

	// no available rekeyable sessions: error
	key, err := sb.getKeyForSending()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := session6.makeTestSession()
	sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource,
		session7.Sending, session.e2eParams)
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
	sb := NewRelationship(mgr, session7.Send, params.GetDefaultE2ESessionParams())
	sb.sessions = make([]*session7.Session, 0)
	sb.sessionByID = make(map[session7.SessionID]*session7.Session)

	session, _ := session6.makeTestSession()
	session = sb.AddSession(session.myPrivKey, session.partnerPubKey,
		session.partnerPubKey,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session.partnerSource,
		session7.Sending, session.e2eParams)
	session.negotiationStatus = session7.Confirmed
	// The added session isn't ready for rekey, so it's not returned here
	negotiations := sb.TriggerNegotiation()
	if len(negotiations) != 0 {
		t.Errorf("should have had zero negotiations: %+v", negotiations)
	}
	session2, _ := session6.makeTestSession()
	// Make only a few keys available to trigger the rekeyThreshold
	session2 = sb.AddSession(session2.myPrivKey, session2.partnerPubKey,
		session2.partnerPubKey,
		session.mySIDHPrivKey, session.partnerSIDHPubKey,
		session2.partnerSource,
		session7.Sending, session2.e2eParams)
	session2.keyState.SetNumAvailableTEST(4, t)
	session2.negotiationStatus = session7.Confirmed
	negotiations = sb.TriggerNegotiation()
	if len(negotiations) != 1 {
		t.Fatal("should have had one negotiation")
	}
	if negotiations[0].GetID() != session2.GetID() {
		t.Error("negotiated sessions should include the rekeyable " +
			"session")
	}
	if session2.negotiationStatus != session7.NewSessionTriggered {
		t.Errorf("Trigger negotiations should have set status to "+
			"triggered: %s", session2.negotiationStatus)
	}

	// Unconfirmed sessions should also be included in the list
	// as the client should attempt to confirm them
	session3, _ := session6.makeTestSession()

	session3 = sb.AddSession(session3.myPrivKey, session3.partnerPubKey,
		session3.partnerPubKey,
		session3.mySIDHPrivKey, session3.partnerSIDHPubKey,
		session3.partnerSource,
		session7.Sending, session3.e2eParams)
	session3.negotiationStatus = session7.Unconfirmed

	// Set session 2 status back to Confirmed to show that more than one session can be returned
	session2.negotiationStatus = session7.Confirmed
	// Trigger negotiations
	negotiations = sb.TriggerNegotiation()

	if len(negotiations) != 2 {
		t.Fatal("num of negotiated sessions here should be 2")
	}
	found := false
	for i := range negotiations {
		if negotiations[i].GetID() == session3.GetID() {
			found = true
			if negotiations[i].negotiationStatus != session7.Sending {
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
	rng := csprng.NewSystemRNG()
	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)
	fps := ratchet.newFingerprints()
	g := session6.getGroup()
	return &Manager{
		ctx: &ratchet.context{
			fa:   &fps,
			grp:  g,
			myID: &id.ID{},
		},
		kv:                      versioned.NewKV(make(ekv.Memstore)),
		partner:                 id.NewIdFromUInt(8, id.User, t),
		originMyPrivKey:         g.NewInt(2),
		originPartnerPubKey:     g.NewInt(3),
		originMySIDHPrivKey:     mySIDHPrivKey,
		originPartnerSIDHPubKey: partnerSIDHPubKey,
	}
}

// Revises a session to fit a sessionbuff and saves it to the sessionbuff's kv store
func adaptToBuff(session *session7.Session, buff *relationship, t *testing.T) {
	session.relationship = buff
	session.keyState.SetKvTEST(buff.manager.kv, t)
	err := session.keyState.SaveTEST(t)
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

func Test_relationship_getNewestRekeyableSession(t *testing.T) {
	// TODO: Add test cases.
}
