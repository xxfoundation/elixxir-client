package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/context/utility"
	"gitlab.com/elixxir/client/storage/e2e"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/states"
	"time"
)

func startTrigger(ctx *context.Context, c chan message.Receive,
	stop *stoppable.Single) {
	for true {
		select {
		case <-stop.Quit():
			return
		case request := <-c:
			handleTrigger(ctx, request)
		}
	}
}

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
	oldSessionID, PartnerPublicKey, err := unmarshalTrigger(
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
			"Exchange Trigger from partner %s: %s", oldSessionID, request.Sender,
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

	// store in critical messages buffer first to ensure it is resent if the
	// send fails
	ctx.Session.GetCriticalMessages().AddProcessing(m, e2eParams)

	rounds, err := ctx.Manager.SendE2E(m, e2eParams)

	//Register the event for all rounds
	sendResults := make(chan ds.EventReturn, len(rounds))
	roundEvents := ctx.Manager.GetInstance().GetRoundEvents()
	for _, r := range rounds {
		roundEvents.AddRoundEventChan(r, sendResults, 1*time.Minute,
			states.COMPLETED, states.FAILED)
	}

	//Wait until the result tracking responds
	success, numTimeOut, numRoundFail := utility.TrackResults(sendResults, len(rounds))

	// If a single partition of the Key Negotiation request does not
	// transmit, the partner will not be able to read the confirmation. If
	// such a failure occurs
	if !success {
		jww.ERROR.Printf("Key Negotiation for %s failed to "+
			"transmit %v/%v paritions: %v round failures, %v timeouts",
			newSession, numRoundFail+numTimeOut, len(rounds), numRoundFail,
			numTimeOut)
		newSession.SetNegotiationStatus(e2e.Unconfirmed)
		ctx.Session.GetCriticalMessages().Failed(m)
		return
	}

	// otherwise, the transmission is a success and this should be denoted
	// in the session and the log
	newSession.SetNegotiationStatus(e2e.Sent)
	ctx.Session.GetCriticalMessages().Succeeded(m)
	jww.INFO.Printf("Key Negotiation transmission for %s sucesfull",
		newSession)
}

func unmarshalTrigger(grp *cyclic.Group, payload []byte) (e2e.SessionID,
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
