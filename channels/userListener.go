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
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

// the userListener adheres to the [broadcast.ListenerFunc] interface and is
// used when user messages are received on the channel
type userListener struct {
	name      NameService
	chID      *id.ID
	trigger   triggerEventFunc
	checkSent messageReceiveFunc
}

// Listen is called when a message is received for the user listener
func (ul *userListener) Listen(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	//Decode the message as a user message
	umi, err := unmarshalUserMessageInternal(payload, ul.chID)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal User Message on "+
			"channel %s", ul.chID)
		return
	}

	um := umi.GetUserMessage()
	cm := umi.GetChannelMessage()
	msgID := umi.GetMessageID()

	//check if we sent the message, ignore triggering if we sent
	if ul.checkSent(msgID, round) {
		return
	}

	/*CRYPTOGRAPHICALLY RELEVANT CHECKS*/

	// check the round to ensure the message is not a replay
	if id.Round(cm.RoundID) != round.ID {
		jww.WARN.Printf("The round message %s send on %d referenced "+
			"(%d) was not the same as the round the message was found on (%d)",
			msgID, ul.chID, cm.RoundID, round.ID)
		return
	}

	// check that the user properly signed the message
	if !ed25519.Verify(um.ECCPublicKey, um.Message, um.Signature) {
		jww.WARN.Printf("Message %s on channel %s purportedly from %s "+
			"failed its user signature with signature %v", msgID,
			ul.chID, cm.Nickname, um.Signature)
		return
	}

	// Modify the timestamp to reduce the chance message order will be ambiguous
	ts := mutateTimestamp(round.Timestamps[states.QUEUED], msgID)

	//TODO: Processing of the message relative to admin commands will be here

	//Submit the message to the event model for listening
	if uuid, err := ul.trigger(ul.chID, umi, ts, receptionID, round,
		Delivered); err != nil {
		jww.WARN.Printf("Error in passing off trigger for "+
			"message (UUID: %d): %+v", uuid, err)
	}

	return
}
