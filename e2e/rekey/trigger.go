////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"fmt"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/elixxir/client/v5/e2e/ratchet"
	"gitlab.com/elixxir/client/v5/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v5/e2e/receive"
	"gitlab.com/elixxir/client/v5/stoppable"
	util "gitlab.com/elixxir/client/v5/storage/utility"
	"gitlab.com/elixxir/crypto/cyclic"
)

const (
	errBadTrigger = "non-e2e trigger from partner %s"
	errUnknown    = "unknown trigger from partner %s"
	errFailed     = "Failed to handle rekey trigger: %s"
)

func startTrigger(ratchet *ratchet.Ratchet, sender E2eSender, net cmix.Client,
	grp *cyclic.Group, c chan receive.Message, stop *stoppable.Single, params Params,
	cleanup func()) {
	for {
		select {
		case <-stop.Quit():
			cleanup()
			stop.ToStopped()
			return
		case request := <-c:
			go func() {
				err := handleTrigger(ratchet, sender, net, grp, request, params,
					stop)
				if err != nil {
					jww.ERROR.Printf(errFailed, err)
				}
			}()
		}
	}
}

func handleTrigger(ratchet *ratchet.Ratchet, sender E2eSender,
	net cmix.Client, grp *cyclic.Group, request receive.Message,
	param Params, stop *stoppable.Single) error {

	jww.DEBUG.Printf("[REKEY] handleTrigger(partner: %s)",
		request.Sender)

	//ensure the message was encrypted properly
	if !request.Encrypted {
		errMsg := fmt.Sprintf(errBadTrigger, request.Sender)
		jww.ERROR.Printf(errMsg)
		return errors.New(errMsg)
	}

	//get the partner
	partner, err := ratchet.GetPartner(request.Sender)
	if err != nil {
		errMsg := fmt.Sprintf(errUnknown, request.Sender)
		jww.ERROR.Printf(errMsg)
		return errors.New(errMsg)
	}

	//unmarshal the message
	oldSessionID, PartnerPublicKey, PartnerSIDHPublicKey, err :=
		unmarshalSource(grp, request.Payload)
	if err != nil {
		jww.ERROR.Printf("[REKEY] could not unmarshal partner %s: %s",
			request.Sender, err)
		return err
	}

	//get the old session which triggered the exchange
	oldSession := partner.GetSendSession(oldSessionID)
	if oldSession == nil {
		err := errors.Errorf("[REKEY] no session %s for partner %s: %s",
			oldSession, request.Sender, err)
		jww.ERROR.Printf(err.Error())
		return err
	}

	//create the new session
	sess, duplicate := partner.NewReceiveSession(PartnerPublicKey,
		PartnerSIDHPublicKey, session.GetDefaultParams(),
		oldSession)
	// new session being nil means the session was a duplicate. This is possible
	// in edge cases where the partner crashes during operation. The session
	// creation in this case ignores the new session, but the confirmation
	// message is still sent so the partner will know the session is confirmed
	if duplicate {
		jww.INFO.Printf("[REKEY] New session from Key Exchange Trigger to "+
			"create session %s for partner %s is a duplicate, request ignored",
			sess.GetID(), request.Sender)
	} else {
		// if the session is new, attempt to trigger garbled message processing
		// automatically skips if there is contention
		net.CheckInProgressMessages()
	}

	//Send the Confirmation Message
	// build the payload, note that for confirmations, we need only send the
	// (generated from new keys) session id, which the other side should
	// know about already.
	// When sending a trigger, the source session id is sent instead
	payload, err := proto.Marshal(&RekeyConfirm{
		SessionID: sess.GetID().Marshal(),
	})

	//If the payload cannot be marshaled, panic
	if err != nil {
		jww.FATAL.Panicf("[REKEY] Failed to marshal payload for Key "+
			"Negotation Confirmation with %s", sess.GetPartner())
	}

	//send the trigger confirmation
	params := cmix.GetDefaultCMIXParams()
	params.Critical = true
	//ignore results, the passed sender interface makes it a critical message
	// fixme: should this ignore the error as well?
	_, _ = sender(param.Confirm, request.Sender, payload,
		params)

	return nil
}

func unmarshalSource(grp *cyclic.Group, payload []byte) (session.SessionID,
	*cyclic.Int, *sidh.PublicKey, error) {

	msg := &RekeyTrigger{}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return session.SessionID{}, nil, nil, errors.Errorf(
			"Failed to unmarshal payload: %s", err)
	}

	oldSessionID := session.SessionID{}

	if err := oldSessionID.Unmarshal(msg.SessionID); err != nil {
		return session.SessionID{}, nil, nil, errors.Errorf(
			"Failed to unmarshal sessionID: %s", err)
	}

	// checking it is inside the group is necessary because otherwise the
	// creation of the cyclic int will crash below
	if !grp.BytesInside(msg.PublicKey) {
		return session.SessionID{}, nil, nil, errors.Errorf(
			"Public key not in e2e group; PublicKey %v",
			msg.PublicKey)
	}

	theirSIDHVariant := sidh.KeyVariant(msg.SidhPublicKey[0])
	theirSIDHPubKey := util.NewSIDHPublicKey(theirSIDHVariant)
	theirSIDHPubKey.Import(msg.SidhPublicKey[1:])

	return oldSessionID, grp.NewIntFromBytes(msg.PublicKey),
		theirSIDHPubKey, nil
}
