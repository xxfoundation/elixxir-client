////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"encoding/json"
	"sync"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentRelationshipLegacySIDHVersion = 0
const currentRelationshipLegacySIDHFingerprintVersion = 0
const relationshipLegacySIDHKey = "relationshipLegacySIDH"
const relationshipLegacySIDHFingerprintKey = "relationshipLegacySIDHFingerprint"

type relationshipLegacySIDH struct {
	t session.RelationshipLegacySIDHType

	kv *versioned.KV

	sessions    []*session.Session
	sessionByID map[session.SessionID]*session.Session

	fingerprint []byte

	mux     sync.RWMutex
	sendMux sync.Mutex

	grp       *cyclic.Group
	myID      *id.ID
	partnerID *id.ID

	cyHandler session.CypherHandler
	rng       *fastRNG.StreamGenerator

	serviceHandler ServiceHandler
}

type ServiceHandler interface {
	AddService(identifier []byte, tag string, source []byte)
	DeleteKey(identifier []byte, tag string)
}

// fixme - this is weird becasue it creates the relationsip and the session.
// Should be refactored to create an empty relationshipLegacySIDH, with a second call
// adding the session
// todo - doscstring
func NewRelationshipLegacySIDH(kv *versioned.KV, t session.RelationshipLegacySIDHType,
	myID, partnerID *id.ID, myOriginPrivateKey,
	partnerOriginPublicKey *cyclic.Int, originMySIDHPrivKey *sidh.PrivateKey,
	originPartnerSIDHPubKey *sidh.PublicKey, initialParams session.Params,
	cyHandler session.CypherHandler, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) *relationshipLegacySIDH {

	kv = kv.Prefix(t.Prefix())

	fingerprint := makeRelationshipLegacySIDHFingerprint(t, grp,
		myOriginPrivateKey, partnerOriginPublicKey, myID,
		partnerID)

	if err := storeRelationshipLegacySIDHFingerprint(fingerprint, kv); err != nil {
		jww.FATAL.Panicf("Failed to store relationshipLegacySIDH fingerpint "+
			"for new relationshipLegacySIDH: %+v", err)
	}

	r := &relationshipLegacySIDH{
		t:           t,
		kv:          kv,
		sessions:    make([]*session.Session, 0),
		sessionByID: make(map[session.SessionID]*session.Session),
		fingerprint: fingerprint,
		grp:         grp,
		cyHandler:   cyHandler,
		myID:        myID,
		partnerID:   partnerID,
		rng:         rng,
	}

	// set to confirmed because the first session is always confirmed as a
	// result of the negotiation before creation
	s := session.NewSession(r.kv, r.t, partnerID, myOriginPrivateKey,
		partnerOriginPublicKey, nil, originMySIDHPrivKey,
		originPartnerSIDHPubKey, session.SessionID{},
		r.fingerprint, session.Confirmed, initialParams, cyHandler,
		grp, rng)

	if err := s.Save(); err != nil {
		jww.FATAL.Panicf("Failed to Send session after setting to "+
			"confirmed: %+v", err)
	}

	r.addSession(s)

	if err := r.save(); err != nil {
		jww.FATAL.Printf("Failed to save RelationshipLegacySIDH %s after "+
			"adding session %s: %s", relationshipLegacySIDHKey, s, err)
	}

	return r
}

// todo - doscstring
func LoadRelationshipLegacySIDH(kv *versioned.KV, t session.RelationshipLegacySIDHType, myID,
	partnerID *id.ID, cyHandler session.CypherHandler, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) (*relationshipLegacySIDH, error) {

	kv = kv.Prefix(t.Prefix())

	r := &relationshipLegacySIDH{
		t:           t,
		sessionByID: make(map[session.SessionID]*session.Session),
		kv:          kv,
		myID:        myID,
		partnerID:   partnerID,
		cyHandler:   cyHandler,
		grp:         grp,
		rng:         rng,
	}

	obj, err := kv.Get(relationshipLegacySIDHKey, currentRelationshipLegacySIDHVersion)
	if err != nil {
		return nil, err
	}

	err = r.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *relationshipLegacySIDH) save() error {

	now := netTime.Now()

	data, err := r.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentRelationshipLegacySIDHVersion,
		Timestamp: now,
		Data:      data,
	}

	return r.kv.Set(relationshipLegacySIDHKey, &obj)
}

// ekv functions
func (r *relationshipLegacySIDH) marshal() ([]byte, error) {
	sessions := make([]session.SessionID, len(r.sessions))

	index := 0
	for sid := range r.sessionByID {
		sessions[index] = sid
		index++
	}

	return json.Marshal(&sessions)
}

func (r *relationshipLegacySIDH) unmarshal(b []byte) error {
	var sessions []session.SessionID

	if err := json.Unmarshal(b, &sessions); err != nil {
		return err
	}

	r.fingerprint = loadRelationshipLegacySIDHFingerprint(r.kv)

	//load all the sessions
	for _, sid := range sessions {
		s, err := session.LoadSession(r.kv, sid, r.fingerprint,
			r.cyHandler, r.grp, r.rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to load session %s for %s: %+v",
				session.MakeSessionPrefix(sid), r.partnerID, err)
		}
		r.addSession(s)
	}

	return nil
}

// todo - doscstring
func (r *relationshipLegacySIDH) Delete() {
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, s := range r.sessions {
		delete(r.sessionByID, s.GetID())
		s.Delete()
	}
}

// todo - doscstring
func (r *relationshipLegacySIDH) AddSession(myPrivKey, partnerPubKey, baseKey *cyclic.Int,
	mySIDHPrivKey *sidh.PrivateKey, partnerSIDHPubKey *sidh.PublicKey,
	trigger session.SessionID, negotiationStatus session.Negotiation,
	e2eParams session.Params) *session.Session {
	r.mux.Lock()
	defer r.mux.Unlock()

	s := session.NewSession(r.kv, r.t, r.partnerID, myPrivKey, partnerPubKey, baseKey,
		mySIDHPrivKey, partnerSIDHPubKey, trigger,
		r.fingerprint, negotiationStatus, e2eParams, r.cyHandler, r.grp, r.rng)

	r.addSession(s)
	if err := r.save(); err != nil {
		jww.FATAL.Printf("Failed to save RelationshipLegacySIDH %s after "+
			"adding session %s: %s", relationshipLegacySIDHKey, s, err)
	}

	return s
}

// todo - doscstring
func (r *relationshipLegacySIDH) addSession(s *session.Session) {
	r.sessions = append([]*session.Session{s}, r.sessions...)
	r.sessionByID[s.GetID()] = s
	return
}

// todo - doscstring
func (r *relationshipLegacySIDH) GetNewest() *session.Session {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if len(r.sessions) == 0 {
		return nil
	}
	return r.sessions[0]
}

// returns the key which is most likely to be successful for sending
func (r *relationshipLegacySIDH) getKeyForSending() (session.Cypher, error) {
	r.sendMux.Lock()
	defer r.sendMux.Unlock()
	s := r.getSessionForSending()
	if s == nil {
		return nil, errors.New("Failed to find a session for sending")
	}

	return s.PopKey()
}

// returns the session which is most likely to be successful for sending
func (r *relationshipLegacySIDH) getSessionForSending() *session.Session {
	sessions := r.sessions

	var confirmedRekey []*session.Session
	var unconfirmedActive []*session.Session
	var unconfirmedRekey []*session.Session

	jww.TRACE.Printf("[REKEY] Sessions Available: %d", len(sessions))

	for _, s := range sessions {
		status := s.Status()
		confirmed := s.IsConfirmed()
		jww.TRACE.Printf("[REKEY] Session Status/Confirmed: (%v, %s), %v",
			status, s.NegotiationStatus(), confirmed)
		if status == session.Active && confirmed {
			//always return the first confirmed active, happy path
			return s
		} else if status == session.RekeyNeeded && confirmed {
			confirmedRekey = append(confirmedRekey, s)
		} else if status == session.Active && !confirmed {
			unconfirmedActive = append(unconfirmedActive, s)
		} else if status == session.RekeyNeeded && !confirmed {
			unconfirmedRekey = append(unconfirmedRekey, s)
		}
	}

	//return the newest based upon priority
	if len(confirmedRekey) > 0 {
		return confirmedRekey[0]
	} else if len(unconfirmedActive) > 0 {
		return unconfirmedActive[0]
	} else if len(unconfirmedRekey) > 0 {
		return unconfirmedRekey[0]
	}

	jww.INFO.Printf("[REKEY] Details about %v sessions which are invalid:",
		len(sessions))
	for i, s := range sessions {
		if s == nil {
			jww.INFO.Printf("[REKEY]\tSession %v is nil", i)
		} else {
			jww.INFO.Printf("[REKEY]\tSession %v: status: %v,"+
				" confirmed: %v", i, s.Status(),
				s.IsConfirmed())
		}
	}

	return nil
}

// TriggerNegotiation returns a list of session that need rekeys. Nil instances mean a new rekey from scratch
func (r *relationshipLegacySIDH) TriggerNegotiation() []*session.Session {
	// Don't need to take the lock due to the use of a copy of the buffer
	sessions := r.getInternalBufferShallowCopy()
	var instructions []*session.Session
	for _, ses := range sessions {
		if ses.TriggerNegotiation() {
			instructions = append(instructions, ses)
		}
	}
	return instructions
}

// returns a key which should be used for rekeying
func (r *relationshipLegacySIDH) getKeyForRekey() (session.Cypher, error) {
	r.sendMux.Lock()
	defer r.sendMux.Unlock()
	s := r.getNewestRekeyableSession()
	if s == nil {
		return nil, errors.New("Failed to find a session for rekeying")
	}

	return s.PopReKey()
}

// returns the newest session which can be used to start a key negotiation
func (r *relationshipLegacySIDH) getNewestRekeyableSession() *session.Session {
	//dont need to take the lock due to the use of a copy of the buffer
	sessions := r.getInternalBufferShallowCopy()
	if len(sessions) == 0 {
		return nil
	}

	var unconfirmed *session.Session

	for _, s := range r.sessions {
		jww.TRACE.Printf("[REKEY] Looking at session %s", s)
		//fmt.Println(i)
		// This looks like it might not be thread safe, I
		// think it is because the failure mode is it skips to
		// a lower key to rekey with, which is always
		// valid. It isn't clear it can fail though because we
		// are accessing the data in the same order it would
		// be written (i think)
		if s.Status() != session.RekeyEmpty {
			if s.IsConfirmed() {
				jww.TRACE.Printf("[REKEY] Selected rekey: %s",
					s)
				return s
			} else if unconfirmed == nil {
				unconfirmed = s
			}
		}
	}
	jww.WARN.Printf("[REKEY] Returning unconfirmed session rekey: %s",
		unconfirmed)
	return unconfirmed
}

// todo - doscstring
func (r *relationshipLegacySIDH) GetByID(id session.SessionID) *session.Session {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.sessionByID[id]
}

// Confirm sets the passed session ID as confirmed and cleans up old sessions
func (r *relationshipLegacySIDH) Confirm(id session.SessionID) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	s, ok := r.sessionByID[id]
	if !ok {
		return errors.Errorf("cannot confirm session %s, "+
			"does not exist", id)
	}

	s.SetNegotiationStatus(session.Confirmed)

	r.clean()

	return nil
}

// adding or removing a session is always done via replacing the entire
// slice, this allow us to copy the slice under the read lock and do the
// rest of the work while not taking the lock
func (r *relationshipLegacySIDH) getInternalBufferShallowCopy() []*session.Session {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.sessions
}

// clean deletes old confirmed sessions
func (r *relationshipLegacySIDH) clean() {

	numConfirmed := uint(0)

	var newSessions []*session.Session
	editsMade := false

	for _, s := range r.sessions {
		if s.IsConfirmed() {
			numConfirmed++
			// if the number of newer confirmed is
			// sufficient, delete the confirmed
			if numConfirmed > maxUnconfirmed {
				delete(r.sessionByID, s.GetID())
				s.Delete()
				editsMade = true
				continue
			}
		}
		newSessions = append(newSessions, s)
	}

	//only do the update and save if changes occurred
	if editsMade {
		r.sessions = newSessions

		if err := r.save(); err != nil {
			jww.FATAL.Printf("cannot save Session Buffer %s after "+
				"clean: %s", r.kv.GetFullKey(relationshipLegacySIDHKey,
				currentRelationshipLegacySIDHVersion), err)
		}
	}
}
