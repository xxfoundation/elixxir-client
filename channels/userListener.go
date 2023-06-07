////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
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
func (ul *userListener) Listen(payload, encryptedPayload []byte, _ []string,
	metadata [2]byte, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	mt := UnmarshalMessageType(metadata)
	// Decode the message as a user message
	umi, err := unmarshalUserMessageInternal(payload, ul.chID, mt)
	if err != nil {
		jww.WARN.Printf("[CH] Failed to unmarshal User Message on channel %s "+
			"in round %d: %+v", ul.chID, round.ID, err)
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
		jww.WARN.Printf("[CH] Message %s for channel %s referenced round %d, "+
			"but the message was found on round %d",
			msgID, ul.chID, cm.RoundID, round.ID)
		return
	}

	// Check that the user properly signed the message
	if !ed25519.Verify(um.ECCPublicKey, um.Message, um.Signature) {
		jww.WARN.Printf("[CH] Message %s on channel %s purportedly from %s "+
			"failed its user signature with signature %x",
			msgID, ul.chID, cm.Nickname, um.Signature)
		return
	}

	// Replace the timestamp on the message if it is outside the allowable range
	ts := message.VetTimestamp(
		time.Unix(0, cm.LocalTimestamp), round.Timestamps[states.QUEUED], msgID)

	// Submit the message to the event model for listening
	uuid, err := ul.trigger(
		ul.chID, umi, encryptedPayload, ts, receptionID,
		round, Delivered)
	if err != nil {
		jww.WARN.Printf(
			"[CH] Error in passing off trigger for message (UUID: %d): %+v",
			uuid, err)
	}

	return
}
