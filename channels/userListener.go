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
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
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

	//Remove the padding
	payloadUnpadded, err := broadcast.DecodeSizedBroadcast(payload)
	if err != nil {
		jww.WARN.Printf("Failed to strip the padding on User Message "+
			"on channel %s", ul.chID)
		return
	}

	//Decode the message as a user message
	umi, err := unmarshalUserMessageInternal(payloadUnpadded)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal User Message on "+
			"channel %s", ul.chID)
		return
	}

	um := umi.GetUserMessage()
	cm := umi.GetChannelMessage()
	msgID := umi.GetMessageID()

	//check if we sent the message, ignore triggering if we sent
	if ul.checkSent(msgID) {
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

	// check that the username lease is valid
	usernameLeaseEnd := time.Unix(0, um.UsernameLease)
	if !usernameLeaseEnd.After(round.Timestamps[states.QUEUED]) {
		jww.WARN.Printf("Message %s on channel %s purportedly from %s "+
			"has an expired lease, ended %s, round %d was sent at %s", msgID,
			ul.chID, um.Username, usernameLeaseEnd, round.ID,
			round.Timestamps[states.QUEUED])
		return
	}

	// check that the signature from the nameserver is valid
	if !ul.name.ValidateChannelMessage(um.Username,
		time.Unix(0, um.UsernameLease), um.ECCPublicKey, um.ValidationSignature) {
		jww.WARN.Printf("Message %s on channel %s purportedly from %s "+
			"failed the check of its Name Server with signature %v", msgID,
			ul.chID, um.Username, um.ValidationSignature)
		return
	}

	// check that the user properly signed the message
	if !ed25519.Verify(um.ECCPublicKey, um.Message, um.Signature) {
		jww.WARN.Printf("Message %s on channel %s purportedly from %s "+
			"failed its user signature with signature %v", msgID,
			ul.chID, um.Username, um.Signature)
		return
	}

	//TODO: Processing of the message relative to admin commands will be here

	//Submit the message to the event model for listening
	ul.trigger(ul.chID, umi, receptionID, round, Delivered)

	return
}
