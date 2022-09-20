////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"fmt"

	"gitlab.com/elixxir/crypto/e2e"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

// Implements the partner.ManagerLegacySIDH interface
type managerLegacySIDH struct {
	kv *versioned.KV

	myID    *id.ID
	partner *id.ID

	originMyPrivKey     *cyclic.Int
	originPartnerPubKey *cyclic.Int

	originMySIDHPrivKey     *sidh.PrivateKey
	originPartnerSIDHPubKey *sidh.PublicKey

	receive *relationshipLegacySIDH
	send    *relationshipLegacySIDH

	grp       *cyclic.Group
	cyHandler session.CypherHandlerLegacySIDH
	rng       *fastRNG.StreamGenerator
}

// NewManagerLegacySIDH creates the relationship and its first Send and Receive sessions.
func NewManagerLegacySIDH(kv *versioned.KV, myID, partnerID *id.ID, myPrivKey,
	partnerPubKey *cyclic.Int, mySIDHPrivKey *sidh.PrivateKey,
	partnerSIDHPubKey *sidh.PublicKey, sendParams,
	receiveParams session.Params, cyHandler session.CypherHandlerLegacySIDH,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator) ManagerLegacySIDH {

	kv = kv.Prefix(makeManagerPrefix(partnerID))

	m := &managerLegacySIDH{
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

	m.send = NewRelationshipLegacySIDH(m.kv, session.Send, myID, partnerID, myPrivKey,
		partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, cyHandler, grp, rng)
	m.receive = NewRelationshipLegacySIDH(m.kv, session.Receive, myID, partnerID,
		myPrivKey, partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		receiveParams, cyHandler, grp, rng)

	return m
}

// LoadManagerLegacySIDH loads a relationship and all buffers and sessions from disk
func LoadManagerLegacySIDH(kv *versioned.KV, myID, partnerID *id.ID,
	cyHandler session.CypherHandlerLegacySIDH, grp *cyclic.Group,
	rng *fastRNG.StreamGenerator) (ManagerLegacySIDH, error) {

	m := &managerLegacySIDH{
		kv:        kv.Prefix(makeManagerPrefix(partnerID)),
		myID:      myID,
		partner:   partnerID,
		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
	}

	var err error

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

	m.send, err = LoadRelationshipLegacySIDH(m.kv, session.Send, myID, partnerID,
		cyHandler, grp, rng)
	if err != nil {
		return nil, errors.WithMessage(err,
			"cannot load partner key relationship due to failure"+
				" to load the Send session buffer")
	}

	m.receive, err = LoadRelationshipLegacySIDH(m.kv, session.Receive, myID, partnerID,
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
func (m *managerLegacySIDH) Delete() error {
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
func (m *managerLegacySIDH) deleteRelationships() error {

	// Delete the send information
	sendKv := m.kv.Prefix(session.Send.Prefix())
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
	receiveKv := m.kv.Prefix(session.Receive.Prefix())
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
func (m *managerLegacySIDH) NewReceiveSession(partnerPubKey *cyclic.Int,
	partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
	source *session.SessionLegacySIDH) (*session.SessionLegacySIDH, bool) {

	// Check if the session already exists
	baseKey := session.GenerateE2ESessionBaseKeyLegacySIDH(source.GetMyPrivKey(),
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
func (m *managerLegacySIDH) NewSendSession(myPrivKey *cyclic.Int,
	mySIDHPrivKey *sidh.PrivateKey, e2eParams session.Params,
	sourceSession *session.SessionLegacySIDH) *session.SessionLegacySIDH {

	// Add the session to the Send session buffer and return
	return m.send.AddSession(myPrivKey, sourceSession.GetPartnerPubKey(),
		nil, mySIDHPrivKey, sourceSession.GetPartnerSIDHPubKey(),
		sourceSession.GetID(), session.Sending, e2eParams)
}

// PopSendCypher returns the key which is most likely to be successful for sending
func (m *managerLegacySIDH) PopSendCypher() (session.CypherLegacySIDH, error) {
	return m.send.getKeyForSending()
}

// PopRekeyCypher returns a key which should be used for rekeying
func (m *managerLegacySIDH) PopRekeyCypher() (session.CypherLegacySIDH, error) {
	return m.send.getKeyForRekey()

}

// PartnerId returns a copy of the ID of the partner.
func (m *managerLegacySIDH) PartnerId() *id.ID {
	return m.partner.DeepCopy()
}

// MyId returns a copy of the ID used as self.
func (m *managerLegacySIDH) MyId() *id.ID {
	return m.myID.DeepCopy()
}

// GetSendSession gets the Send session of the passed ID. Returns nil if no
// session is found.
func (m *managerLegacySIDH) GetSendSession(sid session.SessionID) *session.SessionLegacySIDH {
	return m.send.GetByID(sid)
}

// GetReceiveSession gets the Receive session of the passed ID. Returns nil if
// no session is found.
func (m *managerLegacySIDH) GetReceiveSession(sid session.SessionID) *session.SessionLegacySIDH {
	return m.receive.GetByID(sid)
}

// SendRelationshipFingerprint
func (m *managerLegacySIDH) SendRelationshipFingerprint() []byte {
	return m.send.fingerprint
}

// ReceiveRelationshipFingerprint
func (m *managerLegacySIDH) ReceiveRelationshipFingerprint() []byte {
	return m.receive.fingerprint
}

// Confirm confirms a Send session is known about by the partner.
func (m *managerLegacySIDH) Confirm(sid session.SessionID) error {
	return m.send.Confirm(sid)
}

// TriggerNegotiations returns a list of key exchange operations if any are necessary.
func (m *managerLegacySIDH) TriggerNegotiations() []*session.SessionLegacySIDH {
	return m.send.TriggerNegotiation()
}

func (m *managerLegacySIDH) MyRootPrivateKey() *cyclic.Int {
	return m.originMyPrivKey.DeepCopy()
}

func (m *managerLegacySIDH) PartnerRootPublicKey() *cyclic.Int {
	return m.originPartnerPubKey.DeepCopy()
}

// ConnectionFingerprint returns a unique fingerprint for an E2E
// relationship used for the e2e preimage.
func (m *managerLegacySIDH) ConnectionFingerprint() ConnectionFp {
	return ConnectionFp{fingerprint: e2e.GenerateConnectionFingerprint(m.send.fingerprint, m.receive.fingerprint)}
}

// MakeService Returns a service interface with the
// appropriate identifier for who is being sent to. Will populate
// the metadata with the partner
func (m *managerLegacySIDH) MakeService(tag string) message.Service {
	return message.Service{
		Identifier: m.ConnectionFingerprint().Bytes(),
		Tag:        tag,
		Metadata:   m.partner[:],
	}
}

// Contact assembles and returns a contact.Contact with the partner's ID and DH key.
func (m *managerLegacySIDH) Contact() contact.Contact {
	// Assemble Contact
	return contact.Contact{
		ID:       m.PartnerId(),
		DhPubKey: m.PartnerRootPublicKey(),
	}
}

func makeManagerLegacySIDHPrefix(pid *id.ID) string {
	return fmt.Sprintf(managerPrefix, pid)
}
