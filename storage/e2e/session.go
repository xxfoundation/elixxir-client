////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
	"time"
)

const currentSessionVersion = 0
const sessionPrefix = "session{ID:%s}"
const sessionKey = "session"

type Session struct {
	//pointer to relationship
	relationship *relationship
	//prefixed kv
	kv *versioned.KV
	//params
	params SessionParams

	//type
	t RelationshipType

	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	myPrivKey *cyclic.Int
	// Partner Public Key
	partnerPubKey *cyclic.Int
	// ID of the session which teh partner public key comes from for this
	// sessions creation.  Shares a partner public key if a Send session,
	// shares a myPrivateKey if a Receive session
	partnerSource SessionID
	//fingerprint of relationship
	relationshipFingerprint []byte

	//denotes if the other party has confirmed this key
	negotiationStatus Negotiation

	// Value of the counter at which a rekey is triggered
	ttl uint32

	// Received Keys dirty bits
	// Each bit represents a single Key
	keyState *stateVector

	//mutex
	mux sync.RWMutex
}

// As this is serialized by json, any field that should be serialized
// must be exported
// Utility struct to write part of session data to disk
type SessionDisk struct {
	Params SessionParams

	//session type
	Type uint8

	// Underlying key
	BaseKey []byte
	// Own Private Key
	MyPrivKey []byte
	// Partner Public Key
	PartnerPubKey []byte

	// Group used by the above keys
	Group []byte

	// ID of the session which triggered this sessions creation.
	Trigger []byte

	//denotes if the other party has confirmed this key
	Confirmation uint8

	// Number of keys usable before rekey
	TTL uint32
}

/*CONSTRUCTORS*/
//Generator which creates all keys and structures
func newSession(ship *relationship, t RelationshipType, myPrivKey, partnerPubKey,
	baseKey *cyclic.Int, trigger SessionID, relationshipFingerprint []byte,
	params SessionParams) *Session {

	confirmation := Unconfirmed
	if t == Receive {
		confirmation = Confirmed
	}

	session := &Session{
		params:                  params,
		relationship:            ship,
		t:                       t,
		myPrivKey:               myPrivKey,
		partnerPubKey:           partnerPubKey,
		baseKey:                 baseKey,
		relationshipFingerprint: relationshipFingerprint,
		negotiationStatus:       confirmation,
		partnerSource:           trigger,
	}

	session.kv = session.generate(ship.kv)

	err := session.save()
	if err != nil {
		jww.FATAL.Printf("Failed to make new session for Partner %s: %s",
			ship.manager.partner, err)
	}

	return session
}

// Load session and state vector from kv and populate runtime fields
func loadSession(ship *relationship, kv *versioned.KV,
	relationshipFingerprint []byte) (*Session, error) {

	session := Session{
		relationship: ship,
		kv:           kv,
	}

	obj, err := kv.Get(sessionKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load %s", kv.GetFullKey(sessionKey))
	}

	err = session.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	if session.t == Receive {
		// register key fingerprints
		ship.manager.ctx.fa.add(session.getUnusedKeys())
	}
	session.relationshipFingerprint = relationshipFingerprint

	return &session, nil
}

func (s *Session) save() error {

	now := time.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentSessionVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(sessionKey, &obj)
}

/*METHODS*/
// Done all unused key fingerprints
// Delete this session and its key states from the storage
func (s *Session) Delete() {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.relationship.manager.ctx.fa.remove(s.getUnusedKeys())

	stateVectorErr := s.keyState.Delete()
	sessionErr := s.kv.Delete(sessionKey)

	if stateVectorErr != nil && sessionErr != nil {
		jww.ERROR.Printf("Error deleting state vector with key %v: %v", stateVectorKey, stateVectorErr.Error())
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if sessionErr != nil {
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if stateVectorErr != nil {
		jww.ERROR.Panicf("Error deleting state vector with key %v: %v", stateVectorKey, stateVectorErr.Error())
	}
}

//Gets the base key.
func (s *Session) GetBaseKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.baseKey.DeepCopy()
}

func (s *Session) GetMyPrivKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.myPrivKey.DeepCopy()
}

func (s *Session) GetPartnerPubKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.partnerPubKey.DeepCopy()
}

func (s *Session) GetSource() SessionID {
	// no lock is needed because this cannot be edited
	return s.partnerSource
}

//underlying definition of session id
func getSessionIDFromBaseKey(baseKey *cyclic.Int) SessionID {
	// no lock is needed because this cannot be edited
	sid := SessionID{}
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	copy(sid[:], h.Sum(nil))
	return sid
}

//underlying definition of session id
// FOR TESTING PURPOSES ONLY
func GetSessionIDFromBaseKeyForTesting(baseKey *cyclic.Int, i interface{}) SessionID {
	switch i.(type) {
	case *testing.T:
		break
	case *testing.M:
		break
	case *testing.B:
		break
	default:
		globals.Log.FATAL.Panicf("GetSessionIDFromBaseKeyForTesting is restricted to testing only. Got %T", i)
	}
	return getSessionIDFromBaseKey(baseKey)
}

//Blake2B hash of base key used for storage
func (s *Session) GetID() SessionID {
	return getSessionIDFromBaseKey(s.baseKey)
}

// returns the ID of the partner for this session
func (s *Session) GetPartner() *id.ID {
	if s.relationship != nil {
		return s.relationship.manager.partner
	} else {
		return nil
	}
}

//ekv functions
func (s *Session) marshal() ([]byte, error) {
	sd := SessionDisk{}

	grp := s.relationship.manager.ctx.grp

	sd.Params = s.params
	sd.Type = uint8(s.t)
	sd.BaseKey = s.baseKey.Bytes()
	sd.MyPrivKey = s.myPrivKey.Bytes()
	sd.PartnerPubKey = s.partnerPubKey.Bytes()
	sd.Trigger = s.partnerSource[:]
	grpBytes, err := grp.GobEncode()
	if err != nil {
		return nil, err
	}
	sd.Group = grpBytes

	// assume in progress confirmations and session creations have failed on
	// reset, therefore do not store their pending progress
	if s.negotiationStatus == Sending {
		sd.Confirmation = uint8(Unconfirmed)
	} else if s.negotiationStatus == NewSessionTriggered {
		sd.Confirmation = uint8(Confirmed)
	} else {
		sd.Confirmation = uint8(s.negotiationStatus)
	}

	sd.TTL = s.ttl

	return json.Marshal(&sd)
}

func (s *Session) unmarshal(b []byte) error {

	sd := SessionDisk{}

	err := json.Unmarshal(b, &sd)

	if err != nil {
		return err
	}

	grp := &cyclic.Group{}
	err = grp.GobDecode(sd.Group)
	if err != nil {
		return err
	}

	s.params = sd.Params
	s.t = RelationshipType(sd.Type)
	s.baseKey = grp.NewIntFromBytes(sd.BaseKey)
	s.myPrivKey = grp.NewIntFromBytes(sd.MyPrivKey)
	s.partnerPubKey = grp.NewIntFromBytes(sd.PartnerPubKey)
	s.negotiationStatus = Negotiation(sd.Confirmation)
	s.ttl = sd.TTL
	copy(s.partnerSource[:], sd.Trigger)

	s.keyState, err = loadStateVector(s.kv, "")
	if err != nil {
		return err
	}

	return nil
}

//key usage
// Pops the first unused key, skipping any which are denoted as used.
// will return if the remaining keys are designated as rekeys
func (s *Session) PopKey() (*Key, error) {
	if s.keyState.GetNumAvailable() <= uint32(s.params.NumRekeys) {
		return nil, errors.New("no more keys left, remaining reserved " +
			"for rekey")
	}
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newKey(s, keyNum), nil
}

func (s *Session) PopReKey() (*Key, error) {
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newKey(s, keyNum), nil
}

func (s *Session) GetRelationshipFingerprint() []byte {
	return s.relationshipFingerprint
}

// returns the state of the session, which denotes if the Session is active,
// functional but in need of a rekey, empty of Send key, or empty of rekeys
func (s *Session) Status() Status {
	// copy the num available so it stays consistent as this function does its
	// checks
	numAvailable := s.keyState.GetNumAvailable()

	if numAvailable == 0 {
		return RekeyEmpty
	} else if numAvailable <= uint32(s.params.NumRekeys) {
		return Empty
		// do not need to make a copy of getNumKeys becasue it is static and
		// only used once
	} else if numAvailable <= s.keyState.GetNumKeys()-s.ttl {
		return RekeyNeeded
	} else {
		return Active
	}
}

// Sets the negotiation status, this tracks the state of the key negotiation,
// only certain movements are allowed
//   Unconfirmed <--> Sending --> Sent --> Confirmed <--> NewSessionTriggered --> NewSessionCreated
//				  -------------->
//
// Saves the session unless the status is sending so that on reload the rekey
// will be redone if it was in the process of sending

// Moving from Unconfirmed to Sending and from Confirmed to NewSessionTriggered
// is handled by  Session.triggerNegotiation() which is called by the
// Manager as part of Manager.TriggerNegotiations() and will be rejected
// from this function

var legalStateChanges = [][]bool{
	{false, false, false, false, false, false},
	{true, false, true, true, false, false},
	{false, false, false, true, false, false},
	{false, false, false, false, true, false},
	{false, false, false, true, false, true},
	{false, false, false, false, false, false},
}

func (s *Session) SetNegotiationStatus(status Negotiation) {
	if err := s.TrySetNegotiationStatus(status); err != nil {
		jww.FATAL.Panicf("Failed to set Negotiation status: %s", err)
	}
}

func (s *Session) TrySetNegotiationStatus(status Negotiation) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	//only allow the correct state changes to propagate
	if !legalStateChanges[s.negotiationStatus][status] {
		return errors.Errorf("Negotiation status change from %s to %s "+
			"is not valid", s.negotiationStatus, status)
	}

	// the states of Sending and NewSessionTriggered are not saved to disk when
	// moved from Unconfirmed or Confirmed respectively so the actions are
	// re-triggered if there is a crash and reload. As a result, a save when
	// reverting states is unnecessary
	save := !((s.negotiationStatus == Sending && status == Unconfirmed) ||
		(s.negotiationStatus == NewSessionTriggered && status == Confirmed))

	//change the state
	oldStatus := s.negotiationStatus
	s.negotiationStatus = status

	//save the status if appropriate
	if save {
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to save Session %s when moving from %s to %s", s, oldStatus, status)
		}
	}

	return nil
}

// This function, in a mostly thread safe manner, checks if the session needs a
// negotiation, returns if it does while updating the session to denote the
// negotiation was triggered
// WARNING: This function relies on proper action by the caller for data safety.
// When triggering the creation of a new session (the first case) it does not
// store to disk the fact that it has triggered the session. This is because
// every session should only partnerSource one other session and in the event that
// session partnerSource does not resolve before a crash, by not storing it the
// partnerSource will automatically happen again when reloading after the crash.
// In order to ensure the session creation is not triggered again after the
// reload, it is the responsibility of the caller to call
// Session.SetConfirmationStatus(NewSessionCreated) .
func (s *Session) triggerNegotiation() bool {
	// Due to the fact that a read lock cannot be transitioned to a
	// write lock, the state checks need to happen a second time because it
	// is possible for another thread to take the read lock and update the
	// state between this thread releasing it and regaining it again. In this
	// case, such double locking is preferable because the majority of the time,
	// the checked cases will turn out to be false.
	s.mux.RLock()
	// If we've used more keys than the TTL, it's time for a rekey
	if s.keyState.GetNumUsed() >= s.ttl && s.negotiationStatus == Confirmed {
		s.mux.RUnlock()
		s.mux.Lock()
		if s.keyState.GetNumUsed() >= s.ttl && s.negotiationStatus == Confirmed {
			//partnerSource a rekey to create a new session
			s.negotiationStatus = NewSessionTriggered
			// no save is make after the update because we do not want this state
			// saved to disk. The caller will shortly execute the operation,
			// and then move to the next state. If a crash occurs before, by not
			// storing this state this operation will be repeated after reload
			// The save function has been modified so if another call causes a
			// save, "NewSessionTriggered" will be overwritten with "Confirmed"
			// in the saved data.
			s.mux.Unlock()
			return true
		} else {
			s.mux.Unlock()
			return false
		}
		// fixme: Is this a bug? In rekey.go, it seems a session would never be unconfirmed
		//  as it would be set to sending. Possible that this is wrong or the switch statement is
	} else if s.negotiationStatus == Unconfirmed {
		// retrigger this sessions negotiation
		s.mux.RUnlock()
		s.mux.Lock()
		if s.negotiationStatus == Unconfirmed {
			s.negotiationStatus = Sending
			// no save is made after the update because we do not want this state
			// saved to disk. The caller will shortly execute the operation,
			// and then move to the next state. If a crash occurs before, by not
			// storing this state this operation will be repeated after reload
			// The save function has been modified so if another call causes a
			// save, "Sending" will be overwritten with "Unconfirmed"
			// in the saved data.
			s.mux.Unlock()
			return true
		} else {
			s.mux.Unlock()
			return false
		}
	}
	s.mux.RUnlock()
	return false
}

// checks if the session has been confirmed
func (s *Session) NegotiationStatus() Negotiation {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.negotiationStatus
}

// checks if the session has been confirmed
func (s *Session) IsConfirmed() bool {
	c := s.NegotiationStatus()
	//fmt.Println(c)
	return c >= Confirmed
}

func (s *Session) String() string {
	partner := s.GetPartner()
	if partner != nil {
		return fmt.Sprintf("{Partner: %s, ID: %s}",
			partner, s.GetID())
	} else {
		return fmt.Sprintf("{Partner: nil, ID: %s}", s.GetID())
	}
}

/*PRIVATE*/
func (s *Session) useKey(keynum uint32) {
	s.keyState.Use(keynum)
}

// generates keys from the base data stored in the session object.
// myPrivKey will be generated if not present
func (s *Session) generate(kv *versioned.KV) *versioned.KV {
	grp := s.relationship.manager.ctx.grp

	//generate private key if it is not present
	if s.myPrivKey == nil {
		stream := s.relationship.manager.ctx.rng.GetStream()
		s.myPrivKey = dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp,
			stream)
		stream.Close()
	}

	// compute the base key if it is not already there
	if s.baseKey == nil {
		s.baseKey = dh.GenerateSessionKey(s.myPrivKey, s.partnerPubKey, grp)
	}

	kv = kv.Prefix(makeSessionPrefix(s.GetID()))

	//generate ttl and keying info
	keysTTL, numKeys := e2e.GenerateKeyTTL(s.baseKey.GetLargeInt(),
		s.params.MinKeys, s.params.MaxKeys, s.params.TTLParams)

	//ensure that enough keys are remaining to rekey
	if numKeys-uint32(keysTTL) < uint32(s.params.NumRekeys) {
		numKeys = uint32(keysTTL + s.params.NumRekeys)
	}

	s.ttl = uint32(keysTTL)

	//create the new state vectors. This will cause disk operations storing them

	// To generate the state vector key correctly,
	// basekey must be computed as the session ID is the hash of basekey
	var err error
	s.keyState, err = newStateVector(kv, "", numKeys)
	if err != nil {
		jww.FATAL.Printf("Failed key generation: %s", err)
	}

	//register keys for reception if this is a reception session
	if s.t == Receive {
		//register keys
		s.relationship.manager.ctx.fa.add(s.getUnusedKeys())
	}

	return kv
}

//returns key objects for all unused keys
func (s *Session) getUnusedKeys() []*Key {
	keyNums := s.keyState.GetUnusedKeyNums()

	keys := make([]*Key, len(keyNums))
	for i, keyNum := range keyNums {
		keys[i] = newKey(s, keyNum)
	}

	return keys
}

//builds the
func makeSessionPrefix(sid SessionID) string {
	return fmt.Sprintf(sessionPrefix, sid)
}
