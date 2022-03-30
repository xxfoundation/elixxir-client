package ratchet

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Ratchet2 interface {
	// AddPartner adds a partner. Automatically creates both send and receive
	// sessions using the passed cryptographic data and per the parameters sent
	//
	AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams session.Params)

	GetPartner(partnerID *id.ID) (*partner.Manager, error)
	DeletePartner(partnerId *id.ID)
	GetAllPartnerIDs() []*id.ID
	GetDHPrivateKey() *cyclic.Int
	GetDHPublicKey() *cyclic.Int
	AddService(tag string, processor message.Processor)
	RemoveService(tag string)
}
