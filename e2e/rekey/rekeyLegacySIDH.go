////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/cmix"
	session "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	commsNetwork "gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/states"
)

func negotiateLegacySIDH(instance *commsNetwork.Instance, grp *cyclic.Group, sendE2E E2eSender,
	param Params, sess *session.Session, sendTimeout time.Duration) error {

	// Note: All new sending sessions are set to "Sending" status by default

	//generate public key
	pubKey := diffieHellman.GeneratePublicKey(sess.GetMyPrivKey(), grp)

	sidhPrivKey := sess.GetMySIDHPrivKey()
	sidhPubKey := util.NewSIDHPublicKey(sidhPrivKey.Variant())
	sidhPrivKey.GeneratePublicKey(sidhPubKey)
	sidhPubKeyBytes := make([]byte, sidhPubKey.Size()+1)
	sidhPubKeyBytes[0] = byte(sidhPubKey.Variant())
	sidhPubKey.Export(sidhPubKeyBytes[1:])

	//build the payload
	payload, err := proto.Marshal(&RekeyTrigger{
		PublicKey:     pubKey.Bytes(),
		SidhPublicKey: sidhPubKeyBytes,
		SessionID:     sess.GetSource().Marshal(),
	})

	//If the payload cannot be marshaled, panic
	if err != nil {
		jww.FATAL.Printf("[REKEY] Failed to marshal payload for Key "+
			"Negotiation Trigger with %s", sess.GetPartner())
	}

	//send the message under the key exchange
	params := cmix.GetDefaultCMIXParams()
	params.DebugTag = "kx.Request"

	// fixme: should this use the key residue?
	sendReport, err := sendE2E(param.Trigger, sess.GetPartner(),
		payload, params)
	// If the send fails, returns the error so it can be handled. The caller
	// should ensure the calling session is in a state where the Rekey will
	// be triggered next time a key is used
	if err != nil {
		return errors.Errorf(
			"[REKEY] Failed to send the key negotiation message "+
				"for %s: %s", sess, err)
	}

	//create the runner which will handle the result of sending the messages
	sendResults := make(chan ds.EventReturn, len(sendReport.RoundList))

	//Register the event for all rounds
	roundEvents := instance.GetRoundEvents()
	for _, r := range sendReport.RoundList {
		roundEvents.AddRoundEventChan(r, sendResults, sendTimeout,
			states.COMPLETED, states.FAILED)
	}

	//Wait until the result tracking responds
	success, numRoundFail, numTimeOut := cmix.TrackResults(sendResults,
		len(sendReport.RoundList))

	// If a single partition of the Key Negotiation request does not
	// transmit, the partner cannot read the result. Log the error and set
	// the session as unconfirmed so it will re-trigger the negotiation
	if !success {
		_ = sess.TrySetNegotiationStatus(session.Unconfirmed)
		return errors.Errorf("[REKEY] Key Negotiation rekey for %s failed to "+
			"transmit %v/%v paritions: %v round failures, %v timeouts, msgID: %s",
			sess, numRoundFail+numTimeOut, len(sendReport.RoundList), numRoundFail,
			numTimeOut, sendReport.MessageId)
	}

	// otherwise, the transmission is a success and this should be denoted
	// in the session and the log
	jww.INFO.Printf("[REKEY] Key Negotiation rekey transmission for %s, msgID %s successful",
		sess, sendReport.MessageId)
	err = sess.TrySetNegotiationStatus(session.Sent)
	if err != nil {
		if sess.NegotiationStatus() == session.NewSessionTriggered {
			msg := fmt.Sprintf("All channels exhausted for %s, "+
				"rekey impossible.", sess)
			return errors.WithMessage(err, msg)
		}
	}
	return err
}
