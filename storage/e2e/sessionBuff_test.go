package e2e

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Subtest: unmarshal/marshal with one session in the buff
func TestSessionBuff_MarshalUnmarshal(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	g := mgr.ctx.grp
	session := newSession(mgr, g.NewInt(2), g.NewInt(3), g.NewInt(4), GetDefaultSessionParams(),
		Receive, SessionID{})
	sb.AddSession(session)

	// Serialization should include session slice only
	serialized, err := sb.marshal()
	if err != nil {
		t.Fatal(err)
	}

	sb2 := NewSessionBuff(mgr, "test")
	err = sb2.unmarshal(serialized)
	if err != nil {
		t.Fatal(err)
	}

	// compare sb2 sesh list and map
	if !sessionBuffsEqual(sb, sb2) {
		t.Error("session buffs not equal")
	}
}

// Shows that LoadSessionBuff returns an equivalent session buff to the one that was saved
func TestLoadSessionBuff(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	g := mgr.ctx.grp
	session := newSession(mgr, g.NewInt(2), g.NewInt(3), g.NewInt(4), GetDefaultSessionParams(),
		Receive, SessionID{})
	sb.AddSession(session)
	err := sb.save()
	if err != nil {
		t.Fatal(err)
	}

	sb2, err := LoadSessionBuff(mgr, "test")
	if err != nil {
		t.Fatal(err)
	}

	if !sessionBuffsEqual(sb, sb2) {
		t.Error("session buffers not equal")
	}
}

// Shows that NewSessionBuff returns a valid session buff
func TestNewSessionBuff(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	if mgr != sb.manager {
		t.Error("managers should be identical")
	}
	if sb.sessionByID == nil || len(sb.sessionByID) != 0 {
		t.Error("session map should not be nil, and should be empty")
	}
	if sb.sessions == nil || len(sb.sessions) != 0 {
		t.Error("session list should not be nil, and should be empty")
	}
}

// Shows that AddSession adds one session to the buff
func TestSessionBuff_AddSession(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	if len(sb.sessions) != 0 {
		t.Error("starting session slice length should be 0")
	}
	if len(sb.sessionByID) != 0 {
		t.Error("starting session map length should be 0")
	}
	session, _ := makeTestSession()
	// Note: AddSession doesn't change the session manager or set anything else up
	// to match the session to the session buffer. To work properly, the session
	// should have been created using the same manager (which is not the case in
	// this test.)
	sb.AddSession(session)
	if len(sb.sessions) != 1 {
		t.Error("ending session slice length should be 1")
	}
	if len(sb.sessionByID) != 1 {
		t.Error("ending session map length should be 1")
	}
	if session.GetID() != sb.sessions[0].GetID() {
		t.Error("session added should have same ID")
	}
}

// GetNewest should get the session that was most recently added to the buff
func TestSessionBuff_GetNewest(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	// The newest session should be nil upon session buffer creation
	nilSession := sb.GetNewest()
	if nilSession != nil {
		t.Error("should have gotten a nil session from a buffer with no sessions")
	}

	session, _ := makeTestSession()
	sb.AddSession(session)
	if session.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

	session2, _ := makeTestSession()
	sb.AddSession(session2)
	if session2.GetID() != sb.GetNewest().GetID() {
		t.Error("session added should have same ID")
	}

}

// Shows that Confirm confirms the specified session in the buff
func TestSessionBuff_Confirm(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	session, _ := makeTestSession()
	session.negotiationStatus = Sent
	adaptToBuff(session, sb, t)
	sb.AddSession(session)

	if session.IsConfirmed() {
		t.Error("session should not be confirmed before confirmation")
	}

	err := sb.Confirm(session.GetID())
	if err != nil {
		t.Fatal(err)
	}

	if !session.IsConfirmed() {
		t.Error("session should be confirmed after confirmation")
	}
}

// Shows that the session buff returns an error when the session doesn't exist
func TestSessionBuff_Confirm_Err(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	session, _ := makeTestSession()

	err := sb.Confirm(session.GetID())
	if err == nil {
		t.Error("Confirming a session not in the buff should result in an error")
	}
}

// Shows that a session can get got by ID from the buff
func TestSessionBuff_GetByID(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	session, _ := makeTestSession()
	sb.AddSession(session)
	session2 := sb.GetByID(session.GetID())
	if !reflect.DeepEqual(session, session2) {
		t.Error("gotten session should be the same")
	}
}

// Shows that GetNewestRekeyableSession acts as expected:
// returning sessions that are confirmed and past ttl
func TestSessionBuff_GetNewestRekeyableSession(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	// no available rekeyable sessions: nil
	session2 := sb.getNewestRekeyableSession()
	if session2 != nil {
		t.Error("newest rekeyable session should be nil")
	}

	// add a rekeyable session: that session
	session, _ := makeTestSession()
	sb.AddSession(session)
	session3 := sb.getNewestRekeyableSession()
	if session3.GetID() != session.GetID() {
		t.Error("didn't get the expected session")
	}

	// add another rekeyable session: that session
	additionalSession, _ := makeTestSession()
	sb.AddSession(additionalSession)
	session4 := sb.getNewestRekeyableSession()
	if session4.GetID() != additionalSession.GetID() {
		t.Error("didn't get the expected session")
	}

	// make the very newest session unrekeyable: the previous session
	additionalSession.negotiationStatus = Unconfirmed
	session5 := sb.getNewestRekeyableSession()
	if session5.GetID() != session.GetID() {
		t.Error("didn't get the expected session")
	}
}

// Shows that GetSessionForSending follows the hierarchy of sessions correctly
func TestSessionBuff_GetSessionForSending(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	none := sb.getSessionForSending()
	if none != nil {
		t.Error("getSessionForSending should return nil if there aren't any sendable sessions")
	}

	// First case: unconfirmed rekey
	unconfirmedRekey, _ := makeTestSession()
	unconfirmedRekey.negotiationStatus = Unconfirmed
	unconfirmedRekey.keyState.numAvailable = 600
	t.Log(unconfirmedRekey.Status())
	sb.AddSession(unconfirmedRekey)
	sending := sb.getSessionForSending()
	if sending.GetID() != unconfirmedRekey.GetID() {
		t.Error("got an unexpected session")
	}

	// Second case: unconfirmed active
	unconfirmedActive, _ := makeTestSession()
	unconfirmedActive.negotiationStatus = Unconfirmed
	unconfirmedActive.keyState.numAvailable = 2000
	t.Log(unconfirmedActive.Status())
	sb.AddSession(unconfirmedActive)
	sending = sb.getSessionForSending()
	if sending.GetID() != unconfirmedActive.GetID() {
		t.Error("got an unexpected session")
	}

	// Third case: confirmed rekey
	confirmedRekey, _ := makeTestSession()
	confirmedRekey.keyState.numAvailable = 600
	t.Log(confirmedRekey.Status())
	sb.AddSession(confirmedRekey)
	sending = sb.getSessionForSending()
	if sending.GetID() != confirmedRekey.GetID() {
		t.Error("got an unexpected session")
	}

	// Fourth case: confirmed active
	confirmedActive, _ := makeTestSession()
	confirmedActive.keyState.numAvailable = 2000
	t.Log(confirmedActive.Status())
	sb.AddSession(confirmedActive)
	sending = sb.getSessionForSending()
	if sending.GetID() != confirmedActive.GetID() {
		t.Error("got an unexpected session")
	}
}

// Shows that GetKeyForRekey returns a key if there's an appropriate session for rekeying
func TestSessionBuff_GetKeyForRekey(t *testing.T) {
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	// no available rekeyable sessions: error
	key, err := sb.getKeyForRekey()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := makeTestSession()
	sb.AddSession(session)
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
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")

	// no available rekeyable sessions: error
	key, err := sb.getKeyForSending()
	if err == nil {
		t.Error("should have returned an error with no sessions available")
	}
	if key != nil {
		t.Error("shouldn't have returned a key with no sessions available")
	}

	session, _ := makeTestSession()
	sb.AddSession(session)
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
	mgr := makeTestSessionBuffManager(t)
	sb := NewSessionBuff(mgr, "test")
	session, _ := makeTestSession()
	sb.AddSession(session)
	// The added session isn't ready for rekey so it's not returned here
	negotiations := sb.TriggerNegotiation()
	if len(negotiations) != 0 {
		t.Error("should have had zero negotiations")
	}
	session2, _ := makeTestSession()
	// Make only a few keys available to trigger the ttl
	session2.keyState.numAvailable = 4
	sb.AddSession(session2)
	negotiations = sb.TriggerNegotiation()
	if len(negotiations) != 1 {
		t.Fatal("should have had one negotiation")
	}
	if negotiations[0].GetID() != session2.GetID() {
		t.Error("negotiated sessions should include the rekeyable session")
	}
	if session2.negotiationStatus != NewSessionTriggered {
		t.Error("Trigger negotiations should have set status to triggered")
	}

	// Unconfirmed sessions should also be included in the list
	// as the client should attempt to confirm them
	session3, _ := makeTestSession()
	session3.negotiationStatus = Unconfirmed
	sb.AddSession(session3)

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

func makeTestSessionBuff(t *testing.T) *sessionBuff {
	sb := &sessionBuff{
		manager:     makeTestSessionBuffManager(t),
		sessions:    make([]*Session, 0),
		sessionByID: make(map[SessionID]*Session),
		key:         "test",
	}
	sb.kv = sb.manager.kv
	return sb
}

func makeTestSessionBuffManager(t *testing.T) *Manager {
	fps := newFingerprints()
	return &Manager{
		ctx: &context{
			fa:  &fps,
			grp: getGroup(),
		},
		kv:      versioned.NewKV(make(ekv.Memstore)),
		partner: id.NewIdFromUInt(8, id.User, t),
	}
}

// Revises a session to fit a sessionbuff and saves it to the sessionbuff's kv store
func adaptToBuff(session *Session, buff *sessionBuff, t *testing.T) {
	session.manager = buff.manager
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
func sessionBuffsEqual(buff *sessionBuff, buff2 *sessionBuff) bool {
	if buff.key != buff2.key {
		return false
	}
	if len(buff.sessionByID) != len(buff2.sessionByID) {
		return false
	}
	if len(buff.sessions) != len(buff2.sessions) {
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
