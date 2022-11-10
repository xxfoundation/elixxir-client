////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sync"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/storage/utility"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentSessionVersion = 0
const sessionPrefix = "session{ID:%s}"
const sessionKey = "session"

type Session struct {
	//prefixed kv
	kv *versioned.KV
	//params
	e2eParams Params

	sID SessionID

	partner *id.ID

	//type
	t RelationshipType

	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	myPrivKey *cyclic.Int
	// Partner Public Key
	partnerPubKey *cyclic.Int

	// SIDH Keys of the same
	mySIDHPrivKey     *sidh.PrivateKey
	partnerSIDHPubKey *sidh.PublicKey

	// ID of the session which teh partner public key comes from for this
	// sessions creation.  Shares a partner public key if a Send session,
	// shares a myPrivateKey if a Receive session
	partnerSource SessionID
	//fingerprint of relationship
	relationshipFingerprint []byte

	//denotes if the other party has confirmed this key
	negotiationStatus Negotiation

	// Number of keys used before the system attempts a rekey
	rekeyThreshold uint32

	// Received Keys dirty bits
	// Each bit represents a single Key
	keyState *utility.StateVector

	//mutex
	mux sync.RWMutex

	//interfaces
	cyHandler CypherHandler
	grp       *cyclic.Group
	rng       *fastRNG.StreamGenerator
}

// SessionDisk is a utility struct to write part of session data to disk.
// As this is serialized by json, any field that should be serialized
// must be exported
type SessionDisk struct {
	E2EParams Params

	//session type
	Type uint8

	// Underlying key
	BaseKey []byte
	// Own Private Key
	MyPrivKey []byte
	// Partner Public Key
	PartnerPubKey []byte
	// Own SIDH Private Key
	MySIDHPrivKey []byte
	// Note: only 3 bit patterns: 001, 010, 100
	MySIDHVariant byte
	// Partner SIDH Public Key
	PartnerSIDHPubKey []byte
	// Note: only 3 bit patterns: 001, 010, 100
	PartnerSIDHVariant byte

	// ID of the session which triggered this sessions creation.
	Trigger []byte
	// relationship fp
	RelationshipFingerprint []byte

	//denotes if the other party has confirmed this key
	Confirmation uint8

	// Number of keys usable before rekey
	RekeyThreshold uint32

	Partner []byte
}

/*CONSTRUCTORS*/

//NewSession - Generator which creates all keys and structures
func NewSession(kv *versioned.KV, t RelationshipType, partner *id.ID, myPrivKey,
	partnerPubKey, baseKey *cyclic.Int, mySIDHPrivKey *sidh.PrivateKey,
	partnerSIDHPubKey *sidh.PublicKey, trigger SessionID,
	relationshipFingerprint []byte, negotiationStatus Negotiation,
	e2eParams Params, cyHandler CypherHandler, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) *Session {

	if e2eParams.MinKeys < 10 {
		jww.FATAL.Panicf("Cannot create a session with a minimum "+
			"number of keys (%d) less than 10", e2eParams.MinKeys)
	}

	session := &Session{
		e2eParams:               e2eParams,
		t:                       t,
		myPrivKey:               myPrivKey,
		partnerPubKey:           partnerPubKey,
		mySIDHPrivKey:           mySIDHPrivKey,
		partnerSIDHPubKey:       partnerSIDHPubKey,
		baseKey:                 baseKey,
		relationshipFingerprint: relationshipFingerprint,
		negotiationStatus:       negotiationStatus,
		partnerSource:           trigger,
		partner:                 partner,
		cyHandler:               cyHandler,
		grp:                     grp,
		rng:                     rng,
	}

	session.finalizeKeyNegotiation()
	session.kv = kv.Prefix(MakeSessionPrefix(session.sID))
	session.buildChildKeys()

	myPubKey := dh.GeneratePublicKey(session.myPrivKey, grp)

	jww.INFO.Printf("New Session with Partner %s:\n\tType: %s"+
		"\n\tBaseKey: %s\n\tRelationship Fingerprint: %v\n\tNumKeys: %d"+
		"\n\tMy Public Key: %s\n\tPartner Public Key: %s"+
		"\n\tMy Public SIDH: %s\n\tPartner Public SIDH: %s",
		partner,
		t,
		session.baseKey.TextVerbose(16, 0),
		session.relationshipFingerprint,
		session.rekeyThreshold,
		myPubKey.TextVerbose(16, 0),
		session.partnerPubKey.TextVerbose(16, 0),
		utility.StringSIDHPrivKey(session.mySIDHPrivKey),
		utility.StringSIDHPubKey(session.partnerSIDHPubKey))

	err := session.Save()
	if err != nil {
		jww.FATAL.Printf("Failed to make new session for Partner %s: %s",
			partner, err)
	}

	return session
}

// LoadSession and state vector from kv and populate runtime fields
func LoadSession(kv *versioned.KV, sessionID SessionID,
	relationshipFingerprint []byte, cyHandler CypherHandler,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator) (*Session, error) {

	session := Session{
		kv:        kv.Prefix(MakeSessionPrefix(sessionID)),
		sID:       sessionID,
		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
	}

	obj, err := session.kv.Get(sessionKey, currentSessionVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load %s",
			session.kv.GetFullKey(sessionKey, currentSessionVersion))
	}

	// TODO: Not necessary until we have versions on this object...
	//obj, err := sessionUpgradeTable.Upgrade(obj)

	err = session.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	if session.t == Receive {
		// register key fingerprints
		for _, cy := range session.getUnusedKeys() {
			cyHandler.AddKey(cy)
		}
	}
	session.relationshipFingerprint = relationshipFingerprint

	return &session, nil
}

// todo - doscstring
func (s *Session) Save() error {

	now := netTime.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentSessionVersion,
		Timestamp: now,
		Data:      data,
	}

	jww.WARN.Printf("saving with KV: %v", s.kv)

	return s.kv.Set(sessionKey, &obj)
}

/*METHODS*/
// Done all unused key fingerprints

// Delete removes this session and its key states from the storage
func (s *Session) Delete() {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, cy := range s.getUnusedKeys() {
		s.cyHandler.DeleteKey(cy)
	}

	stateVectorErr := s.keyState.Delete()
	sessionErr := s.kv.Delete(sessionKey, currentSessionVersion)

	if stateVectorErr != nil && sessionErr != nil {
		jww.ERROR.Printf("Error deleting state vector %s: %v", s.keyState, stateVectorErr.Error())
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if sessionErr != nil {
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if stateVectorErr != nil {
		jww.ERROR.Panicf("Error deleting state vector %s: %v", s.keyState, stateVectorErr.Error())
	}
}

// GetBaseKey retrieves the base key.
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

func (s *Session) GetMySIDHPrivKey() *sidh.PrivateKey {
	// no lock is needed because this should never be edited
	return s.mySIDHPrivKey
}

func (s *Session) GetPartnerSIDHPubKey() *sidh.PublicKey {
	// no lock is needed because this should never be edited
	return s.partnerSIDHPubKey
}

func (s *Session) GetSource() SessionID {
	// no lock is needed because this cannot be edited
	return s.partnerSource
}

//underlying definition of session id
// FOR TESTING PURPOSES ONLY
func GetSessionIDFromBaseKeyForTesting(baseKey *cyclic.Int, i interface{}) SessionID {
	switch i.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("GetSessionIDFromBaseKeyForTesting is restricted to testing only. Got %T", i)
	}
	return GetSessionIDFromBaseKey(baseKey)
}

// GetID Blake2B hash of base key used for storage
func (s *Session) GetID() SessionID {
	return s.sID
}

// GetPartner returns the ID of the partner for this session
func (s *Session) GetPartner() *id.ID {
	if s.partner != nil {
		return s.partner
	} else {
		return nil
	}
}

//key usage

// PopKey Pops the first unused key, skipping any which are denoted as used.
// will return if the remaining keys are designated as rekeys
func (s *Session) PopKey() (Cypher, error) {
	if s.keyState.GetNumAvailable() <= uint32(s.e2eParams.NumRekeys) {
		return nil, errors.New("no more keys left, remaining reserved " +
			"for rekey")
	}
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newCypher(s, keyNum), nil
}

// PopReKey Pops the first unused key, skipping any which are denoted as used,
// including keys designated for rekeys
func (s *Session) PopReKey() (Cypher, error) {
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newCypher(s, keyNum), nil
}

// todo - doscstring
func (s *Session) GetRelationshipFingerprint() []byte {
	return s.relationshipFingerprint
}

// returns the state of the session, which denotes if the Session is active,
// functional but in need of a rekey, empty of Send key, or empty of rekeys
func (s *Session) Status() Status {
	// copy the num available so it stays consistent as this function does its
	// checks
	numAvailable := s.keyState.GetNumAvailable()
	numUsed := s.keyState.GetNumUsed()

	if numAvailable == 0 {
		return RekeyEmpty
	} else if numAvailable <= uint32(s.e2eParams.NumRekeys) {
		return Empty
		// do not need to make a copy of getNumKeys becasue it is static and
		// only used once
	} else if numUsed >= s.rekeyThreshold {
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
// is handled by  Session.TriggerNegotiation() which is called by the
// Manager as part of Manager.TriggerNegotiations() and will be rejected
// from this function

var legalStateChanges = [][]bool{
	// Unconf  Sending  Sent   Confi  NewTrig  NewCreat
	{false, true, false, false, false, false}, // Unc
	{true, false, true, true, false, false},   // Sending
	{false, false, false, true, false, false}, // Sent
	{false, false, false, false, true, false}, // Confi
	{false, false, false, true, false, true},  // NewTrig
	{false, false, true, false, false, false}, // NewCreat
}

// todo - doscstring
func (s *Session) SetNegotiationStatus(status Negotiation) {
	if err := s.TrySetNegotiationStatus(status); err != nil {
		jww.FATAL.Panicf("Failed to set Negotiation status: %s", err)
	}
}

// todo - doscstring
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
		if err := s.Save(); err != nil {
			jww.FATAL.Panicf("Failed to save Session %s when moving from %s to %s", s, oldStatus, status)
		}
	}

	return nil
}

// TriggerNegotiation in a mostly thread safe manner, checks if the session needs a
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
func (s *Session) TriggerNegotiation() bool {
	// Due to the fact that a read lock cannot be transitioned to a
	// write lock, the state checks need to happen a second time because it
	// is possible for another thread to take the read lock and update the
	// state between this thread releasing it and regaining it again. In this
	// case, such double locking is preferable because the majority of the time,
	// the checked cases will turn out to be false.
	s.mux.RLock()
	// If we've used more keys than the RekeyThreshold, it's time for a rekey
	if s.keyState.GetNumUsed() >= s.rekeyThreshold &&
		s.negotiationStatus < NewSessionTriggered {
		s.mux.RUnlock()
		s.mux.Lock()
		if s.keyState.GetNumUsed() >= s.rekeyThreshold &&
			s.negotiationStatus < NewSessionTriggered {
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
	} else if s.negotiationStatus == Unconfirmed && decideIfResendRekey(s.rng,
		s.e2eParams.UnconfirmedRetryRatio) {
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

// NegotiationStatus checks if the session has been confirmed
func (s *Session) NegotiationStatus() Negotiation {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.negotiationStatus
}

// IsConfirmed checks if the session has been confirmed
func (s *Session) IsConfirmed() bool {
	c := s.NegotiationStatus()
	return c >= Confirmed
}

// todo - doscstring
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

// todo - doscstring
// finalizeKeyNegotiation generates keys from the base data stored in the session object.
// myPrivKey will be generated if not present
func (s *Session) finalizeKeyNegotiation() {
	grp := s.grp

	//Generates private key if it is not present
	if s.myPrivKey == nil {
		stream := s.rng.GetStream()
		s.myPrivKey = dh.GeneratePrivateKey(len(grp.GetPBytes()),
			grp, stream)
		// get the variant opposite my partners variant
		sidhVariant := utility.GetCompatibleSIDHVariant(
			s.partnerSIDHPubKey.Variant())
		s.mySIDHPrivKey = utility.NewSIDHPrivateKey(sidhVariant)
		s.mySIDHPrivKey.Generate(stream)
		stream.Close()
	}

	// compute the base key if it is not already there
	if s.baseKey == nil {
		s.baseKey = GenerateE2ESessionBaseKey(s.myPrivKey,
			s.partnerPubKey, grp, s.mySIDHPrivKey,
			s.partnerSIDHPubKey)
	}

	s.sID = GetSessionIDFromBaseKey(s.baseKey)
}

// todo - doscstring
func (s *Session) buildChildKeys() {
	p := s.e2eParams
	h, _ := hash.NewCMixHash()

	//generates rekeyThreshold and keying info
	numKeys := uint32(randomness.RandInInterval(big.NewInt(
		int64(p.MaxKeys-p.MinKeys)),
		s.baseKey.Bytes(), h).Int64() + int64(p.MinKeys))

	// start rekeying when enough keys have been used
	s.rekeyThreshold = uint32(math.Ceil(s.e2eParams.RekeyThreshold * float64(numKeys)))

	// the total number of keys should be the number of rekeys plus the
	// number of keys to use
	numKeys = numKeys + uint32(s.e2eParams.NumRekeys)

	// create the new state vectors. This will cause disk operations
	// storing them

	// To generate the state vector key correctly,
	// basekey must be computed as the session ID is the hash of basekey
	var err error
	s.keyState, err = utility.NewStateVector(s.kv, "", numKeys)
	if err != nil {
		jww.FATAL.Printf("Failed key generation: %s", err)
	}

	//register keys for reception if this is a reception session
	if s.t == Receive {
		for _, cy := range s.getUnusedKeys() {
			s.cyHandler.AddKey(cy)
		}
	}
}

//returns key objects for all unused keys
func (s *Session) getUnusedKeys() []Cypher {
	keyNums := s.keyState.GetUnusedKeyNums()

	keys := make([]Cypher, len(keyNums))
	for i, keyNum := range keyNums {
		keys[i] = newCypher(s, keyNum)
	}

	return keys
}

//ekv functions
func (s *Session) marshal() ([]byte, error) {
	sd := SessionDisk{}

	sd.E2EParams = s.e2eParams
	sd.Type = uint8(s.t)
	sd.BaseKey = s.baseKey.Bytes()
	sd.MyPrivKey = s.myPrivKey.Bytes()
	sd.PartnerPubKey = s.partnerPubKey.Bytes()
	sd.MySIDHPrivKey = make([]byte, s.mySIDHPrivKey.Size())
	sd.PartnerSIDHPubKey = make([]byte, s.partnerSIDHPubKey.Size())

	s.mySIDHPrivKey.Export(sd.MySIDHPrivKey)
	sd.MySIDHVariant = byte(s.mySIDHPrivKey.Variant())

	s.partnerSIDHPubKey.Export(sd.PartnerSIDHPubKey)
	sd.PartnerSIDHVariant = byte(s.partnerSIDHPubKey.Variant())

	sd.Trigger = s.partnerSource[:]
	sd.RelationshipFingerprint = s.relationshipFingerprint
	sd.Partner = s.partner.Bytes()

	// assume in progress confirmations and session creations have failed on
	// reset, therefore do not store their pending progress
	if s.negotiationStatus == Sending {
		sd.Confirmation = uint8(Unconfirmed)
	} else if s.negotiationStatus == NewSessionTriggered {
		sd.Confirmation = uint8(Confirmed)
	} else {
		sd.Confirmation = uint8(s.negotiationStatus)
	}

	sd.RekeyThreshold = s.rekeyThreshold

	return json.Marshal(&sd)
}

func (s *Session) unmarshal(b []byte) error {

	sd := SessionDisk{}

	err := json.Unmarshal(b, &sd)

	if err != nil {
		return err
	}

	grp := s.grp

	s.e2eParams = sd.E2EParams
	s.t = RelationshipType(sd.Type)
	s.baseKey = grp.NewIntFromBytes(sd.BaseKey)
	s.myPrivKey = grp.NewIntFromBytes(sd.MyPrivKey)
	s.partnerPubKey = grp.NewIntFromBytes(sd.PartnerPubKey)

	mySIDHVariant := sidh.KeyVariant(sd.MySIDHVariant)
	s.mySIDHPrivKey = utility.NewSIDHPrivateKey(mySIDHVariant)
	err = s.mySIDHPrivKey.Import(sd.MySIDHPrivKey)
	if err != nil {
		return err
	}

	partnerSIDHVariant := sidh.KeyVariant(sd.PartnerSIDHVariant)
	s.partnerSIDHPubKey = utility.NewSIDHPublicKey(partnerSIDHVariant)
	err = s.partnerSIDHPubKey.Import(sd.PartnerSIDHPubKey)
	if err != nil {
		return err
	}

	s.negotiationStatus = Negotiation(sd.Confirmation)
	s.rekeyThreshold = sd.RekeyThreshold
	s.relationshipFingerprint = sd.RelationshipFingerprint
	s.partner, _ = id.Unmarshal(sd.Partner)
	copy(s.partnerSource[:], sd.Trigger)

	s.keyState, err = utility.LoadStateVector(s.kv, "")
	if err != nil {
		return err
	}

	return nil
}

// MakeSessionPrefix builds the prefix
func MakeSessionPrefix(sid SessionID) string {
	return fmt.Sprintf(sessionPrefix, sid)
}

// todo - doscstring
func decideIfResendRekey(genRng *fastRNG.StreamGenerator, ratio float64) bool {
	stream := genRng.GetStream()
	b := make([]byte, 4)
	stream.Read(b)
	stream.Close()
	randNum := binary.BigEndian.Uint32(b)
	max := uint32(float64(math.MaxUint32) * ratio)
	return randNum < max
}
