package e2e

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/interfaces/message"
	message2 "gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type Handler interface {
	SendE2E(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
	AddPartner(partnerID *id.ID, partnerPubKey,
		myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams params.E2ESessionParams)
	GetPartner(partnerID *id.ID) (*Manager, error)
	DeletePartner(partnerId *id.ID)
	GetAllPartnerIDs() []*id.ID
	//aproach 2
	AddService(tag string, source []byte, processor message2.Processor)
	RemoveService(tag string, source []byte)
}
