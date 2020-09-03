package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

func handleTrigger(ctx *context.Context, request message.Receive) {
	//ensure the message was encrypted properly
	if request.Encryption != message.E2E {
		jww.ERROR.Printf("Received non-e2e encrypted Key Exchange "+
			"Trigger from partner %s", request.Sender)
		return
	}

	//Get the partner
	partner, err := ctx.Session.E2e().GetPartner(request.Sender)
	if err != nil {
		jww.ERROR.Printf("Received Key Exchange Trigger with unknown "+
			"partner %s", request.Sender)
		return
	}

	//unmarshal the message
	oldSessionID, PartnerPublicKey, err := unmarshalKeyExchangeTrigger(
		ctx.Session.E2e().GetGroup(), request.Payload)
	if err != nil {
		jww.ERROR.Printf("Failed to unmarshal Key Exchange Trigger with "+
			"partner %s: %s", request.Sender, err)
		return
	}

	//get the old session which triggered the exchange
	oldSession := partner.GetSendSession(oldSessionID)
	if oldSession == nil {
		jww.ERROR.Printf("Failed to find parent session %s for Key "+
			"Exchange Trigger from partner %s: %s", oldSession, request.Sender,
			err)
		return
	}

	//create the new session
	newSession, duplicate := partner.NewReceiveSession(PartnerPublicKey,
		e2e.GetDefaultSessionParams(), oldSession)
	// new session being nil means the session was a duplicate. This is possible
	// in edge cases where the partner crashes during operation. The session
	// creation in this case ignores the new session, but the confirmation
	// message is still sent so the partner will know the session is confirmed
	if duplicate {
		jww.INFO.Printf("New session from Key Exchange Trigger to "+
			"create session %s for partner %s is a duplicate, request ignored",
			newSession.GetID(), request.Sender)
	}

	//Send the Confirmation Message
	//build the payload
	payload, err := proto.Marshal(&RekeyConfirm{
		SessionID: newSession.GetTrigger().Marshal(),
	})

	//If the payload cannot be marshaled, panic
	if err != nil {
		jww.FATAL.Printf("Failed to marshal payload for Key "+
			"Negotation Confirmation with %s", newSession.GetPartner())
	}

	//build the message
	m := message.Send{
		Recipient:   newSession.GetPartner(),
		Payload:     payload,
		MessageType: message.KeyExchangeConfirm,
	}

	//send the message under the key exchange
	e2eParams := params.GetDefaultE2E()
	cmixParams := params.GetDefaultCMIX()

	rounds, err := ctx.Manager.SendE2E(m, e2eParams, cmixParams)

}

func unmarshalKeyExchangeTrigger(grp *cyclic.Group, payload []byte) (e2e.SessionID,
	*cyclic.Int, error) {

	msg := &RekeyTrigger{}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return e2e.SessionID{}, nil, errors.Errorf("Failed to "+
			"unmarshal payload: %s", err)
	}

	oldSessionID := e2e.SessionID{}
	if err := oldSessionID.Unmarshal(msg.SessionID); err != nil {
		return e2e.SessionID{}, nil, errors.Errorf("Failed to unmarshal"+
			" sessionID: %s", err)
	}

	if !grp.BytesInside(msg.PublicKey) {
		return e2e.SessionID{}, nil, errors.Errorf("Public key not in e2e group; PublicKey %v",
			msg.PublicKey)
	}

	return oldSessionID, grp.NewIntFromBytes(msg.PublicKey), nil
}
