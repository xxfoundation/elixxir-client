package keyExchange

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
	jww "github.com/spf13/jwalterweatherman"
)

type KeyExchangeSend interface {
	SendRekey(m context.Message) ([]id.Round, error)
}

func Rekey(ctx *context.Context, partner *id.ID, kx KeyExchangeSend) {
	e2eStore := ctx.Session.E2e()

	//get the key manager
	manager, err := e2eStore.GetPartner(partner)
	if err != nil {
		jww.ERROR.Printf("Failed to Rekey with %s: Failed to retrieve "+
			"key manager: %s", partner, err)
	}

	//create the session
	session, err := manager.NewSendSession(nil, e2e.GetDefaultSessionParams())
	if err != nil {
		jww.ERROR.Printf("Failed to Rekey with %s: Failed to make new "+
			"session: %s", partner, err)
	}

	//generate public key
	pubKey := diffieHellman.GeneratePublicKey(session.GetMyPrivKey(),
		e2eStore.GetGroup())

	//send session
	m := context.Message{
		Recipient:   partner,
		Payload:     pubKey.Bytes(),
		MessageType: 42,
	}

	rounds, err := kx.SendRekey(m)
	if err != nil {

	}

}
