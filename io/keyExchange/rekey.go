package keyExchange

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/utility"
	"gitlab.com/elixxir/client/storage/e2e"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/states"
	jww "github.com/spf13/jwalterweatherman"
	"time"
)

func CheckKeyExchanges(ctx *context.Context, manager *e2e.Manager) {
	sessions := manager.TriggerNegotiations()
	for _, ses := range sessions {
		locakSes := ses
		go trigger(ctx, manager, locakSes)
	}
}

// There are two types of key negotiations that can be triggered, creating a new
// session and negotiation, or resenting a negotiation for an already created
// session. They run the same negotiation, the former does it on a newly created
// session while the latter on an extand
func trigger(ctx *context.Context, manager *e2e.Manager, session *e2e.Session) {
	var negotiatingSession *e2e.Session
	switch session.ConfirmationStatus() {
	// If the passed session is triggering a negotiation on a new session to
	// replace itself, then create the session
	case e2e.NewSessionTriggered:
		//create the session, pass a nil private key to generate a new one
		negotiatingSession = manager.NewSendSession(nil, e2e.GetDefaultSessionParams())
		//move the state of the triggering session forward
		session.SetNegotiationStatus(e2e.NewSessionCreated)
	// If the session has not successfully negotiated, redo its negotiation
	case e2e.Unconfirmed:
		negotiatingSession = session
	default:
		jww.FATAL.Panicf("Session %s provided invalid e2e "+
			"negotiating status: %s", session, session.ConfirmationStatus())
	}

	// send the rekey notification to the partner
	err := negotiate(ctx, negotiatingSession)
	// if sending the negotiation fails, revert the state of the session to
	// unconfirmed so it will be triggered in the future
	if err != nil {
		jww.ERROR.Printf("Failed to do Key Negotiation: %s", err)
		negotiatingSession.SetNegotiationStatus(e2e.Unconfirmed)
	}
}

func negotiate(ctx *context.Context, session *e2e.Session) error {
	e2eStore := ctx.Session.E2e()

	//generate public key
	pubKey := diffieHellman.GeneratePublicKey(session.GetMyPrivKey(),
		e2eStore.GetGroup())

	//send session
	m := context.Message{
		Recipient:   session.GetPartner(),
		Payload:     pubKey.Bytes(),
		MessageType: 42,
	}

	//send the message under the key exchange
	e2eParams := params.GetDefaultE2E()
	e2eParams.Type = params.KeyExchange
	cmixParams := params.GetDefaultCMIX()
	cmixParams.Retries = 20

	rounds, err := ctx.Manager.SendE2E(m, e2eParams, cmixParams)
	// If the send fails, returns the error so it can be handled. The caller
	// should ensure the calling session is in a state where the Rekey will
	// be triggered next time a key is used
	if err != nil {
		return errors.Errorf("Failed to send the key negotation message "+
			"for %s: %s", session, err)
	}

	//create the runner which will handle the result of sending the messages
	sendResults := make(chan ds.EventReturn, len(rounds))

	//Register the event for all rounds
	roundEvents := ctx.Manager.GetInstance().GetRoundEvents()
	for _, r := range rounds {
		roundEvents.AddRoundEventChan(r, sendResults, 1*time.Minute,
			states.COMPLETED, states.FAILED)
	}

	//Start the thread which will handle the outcome of the send
	go trackNegotiationResult(sendResults, len(rounds), session)
}

func trackNegotiationResult(resultsCh chan ds.EventReturn, numResults int, session *e2e.Session) {
	success, numTimeOut, numRoundFail := utility.TrackResults(resultsCh, numResults)

	// If a single partition of the Key Negotiation request does not
	// transmit, the partner cannot read the result. Log the error and set
	// the session as unconfirmed so it will re-trigger the negotiation
	if !success {
		jww.ERROR.Printf("Key Negotiation for %s failed to "+
			"transmit %v/%v paritions: %v round failures, %v timeouts",
			session, numRoundFail+numTimeOut, numResults, numRoundFail,
			numTimeOut)
		session.SetNegotiationStatus(e2e.Unconfirmed)
		return
	}

	// otherwise, the transmission is a success and this should be denoted
	// in the session and the log
	jww.INFO.Printf("Key Negotiation transmission for %s sucesfull",
		session)
	session.SetNegotiationStatus(e2e.Sent)
}

