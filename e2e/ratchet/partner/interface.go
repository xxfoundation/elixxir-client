////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Manager create and manages both E2E send and receive sessions using the passed cryptographic data
type Manager interface {
	// PartnerId returns the ID of the E2E partner
	PartnerId() *id.ID
	// MyId returns my ID used for the E2E relationship
	MyId() *id.ID
	// MyRootPrivateKey returns first private key in the DAG
	MyRootPrivateKey() *cyclic.Int
	// PartnerRootPublicKey returns the partner's first public key in the DAG
	PartnerRootPublicKey() *cyclic.Int
	// SendRelationshipFingerprint returns the fingerprint of the send session
	SendRelationshipFingerprint() []byte
	// ReceiveRelationshipFingerprint returns the fingerprint of the receive session
	ReceiveRelationshipFingerprint() []byte
	// ConnectionFingerprint returns a unique fingerprint for an E2E relationship in string format
	ConnectionFingerprint() ConnectionFp
	// Contact returns the contact of the E2E partner
	Contact() contact.Contact

	// PopSendCypher returns the key which is most likely to be successful for sending
	PopSendCypher() (session.Cypher, error)
	// PopRekeyCypher returns a key which should be used for rekeying
	PopRekeyCypher() (session.Cypher, error)

	// NewReceiveSession creates a new Receive session using the latest private key
	// this user has sent and the new public key received from the partner. If the
	// session already exists, then it will not be overwritten and the extant
	// session will be returned with the bool set to true denoting a duplicate. This
	// allows for support of duplicate key exchange triggering.
	NewReceiveSession(partnerPubKey *cyclic.Int,
		partnerCTIDHPubKey nike.PublicKey, e2eParams session.Params,
		source *session.Session) (*session.Session, bool)
	// NewSendSession creates a new Send session using the latest public key
	// received from the partner and a new private key for the user. Passing in a
	// private key is optional. A private key will be generated if none is passed.
	NewSendSession(myDHPrivKey *cyclic.Int, myCTIDHPrivateKey nike.PrivateKey,
		e2eParams session.Params, source *session.Session) *session.Session
	// GetSendSession gets the Send session of the passed ID. Returns nil if no session is found.
	GetSendSession(sid session.SessionID) *session.Session
	//GetReceiveSession gets the Receive session of the passed ID. Returns nil if no session is found.
	GetReceiveSession(sid session.SessionID) *session.Session

	// Confirm sets the passed session ID as confirmed and cleans up old sessions
	Confirm(sid session.SessionID) error

	// TriggerNegotiations returns a list of session that need rekeys
	TriggerNegotiations() []*session.Session

	// MakeService Returns a service interface with the
	// appropriate identifier for who is being sent to. Will populate
	// the metadata with the partner
	MakeService(tag string) message.Service

	// Delete removes the relationship between the partner
	// and deletes the Send and Receive sessions. This includes the
	// sessions and the key vectors
	Delete() error
}

// ManagerLegacySIDH create and manages both E2E send and receive sessions using the passed cryptographic data
type ManagerLegacySIDH interface {
	// PartnerId returns the ID of the E2E partner
	PartnerId() *id.ID
	// MyId returns my ID used for the E2E relationship
	MyId() *id.ID
	// MyRootPrivateKey returns first private key in the DAG
	MyRootPrivateKey() *cyclic.Int
	// PartnerRootPublicKey returns the partner's first public key in the DAG
	PartnerRootPublicKey() *cyclic.Int
	// SendRelationshipFingerprint returns the fingerprint of the send session
	SendRelationshipFingerprint() []byte
	// ReceiveRelationshipFingerprint returns the fingerprint of the receive session
	ReceiveRelationshipFingerprint() []byte
	// ConnectionFingerprint returns a unique fingerprint for an E2E relationship in string format
	ConnectionFingerprint() ConnectionFp
	// Contact returns the contact of the E2E partner
	Contact() contact.Contact

	// PopSendCypher returns the key which is most likely to be successful for sending
	PopSendCypher() (session.Cypher, error)
	// PopRekeyCypher returns a key which should be used for rekeying
	PopRekeyCypher() (session.Cypher, error)

	// NewReceiveSession creates a new Receive session using the latest private key
	// this user has sent and the new public key received from the partner. If the
	// session already exists, then it will not be overwritten and the extant
	// session will be returned with the bool set to true denoting a duplicate. This
	// allows for support of duplicate key exchange triggering.
	NewReceiveSession(partnerPubKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
		source *session.SessionLegacySIDH) (*session.SessionLegacySIDH, bool)
	// NewSendSession creates a new Send session using the latest public key
	// received from the partner and a new private key for the user. Passing in a
	// private key is optional. A private key will be generated if none is passed.
	NewSendSession(myDHPrivKey *cyclic.Int, mySIDHPrivateKey *sidh.PrivateKey,
		e2eParams session.Params, source *session.SessionLegacySIDH) *session.SessionLegacySIDH
	// GetSendSession gets the Send session of the passed ID. Returns nil if no session is found.
	GetSendSession(sid session.SessionID) *session.SessionLegacySIDH
	//GetReceiveSession gets the Receive session of the passed ID. Returns nil if no session is found.
	GetReceiveSession(sid session.SessionID) *session.SessionLegacySIDH

	// Confirm sets the passed session ID as confirmed and cleans up old sessions
	Confirm(sid session.SessionID) error

	// TriggerNegotiations returns a list of session that need rekeys
	TriggerNegotiations() []*session.SessionLegacySIDH

	// MakeService Returns a service interface with the
	// appropriate identifier for who is being sent to. Will populate
	// the metadata with the partner
	MakeService(tag string) message.Service

	// Delete removes the relationship between the partner
	// and deletes the Send and Receive sessions. This includes the
	// sessions and the key vectors
	Delete() error
}
