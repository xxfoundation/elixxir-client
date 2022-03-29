package ratchet

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Ratchet interface {
	AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams session.Params)
	GetPartner(partnerID *id.ID) (*Manager, error)
	DeletePartner(partnerId *id.ID)
	GetAllPartnerIDs() []*id.ID
	GetDHPrivateKey() *cyclic.Int
	GetDHPublicKey() *cyclic.Int
	AddService(tag string, processor message.Processor)
	RemoveService(tag string)
}
