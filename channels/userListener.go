////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// userListener adheres to the [broadcast.ListenerFunc] interface and is used
// when user messages are received on the channel.
type userListener struct {
	name      NameService
	chID      *id.ID
	trigger   triggerEventFunc
	checkSent messageReceiveFunc
}

// Listen is called when a message is received for the user listener.
func (ul *userListener) Listen(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	// Decode the message as a user message
	umi, err := unmarshalUserMessageInternal(payload, ul.chID)
	if err != nil {
		jww.WARN.Printf(
			"Failed to unmarshal User Message on channel %s", ul.chID)
		return
	}

	um := umi.GetUserMessage()
	cm := umi.GetChannelMessage()
	msgID := umi.GetMessageID()

	// Check if we sent the message and ignore triggering if we sent
	if ul.checkSent(msgID, round) {
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure the message is not a replay
	if id.Round(cm.RoundID) != round.ID {
		jww.WARN.Printf("The round message %s send on %d referenced "+
			"(%d) was not the same as the round the message was found on (%d)",
			msgID, ul.chID, cm.RoundID, round.ID)
		return
	}

	// Check that the user properly signed the message
	if !ed25519.Verify(um.ECCPublicKey, um.Message, um.Signature) {
		jww.WARN.Printf("Message %s on channel %s purportedly from %s "+
			"failed its user signature with signature %v", msgID,
			ul.chID, cm.Nickname, um.Signature)
		return
	}

	// Replace the timestamp on the message if it is outside the allowable range
	ts := vetTimestamp(
		time.Unix(0, cm.LocalTimestamp), round.Timestamps[states.QUEUED], msgID)

	// TODO: Processing of the message relative to admin commands will be here.

	// Submit the message to the event model for listening
	uuid, err := ul.trigger(ul.chID, umi, ts, receptionID, round, Delivered)
	if err != nil {
		jww.WARN.Printf("Error in passing off trigger for "+
			"message (UUID: %d): %+v", uuid, err)
	}

	return
}
