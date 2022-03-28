///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package keyExchange

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	network2 "gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/states"
	"time"
)

func CheckKeyExchanges(instance *network.Instance, sendE2E interfaces.SendE2E,
	events event.Manager, sess *storage.Session,
	manager *e2e.Manager, sendTimeout time.Duration,
	stop *stoppable.Single) {
	sessions := manager.TriggerNegotiations()
	for _, session := range sessions {
		go trigger(instance, sendE2E, events, sess, manager, session,
			sendTimeout, stop)
	}
}

// There are two types of key negotiations that can be triggered, creating a new
// session and negotiation, or resetting a negotiation for an already created
// session. They run the same negotiation, the former does it on a newly created
// session while the latter on an extant session
func trigger(instance *network.Instance, sendE2E interfaces.SendE2E,
	events event.Manager, sess *storage.Session,
	manager *e2e.Manager, session *e2e.Session,
	sendTimeout time.Duration, stop *stoppable.Single) {
	var negotiatingSession *e2e.Session
	jww.INFO.Printf("[REKEY] Negotiation triggered for session %s with "+
		"status: %s", session, session.NegotiationStatus())
	switch session.NegotiationStatus() {
	// If the passed session is triggering a negotiation on a new session to
	// replace itself, then create the session
	case e2e.NewSessionTriggered:
		//create the session, pass a nil private key to generate a new one
		negotiatingSession = manager.NewSendSession(nil, nil,
			sess.E2e().GetE2ESessionParams())
		//move the state of the triggering session forward
		session.SetNegotiationStatus(e2e.NewSessionCreated)

	// If the session is set to send a negotiation
	case e2e.Sending:
		negotiatingSession = session
	default:
		jww.FATAL.Panicf("[REKEY] Session %s provided invalid e2e "+
			"negotiating status: %s", session, session.NegotiationStatus())
	}

	rekeyPreimage := manager.GetSilentPreimage()

	// send the rekey notification to the partner
	err := negotiate(instance, sendE2E, sess, negotiatingSession,
		sendTimeout, rekeyPreimage, stop)
	// if sending the negotiation fails, revert the state of the session to
	// unconfirmed so it will be triggered in the future
	if err != nil {
		jww.ERROR.Printf("[REKEY] Failed to do Key Negotiation with "+
			"session %s: %s", session, err)
		events.Report(1, "Rekey", "NegotiationFailed", err.Error())
	}
}

func negotiate(instance *network.Instance, sendE2E interfaces.SendE2E,
	sess *storage.Session, session *e2e.Session, sendTimeout time.Duration,
	rekeyPreimage []byte, stop *stoppable.Single) error {
	e2eStore := sess.E2e()

	//generate public key
	pubKey := diffieHellman.GeneratePublicKey(session.GetMyPrivKey(),
		e2eStore.GetGroup())

	sidhPrivKey := session.GetMySIDHPrivKey()
	sidhPubKey := util.NewSIDHPublicKey(sidhPrivKey.Variant())
	sidhPrivKey.GeneratePublicKey(sidhPubKey)
	sidhPubKeyBytes := make([]byte, sidhPubKey.Size()+1)
	sidhPubKeyBytes[0] = byte(sidhPubKey.Variant())
	sidhPubKey.Export(sidhPubKeyBytes[1:])

	//build the payload
	payload, err := proto.Marshal(&RekeyTrigger{
		PublicKey:     pubKey.Bytes(),
		SidhPublicKey: sidhPubKeyBytes,
		SessionID:     session.GetSource().Marshal(),
	})

	//If the payload cannot be marshaled, panic
	if err != nil {
		jww.FATAL.Printf("[REKEY] Failed to marshal payload for Key "+
			"Negotiation Trigger with %s", session.GetPartner())
	}

	//send session
	m := message.Send{
		Recipient:   session.GetPartner(),
		Payload:     payload,
		MessageType: message.KeyExchangeTrigger,
	}

	//send the message under the key exchange
	e2eParams := params.GetDefaultE2E()
	e2eParams.Type = params.KeyExchange
	e2eParams.IdentityPreimage = rekeyPreimage
	e2eParams.DebugTag = "kx.Request"

	rounds, msgID, _, err := sendE2E(m, e2eParams, stop)
	// If the send fails, returns the error so it can be handled. The caller
	// should ensure the calling session is in a state where the Rekey will
	// be triggered next time a key is used
	if err != nil {
		return errors.Errorf(
			"[REKEY] Failed to send the key negotiation message "+
				"for %s: %s", session, err)
	}

	//create the runner which will handle the result of sending the messages
	sendResults := make(chan ds.EventReturn, len(rounds))

	//Register the event for all rounds
	roundEvents := instance.GetRoundEvents()
	for _, r := range rounds {
		roundEvents.AddRoundEventChan(r, sendResults, sendTimeout,
			states.COMPLETED, states.FAILED)
	}

	//Wait until the result tracking responds
	success, numRoundFail, numTimeOut := network2.TrackResults(sendResults,
		len(rounds))

	// If a single partition of the Key Negotiation request does not
	// transmit, the partner cannot read the result. Log the error and set
	// the session as unconfirmed so it will re-trigger the negotiation
	if !success {
		session.SetNegotiationStatus(e2e.Unconfirmed)
		return errors.Errorf("[REKEY] Key Negotiation rekey for %s failed to "+
			"transmit %v/%v paritions: %v round failures, %v timeouts, msgID: %s",
			session, numRoundFail+numTimeOut, len(rounds), numRoundFail,
			numTimeOut, msgID)
	}

	// otherwise, the transmission is a success and this should be denoted
	// in the session and the log
	jww.INFO.Printf("[REKEY] Key Negotiation rekey transmission for %s, msgID %s successful",
		session, msgID)
	err = session.TrySetNegotiationStatus(e2e.Sent)
	if err != nil {
		if session.NegotiationStatus() == e2e.NewSessionTriggered {
			msg := fmt.Sprintf("All channels exhausted for %s, "+
				"rekey impossible.", session)
			return errors.WithMessage(err, msg)
		}
	}
	return err
}
