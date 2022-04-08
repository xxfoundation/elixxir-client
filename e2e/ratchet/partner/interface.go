package partner

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Manager interface {
	NewReceiveSession(partnerPubKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
		source *session.Session) (*session.Session, bool)
	NewSendSession(myDHPrivKey *cyclic.Int, mySIDHPrivateKey *sidh.PrivateKey,
		e2eParams session.Params, source *session.Session) *session.Session
	PopSendCypher() (*session.Cypher, error)
	PopRekeyCypher() (*session.Cypher, error)
	GetPartnerID() *id.ID
	GetMyID() *id.ID
	GetSendSession(sid session.SessionID) *session.Session
	GetReceiveSession(sid session.SessionID) *session.Session
	Confirm(sid session.SessionID) error
	TriggerNegotiations() []*session.Session
	GetMyOriginPrivateKey() *cyclic.Int
	GetPartnerOriginPublicKey() *cyclic.Int
	GetSendRelationshipFingerprint() []byte
	GetRelationshipFingerprintBytes() []byte
	GetRelationshipFingerprint() string
	MakeService(tag string) message.Service
	GetContact() contact.Contact
	DeleteRelationship() error
	ClearManager() error
}
