////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"encoding/base64"
	"fmt"
	"gitlab.com/elixxir/crypto/e2e"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

const managerPrefix = "Manager{partner:%s}"
const originMyPrivKeyKey = "originMyPrivKey"
const originPartnerPubKey = "originPartnerPubKey"
const relationshipFpLength = 15

// Implements the partner.Manager interface
type manager struct {
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
}

// NewManager creates the relationship and its first Send and Receive sessions.
func NewManager(kv *versioned.KV, myID, partnerID *id.ID, myPrivKey,
	partnerPubKey *cyclic.Int, mySIDHPrivKey *sidh.PrivateKey,
	partnerSIDHPubKey *sidh.PublicKey, sendParams,
	receiveParams session.Params, cyHandler session.CypherHandler,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator) Manager {

	kv, err := kv.Prefix(makeManagerPrefix(partnerID))
	if err != nil {
		jww.FATAL.Panicf("Failed to add prefix %s to KV: %+v",
			makeManagerPrefix(partnerID), err)
	}

	m := &manager{
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
	}
	if err := utility.StoreCyclicKey(kv, myPrivKey,
		originMyPrivKeyKey); err != nil {
		jww.FATAL.Panicf("Failed to store %s: %+v", originMyPrivKeyKey,
			err)
	}

	if err := utility.StoreCyclicKey(kv, partnerPubKey,
		originPartnerPubKey); err != nil {
		jww.FATAL.Panicf("Failed to store %s: %+v", originPartnerPubKey,
			err)
	}

	m.send = NewRelationship(m.kv, session.Send, myID, partnerID, myPrivKey,
		partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, cyHandler, grp, rng)
	m.receive = NewRelationship(m.kv, session.Receive, myID, partnerID,
		myPrivKey, partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		receiveParams, cyHandler, grp, rng)

	return m
}

// ConnectionFp represents a Partner connection fingerprint
type ConnectionFp struct {
	fingerprint []byte
}

func (c ConnectionFp) Bytes() []byte {
	return c.fingerprint
}

func (c ConnectionFp) String() string {
	// Base 64 encode hash and truncate
	return base64.StdEncoding.EncodeToString(
		c.fingerprint)[:relationshipFpLength]
}

// LoadManager loads a relationship and all buffers and sessions from disk
func LoadManager(kv *versioned.KV, myID, partnerID *id.ID,
	cyHandler session.CypherHandler, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) (Manager, error) {

	kv, err := kv.Prefix(makeManagerPrefix(partnerID))
	if err != nil {
		return nil, err
	}

	m := &manager{
		kv:        kv,
		myID:      myID,
		partner:   partnerID,
		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
	}

	m.originMyPrivKey, err = utility.LoadCyclicKey(m.kv, originMyPrivKeyKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load %s: %+v", originMyPrivKeyKey,
			err)
	}

	m.originPartnerPubKey, err = utility.LoadCyclicKey(m.kv,
		originPartnerPubKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load %s: %+v", originPartnerPubKey,
			err)
	}

	m.send, err = LoadRelationship(m.kv, session.Send, myID, partnerID,
		cyHandler, grp, rng)
	if err != nil {
		return nil, errors.WithMessage(err,
			"cannot load partner key relationship due to failure"+
				" to load the Send session buffer")
	}

	m.receive, err = LoadRelationship(m.kv, session.Receive, myID, partnerID,
		cyHandler, grp, rng)
	if err != nil {
		return nil, errors.WithMessage(err,
			"cannot load partner key relationship due to failure"+
				" to load the Receive session buffer")
	}

	return m, nil
}

// Delete removes the relationship between the partner
// and deletes the Send and Receive sessions. This includes the
// sessions and the key vectors
func (m *manager) Delete() error {
	if err := m.deleteRelationships(); err != nil {
		return errors.WithMessage(err,
			"Failed to delete relationship")
	}

	if err := utility.DeleteCyclicKey(m.kv,
		originPartnerPubKey); err != nil {
		jww.FATAL.Panicf("Failed to delete %s: %+v",
			originPartnerPubKey, err)
	}

	return nil
}

// deleteRelationships removes all relationship and
// relationship adjacent information from storage
func (m *manager) deleteRelationships() error {

	// Delete the send information
	sendKv, err := m.kv.Prefix(session.Send.Prefix())
	if err != nil {
		return err
	}
	m.send.Delete()
	if err := deleteRelationshipFingerprint(sendKv); err != nil {
		return err
	}
	if err := sendKv.Delete(relationshipKey,
		currentRelationshipVersion); err != nil {
		return errors.Errorf("cannot delete send relationship: %v",
			err)
	}

	// Delete the receive information
	receiveKv, err := m.kv.Prefix(session.Receive.Prefix())
	if err != nil {
		return err
	}
	m.receive.Delete()
	if err := deleteRelationshipFingerprint(receiveKv); err != nil {
		return err
	}
	if err := receiveKv.Delete(relationshipKey,
		currentRelationshipVersion); err != nil {
		return errors.Errorf("cannot delete receive relationship: %v",
			err)
	}

	return nil
}

// NewReceiveSession creates a new Receive session using the latest private key
// this user has sent and the new public key received from the partner. If the
// session already exists, then it will not be overwritten and the extant
// session will be returned with the bool set to true denoting a duplicate. This
// allows for support of duplicate key exchange triggering.
func (m *manager) NewReceiveSession(partnerPubKey *cyclic.Int,
	partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
	source *session.Session) (*session.Session, bool) {

	// Check if the session already exists
	baseKey := session.GenerateE2ESessionBaseKey(source.GetMyPrivKey(),
		partnerPubKey, m.grp, source.GetMySIDHPrivKey(),
		partnerSIDHPubKey)

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
func (m *manager) NewSendSession(myPrivKey *cyclic.Int,
	mySIDHPrivKey *sidh.PrivateKey, e2eParams session.Params,
	sourceSession *session.Session) *session.Session {

	// Add the session to the Send session buffer and return
	return m.send.AddSession(myPrivKey, sourceSession.GetPartnerPubKey(),
		nil, mySIDHPrivKey, sourceSession.GetPartnerSIDHPubKey(),
		sourceSession.GetID(), session.Sending, e2eParams)
}

// PopSendCypher returns the key which is most likely to be successful for sending
func (m *manager) PopSendCypher() (session.Cypher, error) {
	return m.send.getKeyForSending()
}

// PopRekeyCypher returns a key which should be used for rekeying
func (m *manager) PopRekeyCypher() (session.Cypher, error) {
	return m.send.getKeyForRekey()

}

// PartnerId returns a copy of the ID of the partner.
func (m *manager) PartnerId() *id.ID {
	return m.partner.DeepCopy()
}

// MyId returns a copy of the ID used as self.
func (m *manager) MyId() *id.ID {
	return m.myID.DeepCopy()
}

// GetSendSession gets the Send session of the passed ID. Returns nil if no
// session is found.
func (m *manager) GetSendSession(sid session.SessionID) *session.Session {
	return m.send.GetByID(sid)
}

// GetReceiveSession gets the Receive session of the passed ID. Returns nil if
// no session is found.
func (m *manager) GetReceiveSession(sid session.SessionID) *session.Session {
	return m.receive.GetByID(sid)
}

// SendRelationshipFingerprint
func (m *manager) SendRelationshipFingerprint() []byte {
	return m.send.fingerprint
}

// ReceiveRelationshipFingerprint
func (m *manager) ReceiveRelationshipFingerprint() []byte {
	return m.receive.fingerprint
}

// Confirm confirms a Send session is known about by the partner.
func (m *manager) Confirm(sid session.SessionID) error {
	return m.send.Confirm(sid)
}

// TriggerNegotiations returns a list of key exchange operations if any are necessary.
func (m *manager) TriggerNegotiations() []*session.Session {
	return m.send.TriggerNegotiation()
}

func (m *manager) MyRootPrivateKey() *cyclic.Int {
	return m.originMyPrivKey.DeepCopy()
}

func (m *manager) PartnerRootPublicKey() *cyclic.Int {
	return m.originPartnerPubKey.DeepCopy()
}

// ConnectionFingerprint returns a unique fingerprint for an E2E
// relationship used for the e2e preimage.
func (m *manager) ConnectionFingerprint() ConnectionFp {
	return ConnectionFp{fingerprint: e2e.GenerateConnectionFingerprint(m.send.fingerprint, m.receive.fingerprint)}
}

// MakeService Returns a service interface with the
// appropriate identifier for who is being sent to. Will populate
// the metadata with the partner
func (m *manager) MakeService(tag string) message.Service {
	return message.Service{
		Identifier: m.ConnectionFingerprint().Bytes(),
		Tag:        tag,
		Metadata:   m.partner[:],
	}
}

// Contact assembles and returns a contact.Contact with the partner's ID and DH key.
func (m *manager) Contact() contact.Contact {
	// Assemble Contact
	return contact.Contact{
		ID:       m.PartnerId(),
		DhPubKey: m.PartnerRootPublicKey(),
	}
}

func makeManagerPrefix(pid *id.ID) string {
	return fmt.Sprintf(managerPrefix, pid)
}
