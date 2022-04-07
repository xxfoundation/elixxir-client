package partner

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Manager2 interface {
	NewReceiveSession(partnerPubKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params,
		source *session.Session) (*session.Session, bool)
	NewSendSession(myPrivKey *cyclic.Int,
		mySIDHPrivKey *sidh.PrivateKey, e2eParams session.Params)
	PopSendCypher() (*session.Cypher, error)
	PopRekeyCypher() (*session.Cypher, error)
	GetPartnerID() *id.ID
	GetSendSession(sid session.SessionID) *session.Session
	GetSendRelationshipFingerprint()
	Confirm(sid session.SessionID)
	TriggerNegotiations() []*session.Session
	GetMyOriginPrivateKey()
	GetPartnerOriginPublicKey()
	GetRelationshipFingerprintBytes() []byte
	MakeService(tag string) message.Service
	GetContact() contact.Contact
}
