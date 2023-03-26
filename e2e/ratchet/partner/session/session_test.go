////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"reflect"
	"testing"
)

func TestSession_generate_noPrivateKeyReceive(t *testing.T) {

	s, _ := makeTestSession()

	// run the finalizeKeyNegotation command
	s.finalizeKeyNegotiation()

	// check that it generated a private key
	if s.myPrivKey == nil {
		t.Errorf("Private key was not generated when missing")
	}

	// verify the base key is correct
	expectedBaseKey := GenerateE2ESessionBaseKey(s.myPrivKey,
		s.partnerPubKey, s.grp, s.mySIDHPrivKey, s.partnerSIDHPubKey)

	if expectedBaseKey.Cmp(s.baseKey) != 0 {
		t.Errorf("generated base key does not match expected base key")
	}

	// verify the rekeyThreshold was generated
	if s.rekeyThreshold == 0 {
		t.Errorf("rekeyThreshold not generated")
	}

	// verify key states was created
	if s.keyState == nil {
		t.Errorf("keystates not generated")
	}

}

func TestSession_generate_PrivateKeySend(t *testing.T) {

	// build the session
	s, _ := makeTestSession()

	// run the finalizeKeyNegotation command
	s.finalizeKeyNegotiation()

	// check that it generated a private key
	if s.myPrivKey.Cmp(s.myPrivKey) != 0 {
		t.Errorf("Public key was generated when not missing")
	}

	// verify the base key is correct
	expectedBaseKey := GenerateE2ESessionBaseKey(s.myPrivKey,
		s.partnerPubKey, s.grp, s.mySIDHPrivKey, s.partnerSIDHPubKey)

	if expectedBaseKey.Cmp(s.baseKey) != 0 {
		t.Errorf("generated base key does not match expected base key")
	}

	// verify the rekeyThreshold was generated
	if s.rekeyThreshold == 0 {
		t.Errorf("rekeyThreshold not generated")
	}

	// verify keyState was created
	if s.keyState == nil {
		t.Errorf("keystates not generated")
	}

}

// Shows that NewSession can result in all the fields being populated
func TestNewSession(t *testing.T) {
	// Make a test session to easily populate all the fields
	sessionA, _ := makeTestSession()

	// Make a new session with the variables we got from MakeTestSession
	sessionB := NewSession(sessionA.kv, sessionA.t, sessionA.partner,
		sessionA.myPrivKey, sessionA.partnerPubKey, sessionA.baseKey,
		sessionA.mySIDHPrivKey, sessionA.partnerSIDHPubKey,
		sessionA.GetID(), []byte(""), sessionA.negotiationStatus,
		sessionA.e2eParams, sessionA.cyHandler, sessionA.grp, sessionA.rng)

	err := cmpSerializedFields(sessionA, sessionB)
	if err != nil {
		t.Error(err)
	}
	// For everything else, just make sure it's populated
	if sessionB.keyState == nil {
		t.Error("NewSession should populate keyState")
	}
	// fixme is this deleted?
	//if sessionB.relationship == nil {
	//	t.Error("NewSession should populate relationship")
	//}
	if sessionB.rekeyThreshold == 0 {
		t.Error("NewSession should populate rekeyThreshold")
	}
}

// Shows that LoadSession can result in all the fields being populated
func TestSession_Load(t *testing.T) {
	// Make a test session to easily populate all the fields
	sessionA, kv := makeTestSession()
	err := sessionA.Save()
	if err != nil {
		t.Fatal(err)
	}

	// SessionA.kv will have a prefix set in MakeTestSession
	// initialize a new one for Load, which will set a prefix internally

	// Load another, identical session from the storage
	sessionB, err := LoadSession(kv, sessionA.GetID(), sessionA.relationshipFingerprint,
		sessionA.cyHandler, sessionA.grp, sessionA.rng)
	if err != nil {
		t.Fatal(err)
	}
	err = cmpSerializedFields(sessionA, sessionB)
	if err != nil {
		t.Error(err)
	}
	// Key state should also be loaded and equivalent to the other session
	// during LoadSession()
	if !reflect.DeepEqual(sessionA.keyState, sessionB.keyState) {
		t.Errorf("Two key states do not match.\nsessionA: %+v\nsessionB: %+v",
			sessionA.keyState, sessionB.keyState)
	}
	// For everything else, just make sure it's populated
	// fixme is this deleted?
	//if sessionB.relationship == nil {
	//	t.Error("load should populate relationship")
	//}
	if sessionB.rekeyThreshold == 0 {
		t.Error("load should populate rekeyThreshold")
	}
}

// Create a new session. Marshal and unmarshal it
func TestSession_Serialization(t *testing.T) {
	s, _ := makeTestSession()
	sSerialized, err := s.marshal()
	if err != nil {
		t.Fatal(err)
	}

	sDeserialized := &Session{
		//relationship: &ratchet.relationship{
		//	manager: &partner.Manager{ctx: ctx},
		//},
		grp: s.grp,
		kv:  s.kv,
	}
	err = sDeserialized.unmarshal(sSerialized)
	if err != nil {
		t.Fatal(err)
	}

}

// PopKey should return a new key from this session
func TestSession_PopKey(t *testing.T) {
	s, _ := makeTestSession()
	keyInterface, err := s.PopKey()
	if err != nil {
		t.Fatal(err)
	}
	key := keyInterface.(*cypher)

	if key == nil {
		t.Error("PopKey should have returned non-nil key")
	}
	if key.session != s {
		t.Error("Key should record it belongs to this session")
	}
	// PopKey should return the first available key
	if key.keyNum != 0 {
		t.Error("First key popped should have keyNum 0")
	}
}

// delete should remove unused keys from this session
func TestSession_Delete(t *testing.T) {
	s, _ := makeTestSession()
	err := s.Save()
	if err != nil {
		t.Fatal(err)
	}
	s.Delete()

	// Getting the keys that should have been stored should now result in an error
	_, err = utility.LoadStateVector(s.kv, "")
	if err == nil {
		t.Error("State vector was gettable")
	}
	_, err = s.kv.Get(sessionKey, 0)
	if err == nil {
		t.Error("Session was gettable")
	}
}

// PopKey should return an error if it's time for this session to rekey
// or if the key state vector is out of keys
// Unfortunately, the key state vector being out of keys is something
// that will also get caught by the other error first. So it's only practical
// to test the one error.
func TestSession_PopKey_Error(t *testing.T) {
	s, _ := makeTestSession()
	// Construct a specific state vector that will quickly run out of keys
	var err error
	s.keyState, err = utility.NewStateVector(s.kv, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.PopKey()
	if err == nil {
		t.Fatal("PopKey should have returned an error")
	}
	t.Log(err)
}

// PopRekey should return the next key
// There's no boundary, except for the number of keyNums in the state vector
func TestSession_PopReKey(t *testing.T) {
	s, _ := makeTestSession()
	keyInterface, err := s.PopReKey()
	if err != nil {
		t.Fatal("PopKey should have returned an error")
	}
	key := keyInterface.(*cypher)

	if key == nil {
		t.Error("Key should be non-nil")
	}
	if key.session != s {
		t.Error("Key should record it belongs to this session")
	}
	// PopReKey should return the first available key
	if key.keyNum != 0 {
		t.Error("First key popped should have keyNum 0")
	}
}

// PopRekey should not return the next key if there are no more keys available
// in the state vector
func TestSession_PopReKey_Err(t *testing.T) {
	s, _ := makeTestSession()
	// Construct a specific state vector that will quickly run out of keys
	var err error
	s.keyState, err = utility.NewStateVector(s.kv, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.PopReKey()
	if err == nil {
		t.Fatal("PopReKey should have returned an error")
	}
}

// Simple test that shows the base key can be got
func TestSession_GetBaseKey(t *testing.T) {
	s, _ := makeTestSession()
	baseKey := s.GetBaseKey()
	if baseKey.Cmp(s.baseKey) != 0 {
		t.Errorf("expected %v, got %v", baseKey.Text(16), s.baseKey.Text(16))
	}
}

// Smoke test for GetID
func TestSession_GetID(t *testing.T) {
	s, _ := makeTestSession()
	sid := s.GetID()
	if len(sid.Marshal()) == 0 {
		t.Error("Zero length for session ID!")
	}
}

// Smoke test for GetPartnerPubKey
func TestSession_GetPartnerPubKey(t *testing.T) {
	s, _ := makeTestSession()
	partnerPubKey := s.GetPartnerPubKey()
	if partnerPubKey.Cmp(s.partnerPubKey) != 0 {
		t.Errorf("expected %v, got %v", partnerPubKey.Text(16), s.partnerPubKey.Text(16))
	}
}

// Smoke test for GetMyPrivKey
func TestSession_GetMyPrivKey(t *testing.T) {
	s, _ := makeTestSession()
	myPrivKey := s.GetMyPrivKey()
	if myPrivKey.Cmp(s.myPrivKey) != 0 {
		t.Errorf("expected %v, got %v", myPrivKey.Text(16), s.myPrivKey.Text(16))
	}
}

// Shows that IsConfirmed returns whether the session is confirmed
func TestSession_IsConfirmed(t *testing.T) {
	s, _ := makeTestSession()
	s.negotiationStatus = Unconfirmed
	if s.IsConfirmed() {
		t.Error("s was confirmed when it shouldn't have been")
	}
	s.negotiationStatus = Confirmed
	if !s.IsConfirmed() {
		t.Error("s wasn't confirmed when it should have been")
	}
}

// Shows that Status can result in all possible statuses
func TestSession_Status(t *testing.T) {
	s, _ := makeTestSession()
	var err error
	s.keyState, err = utility.NewStateVector(s.kv, "", 500)
	if err != nil {
		t.Fatal(err)
	}
	s.keyState.SetNumAvailableTEST(0, t)
	if s.Status() != RekeyEmpty {
		t.Error("status should have been rekey empty with no keys left")
	}
	s.keyState.SetNumAvailableTEST(1, t)
	if s.Status() != Empty {
		t.Error("Status should have been empty")
	}
	// Passing the rekeyThreshold should result in a rekey being needed
	s.keyState.SetNumAvailableTEST(s.keyState.GetNumKeys()-s.rekeyThreshold, t)
	if s.Status() != RekeyNeeded {
		t.Error("Just past the rekeyThreshold, rekey should be needed")
	}
	s.keyState.SetNumAvailableTEST(s.keyState.GetNumKeys(), t)
	s.rekeyThreshold = 450
	if s.Status() != Active {
		t.Errorf("If all keys available, session should be active, recieved: %s", s.Status())
	}
}

// Tests that state transitions as documented don't cause panics
// Tests that the session saves or doesn't save when appropriate
func TestSession_SetNegotiationStatus(t *testing.T) {
	s, _ := makeTestSession()
	//	Normal paths: SetNegotiationStatus should not fail
	// Use timestamps to determine whether a save has occurred
	s.negotiationStatus = Sending
	s.SetNegotiationStatus(Sent)
	if s.negotiationStatus != Sent {
		t.Error("SetNegotiationStatus didn't set the negotiation status")
	}
	object, err := s.kv.Get(sessionKey, 0)
	if err != nil {
		t.Fatal(err)
	}

	s.SetNegotiationStatus(Confirmed)
	if s.negotiationStatus != Confirmed {
		t.Error("SetNegotiationStatus didn't set the negotiation status")
	}
	object, err = s.kv.Get(sessionKey, 0)
	if err != nil {
		t.Fatal(err)
	}

	s.negotiationStatus = NewSessionTriggered
	s.SetNegotiationStatus(NewSessionCreated)
	if s.negotiationStatus != NewSessionCreated {
		t.Error("SetNegotiationStatus didn't set the negotiation status")
	}
	object, err = s.kv.Get(sessionKey, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Reverting paths: SetNegotiationStatus should not fail, and a save should not take place
	s.negotiationStatus = Sending
	s.SetNegotiationStatus(Unconfirmed)
	if s.negotiationStatus != Unconfirmed {
		t.Error("SetNegotiationStatus didn't set the negotiation status")
	}
	newObject, err := s.kv.Get(sessionKey, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(object, newObject) {
		t.Error("save occurred after switching Sent to Confirmed")
	}

	s.negotiationStatus = NewSessionTriggered
	s.SetNegotiationStatus(Confirmed)
	if s.negotiationStatus != Confirmed {
		t.Error("SetNegotiationStatus didn't set the negotiation status")
	}
	newObject, err = s.kv.Get(sessionKey, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(object, newObject) {
		t.Error("save occurred after switching Sent to Confirmed")
	}
}

// Tests that TriggerNegotiation makes only valid state transitions
func TestSession_TriggerNegotiation(t *testing.T) {
	s, _ := makeTestSession()
	// Set up num keys used to be > rekeyThreshold: should partnerSource negotiation
	s.keyState.SetNumAvailableTEST(50, t)
	s.keyState.SetNumKeysTEST(100, t)
	s.rekeyThreshold = 49
	s.negotiationStatus = Confirmed

	if !s.TriggerNegotiation() {
		t.Error("partnerSource negotiation unexpectedly failed")
	}
	if s.negotiationStatus != NewSessionTriggered {
		t.Errorf("negotiationStatus: got %v, expected %v", s.negotiationStatus, NewSessionTriggered)
	}

	// Set up num keys used to be = rekeyThreshold: should partnerSource negotiation
	s.rekeyThreshold = 50
	s.negotiationStatus = Confirmed

	if !s.TriggerNegotiation() {
		t.Error("partnerSource negotiation unexpectedly failed")
	}
	if s.negotiationStatus != NewSessionTriggered {
		t.Errorf("negotiationStatus: got %v, expected %v", s.negotiationStatus, NewSessionTriggered)
	}

	// Set up num keys used to be < rekeyThreshold: shouldn't partnerSource negotiation
	s.rekeyThreshold = 51
	s.negotiationStatus = Confirmed

	if s.TriggerNegotiation() {
		t.Error("trigger negotiation unexpectedly failed")
	}
	if s.negotiationStatus != Confirmed {
		t.Errorf("negotiationStatus: got %s, expected %s", s.negotiationStatus, Confirmed)
	}

	// TODO: this section of the test is rng-based, not good design
	// Test other case: partnerSource sending	confirmation message on unconfirmed session
	//s.negotiationStatus = Unconfirmed
	//if !s.TriggerNegotiation() {
	//	t.Error("partnerSource negotiation unexpectedly failed")
	//}
	//if s.negotiationStatus != Sending {
	//	t.Errorf("negotiationStatus: got %s, expected %s", s.negotiationStatus, Sending)
	//}
}

// Shows that String doesn't cause errors or panics
// Also can be used to examine or change output of String()
func TestSession_String(t *testing.T) {
	s, _ := makeTestSession()
	t.Log(s.String())
}

// Shows that GetSource gets the partnerSource we set
func TestSession_GetTrigger(t *testing.T) {
	s, _ := makeTestSession()
	thisTrigger := s.GetID()
	s.partnerSource = thisTrigger
	if !reflect.DeepEqual(s.GetSource(), thisTrigger) {
		t.Error("Trigger different from expected")
	}
}
