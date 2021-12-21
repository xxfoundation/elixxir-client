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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/states"
)

const (
	errBadTrigger = "non-e2e trigger from partner %s"
	errUnknown    = "unknown trigger from partner %s"
	errFailed     = "Failed to handle rekey trigger: %s"
)

func startTrigger(sess *storage.Session, net interfaces.NetworkManager,
	c chan message.Receive, stop *stoppable.Single, params params.Rekey, cleanup func()) {
	for {
		select {
		case <-stop.Quit():
			cleanup()
			stop.ToStopped()
			return
		case request := <-c:
			go func() {
				err := handleTrigger(sess, net, request, params, stop)
				if err != nil {
					jww.ERROR.Printf(errFailed, err)
				}
			}()
		}
	}
}

func handleTrigger(sess *storage.Session, net interfaces.NetworkManager,
	request message.Receive, param params.Rekey, stop *stoppable.Single) error {
	//ensure the message was encrypted properly
	if request.Encryption != message.E2E {
		errMsg := fmt.Sprintf(errBadTrigger, request.Sender)
		jww.ERROR.Printf(errMsg)
		return errors.New(errMsg)
	}

	//Get the partner
	partner, err := sess.E2e().GetPartner(request.Sender)
	if err != nil {
		errMsg := fmt.Sprintf(errUnknown, request.Sender)
		jww.ERROR.Printf(errMsg)
		return errors.New(errMsg)
	}

	//unmarshal the message
	oldSessionID, PartnerPublicKey, err := unmarshalSource(
		sess.E2e().GetGroup(), request.Payload)
	if err != nil {
		jww.ERROR.Printf("could not unmarshal partner %s: %s",
			request.Sender, err)
		return err
	}

	//get the old session which triggered the exchange
	oldSession := partner.GetSendSession(oldSessionID)
	if oldSession == nil {
		err := errors.Errorf("no session %s for partner %s: %s",
			oldSession, request.Sender, err)
		jww.ERROR.Printf(err.Error())
		return err
	}

	//create the new session
	session, duplicate := partner.NewReceiveSession(PartnerPublicKey,
		sess.E2e().GetE2ESessionParams(), oldSession)
	// new session being nil means the session was a duplicate. This is possible
	// in edge cases where the partner crashes during operation. The session
	// creation in this case ignores the new session, but the confirmation
	// message is still sent so the partner will know the session is confirmed
	if duplicate {
		jww.INFO.Printf("New session from Key Exchange Trigger to "+
			"create session %s for partner %s is a duplicate, request ignored",
			session.GetID(), request.Sender)
	} else {
		// if the session is new, attempt to trigger garbled message processing
		// automatically skips if there is contention
		net.CheckGarbledMessages()
	}

	//Send the Confirmation Message
	//build the payload
	payload, err := proto.Marshal(&RekeyConfirm{
		SessionID: session.GetSource().Marshal(),
	})

	//If the payload cannot be marshaled, panic
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal payload for Key "+
			"Negotation Confirmation with %s", session.GetPartner())
	}

	//build the message
	m := message.Send{
		Recipient:   session.GetPartner(),
		Payload:     payload,
		MessageType: message.KeyExchangeConfirm,
	}

	//send the message under the key exchange
	e2eParams := params.GetDefaultE2E()
	e2eParams.IdentityPreimage = partner.GetSilentPreimage()

	// store in critical messages buffer first to ensure it is resent if the
	// send fails
	sess.GetCriticalMessages().AddProcessing(m, e2eParams)

	rounds, _, _, err := net.SendE2E(m, e2eParams, stop)
	if err != nil {
		return err
	}

	//Register the event for all rounds
	sendResults := make(chan ds.EventReturn, len(rounds))
	roundEvents := net.GetInstance().GetRoundEvents()
	for _, r := range rounds {
		roundEvents.AddRoundEventChan(r, sendResults, param.RoundTimeout,
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
			session, numRoundFail+numTimeOut, len(rounds), numRoundFail,
			numTimeOut)
		sess.GetCriticalMessages().Failed(m, e2eParams)
		return nil
	}

	// otherwise, the transmission is a success and this should be denoted
	// in the session and the log
	sess.GetCriticalMessages().Succeeded(m, e2eParams)
	jww.INFO.Printf("Key Negotiation transmission for %s successfully",
		session)

	return nil
}

func unmarshalSource(grp *cyclic.Group, payload []byte) (e2e.SessionID,
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

	// checking it is inside the group is necessary because otherwise the
	// creation of the cyclic int will crash below
	if !grp.BytesInside(msg.PublicKey) {
		return e2e.SessionID{}, nil, errors.Errorf("Public key not in e2e group; PublicKey %v",
			msg.PublicKey)
	}

	return oldSessionID, grp.NewIntFromBytes(msg.PublicKey), nil
}
