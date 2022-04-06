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
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

const managerPrefix = "State{partner:%s}"
const originMyPrivKeyKey = "originMyPrivKey"
const originPartnerPubKey = "originPartnerPubKey"

type Manager struct {
	kv *versioned.KV

	myID    *id.ID
	partner *id.ID

	originMyPrivKey     *cyclic.Int
	originPartnerPubKey *cyclic.Int

	originMySIDHPrivKey     *sidh.PrivateKey
	originPartnerSIDHPubKey *sidh.PublicKey

	receive *relationship
	send    *relationship

	grp       *cyclic.Group
	cyHandler session.CypherHandler
	rng       *fastRNG.StreamGenerator

	managerID ManagerIdentity
}

// NewManager creates the relationship and its first Send and Receive sessions.
func NewManager(kv *versioned.KV, myID, partnerID *id.ID, myPrivKey,
	partnerPubKey *cyclic.Int, mySIDHPrivKey *sidh.PrivateKey,
	partnerSIDHPubKey *sidh.PublicKey, sendParams,
	receiveParams session.Params, cyHandler session.CypherHandler,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator) *Manager {

	mi := MakeManagerIdentity(partnerID, myID)

	kv = kv.Prefix(makeManagerPrefix(mi))

	m := &Manager{
		kv:                      kv,
		originMyPrivKey:         myPrivKey,
		originPartnerPubKey:     partnerPubKey,
		originMySIDHPrivKey:     mySIDHPrivKey,
		originPartnerSIDHPubKey: partnerSIDHPubKey,
		myID:                    myID,
		partner:                 partnerID,
		cyHandler:               cyHandler,
		grp:                     grp,
		rng:                     rng,
		managerID:               mi,
	}
	if err := utility.StoreCyclicKey(kv, myPrivKey, originMyPrivKeyKey); err != nil {
		jww.FATAL.Panicf("Failed to store %s: %+v", originMyPrivKeyKey,
			err)
	}

	if err := utility.StoreCyclicKey(kv, partnerPubKey, originPartnerPubKey); err != nil {
		jww.FATAL.Panicf("Failed to store %s: %+v", originPartnerPubKey,
			err)
	}

	m.send = NewRelationship(m.kv, session.Send, myID, partnerID, myPrivKey,
		partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey, sendParams, cyHandler,
		grp, rng)
	m.receive = NewRelationship(m.kv, session.Receive, myID, partnerID,
		myPrivKey, partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		receiveParams, cyHandler, grp, rng)

	return m
}

//LoadManager loads a relationship and all buffers and sessions from disk
func LoadManager(kv *versioned.KV, myID, partnerID *id.ID,
	cyHandler session.CypherHandler, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) (*Manager, error) {

	mi := MakeManagerIdentity(partnerID, myID)

	m := &Manager{
		kv:        kv.Prefix(makeManagerPrefix(mi)),
		myID:      myID,
		partner:   partnerID,
		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
		managerID: mi,
	}

	var err error

	m.originMyPrivKey, err = utility.LoadCyclicKey(m.kv, originMyPrivKeyKey)
	if err != nil {
		// if the key cannot be found, this might be an old session, in which case
		// we attempt to revert to the old file structure
		m.kv = kv.Prefix(makeOldManagerPrefix(partnerID))
		m.originMyPrivKey, err = utility.LoadCyclicKey(m.kv, originMyPrivKeyKey)
		if err != nil {
			jww.FATAL.Panicf("Failed to load %s: %+v", originMyPrivKeyKey,
				err)
		}

	}

	m.originPartnerPubKey, err = utility.LoadCyclicKey(m.kv, originPartnerPubKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load %s: %+v", originPartnerPubKey,
			err)
	}

	m.send, err = LoadRelationship(m.kv, session.Send, myID, partnerID,
		cyHandler, grp, rng)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key relationship due to failure to "+
				"load the Send session buffer")
	}

	m.receive, err = LoadRelationship(m.kv, session.Receive, myID, partnerID,
		cyHandler, grp, rng)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key relationship due to failure to "+
				"load the Receive session buffer")
	}

	return m, nil
}

// ClearManager removes the relationship between the partner
// and deletes the Send and Receive sessions. This includes the
// sessions and the key vectors
func ClearManager(m *Manager, kv *versioned.KV) error {
	kv = kv.Prefix(fmt.Sprintf(managerPrefix, m.partner))

	if err := DeleteRelationship(m); err != nil {
		return errors.WithMessage(err,
			"Failed to delete relationship")
	}

	if err := utility.DeleteCyclicKey(m.kv, originPartnerPubKey); err != nil {
		jww.FATAL.Panicf("Failed to delete %s: %+v", originPartnerPubKey,
			err)
	}

	return nil
}

// NewReceiveSession creates a new Receive session using the latest private key
// this user has sent and the new public key received from the partner. If the
// session already exists, then it will not be overwritten and the extant
// session will be returned with the bool set to true denoting a duplicate. This
// allows for support of duplicate key exchange triggering.
func (m *Manager) NewReceiveSession(partnerPubKey *cyclic.Int,
	partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
	source *session.Session) (*session.Session, bool) {

	// Check if the session already exists
	baseKey := session.GenerateE2ESessionBaseKey(source.GetMyPrivKey(), partnerPubKey,
		m.grp, source.GetMySIDHPrivKey(), partnerSIDHPubKey)

	sessionID := session.GetSessionIDFromBaseKey(baseKey)

	if s := m.receive.GetByID(sessionID); s != nil {
		return s, true
	}

	// Add the session to the buffer
	s := m.receive.AddSession(source.GetMyPrivKey(), partnerPubKey, baseKey,
		source.GetMySIDHPrivKey(), partnerSIDHPubKey,
		source.GetID(), session.Confirmed, e2eParams)

	return s, false
}

// NewSendSession creates a new Send session using the latest public key
// received from the partner and a new private key for the user. Passing in a
// private key is optional. A private key will be generated if none is passed.
func (m *Manager) NewSendSession(myPrivKey *cyclic.Int,
	mySIDHPrivKey *sidh.PrivateKey, e2eParams session.Params,
	sourceSession *session.Session) *session.Session {

	// Add the session to the Send session buffer and return
	return m.send.AddSession(myPrivKey, sourceSession.GetPartnerPubKey(), nil,
		mySIDHPrivKey, sourceSession.GetPartnerSIDHPubKey(),
		sourceSession.GetID(), session.Sending, e2eParams)
}

// PopSendCypher gets the correct session to Send with depending on the type
// of Send.
func (m *Manager) PopSendCypher() (*session.Cypher, error) {
	return m.send.getKeyForSending()
}

// PopRekeyCypher gets the correct session to Send with depending on the type
// of Send.
func (m *Manager) PopRekeyCypher() (*session.Cypher, error) {
	return m.send.getKeyForRekey()

}

// GetPartnerID returns a copy of the ID of the partner.
func (m *Manager) GetPartnerID() *id.ID {
	return m.partner.DeepCopy()
}

// GetMyID returns a copy of the ID used as self.
func (m *Manager) GetMyID() *id.ID {
	return m.myID.DeepCopy()
}

// GetSendSession gets the Send session of the passed ID. Returns nil if no
// session is found.
func (m *Manager) GetSendSession(sid session.SessionID) *session.Session {
	return m.send.GetByID(sid)
}

// GetReceiveSession gets the Receive session of the passed ID. Returns nil if
// no session is found.
func (m *Manager) GetReceiveSession(sid session.SessionID) *session.Session {
	return m.receive.GetByID(sid)
}

// GetSendSession gets the Send session of the passed ID. Returns nil if no
// session is found.
func (m *Manager) GetSendRelationshipFingerprint() []byte {
	return m.send.fingerprint
}

// Confirm confirms a Send session is known about by the partner.
func (m *Manager) Confirm(sid session.SessionID) error {
	return m.send.Confirm(sid)
}

// TriggerNegotiations returns a list of key exchange operations if any are
// necessary.
func (m *Manager) TriggerNegotiations() []*session.Session {
	return m.send.TriggerNegotiation()
}

func (m *Manager) GetMyOriginPrivateKey() *cyclic.Int {
	return m.originMyPrivKey.DeepCopy()
}

func (m *Manager) GetPartnerOriginPublicKey() *cyclic.Int {
	return m.originPartnerPubKey.DeepCopy()
}

const relationshipFpLength = 15

// GetRelationshipFingerprint returns a unique fingerprint for an E2E
// relationship. The fingerprint is a base 64 encoded hash of of the two
// relationship fingerprints truncated to 15 characters.
func (m *Manager) GetRelationshipFingerprint() string {

	// Base 64 encode hash and truncate
	return base64.StdEncoding.EncodeToString(m.GetRelationshipFingerprintBytes())[:relationshipFpLength]
}

// GetRelationshipFingerprintBytes returns a unique fingerprint for an E2E
// relationship. used for the e2e preimage.
func (m *Manager) GetRelationshipFingerprintBytes() []byte {
	// Sort fingerprints
	var fps [][]byte

	if bytes.Compare(m.receive.fingerprint, m.send.fingerprint) == 1 {
		fps = [][]byte{m.send.fingerprint, m.receive.fingerprint}
	} else {
		fps = [][]byte{m.receive.fingerprint, m.send.fingerprint}
	}

	// Hash fingerprints
	h, _ := blake2b.New256(nil)
	for _, fp := range fps {
		h.Write(fp)
	}

	// Base 64 encode hash and truncate
	return h.Sum(nil)
}

func (m *Manager) GetIdentity() ManagerIdentity {
	return m.managerID
}

// MakeService Returns a service interface with the
// appropriate identifier for who is being sent to. Will populate
// the metadata with the partner
func (m *Manager) MakeService(tag string) message.Service {
	return message.Service{
		Identifier: m.GetRelationshipFingerprintBytes(),
		Tag:        tag,
		Metadata:   m.partner[:],
	}
}

// GetContact assembles and returns a contact.Contact with the partner's ID
// and DH key.
func (m *Manager) GetContact() contact.Contact {
	// Assemble Contact
	return contact.Contact{
		ID:       m.GetPartnerID(),
		DhPubKey: m.GetPartnerOriginPublicKey(),
	}
}

func makeOldManagerPrefix(pid *id.ID) string {
	return fmt.Sprintf(managerPrefix, pid)
}

func makeManagerPrefix(identity ManagerIdentity) string {
	return fmt.Sprintf(managerPrefix, identity)
}
