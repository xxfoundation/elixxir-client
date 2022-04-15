///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet"
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
)

func startConfirm(ratchet *ratchet.Ratchet, c chan receive.Message,
	stop *stoppable.Single, cleanup func()) {
	for true {
		select {
		case <-stop.Quit():
			cleanup()
			stop.ToStopped()
			return
		case confirmation := <-c:
			handleConfirm(ratchet, confirmation)
		}
	}
}

func handleConfirm(ratchet *ratchet.Ratchet, confirmation receive.Message) {
	//ensure the message was encrypted properly
	if !confirmation.Encrypted {
		jww.ERROR.Printf(
			"[REKEY] Received non-e2e encrypted Key Exchange "+
				"confirm from partner %s to %s", confirmation.Sender,
			confirmation.RecipientID)
		return
	}

	//get the partner
	partner, err := ratchet.GetPartner(confirmation.Sender)
	if err != nil {
		jww.ERROR.Printf(
			"[REKEY] Received Key Exchange Confirmation with unknown "+
				"partner %s", confirmation.Sender)
		return
	}

	//unmarshal the payload
	confimedSessionID, err := unmarshalConfirm(confirmation.Payload)
	if err != nil {
		jww.ERROR.Printf("[REKEY] Failed to unmarshal Key Exchange Trigger with "+
			"partner %s: %s", confirmation.Sender, err)
		return
	}

	//get the confirmed session
	confirmedSession := partner.GetSendSession(confimedSessionID)
	if confirmedSession == nil {
		jww.ERROR.Printf("[REKEY] Failed to find confirmed session %s from "+
			"partner %s", confimedSessionID, confirmation.Sender)
		return
	}

	// Attempt to confirm the session. if this fails just print to the log.
	// This is expected sometimes because some errors cases can cause multiple
	// sends. For example if the sending device runs out of battery after it
	// sends but before it records the send it will resend on reload
	if err := confirmedSession.TrySetNegotiationStatus(session2.Confirmed); err != nil {
		jww.WARN.Printf("[REKEY] Failed to set the negotiation status for the "+
			"confirmation of session %s from partner %s. This is expected in "+
			"some edge cases but could be a sign of an issue if it persists: %s",
			confirmedSession, partner.PartnerId(), err)
	}

	jww.DEBUG.Printf("[REKEY] handled confirmation for session "+
		"%s from partner %s.", confirmedSession, partner.PartnerId())
}

func unmarshalConfirm(payload []byte) (session2.SessionID, error) {

	msg := &RekeyConfirm{}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return session2.SessionID{}, errors.Errorf("Failed to "+
			"unmarshal payload: %s", err)
	}

	confirmedSessionID := session2.SessionID{}
	if err := confirmedSessionID.Unmarshal(msg.SessionID); err != nil {
		return session2.SessionID{}, errors.Errorf("Failed to unmarshal"+
			" sessionID: %s", err)
	}

	return confirmedSessionID, nil
}
