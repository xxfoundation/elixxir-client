////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
)

// adminListener adheres to the [broadcast.ListenerFunc] interface and is used
// when admin messages are received on the channel.
type adminListener struct {
	chID    *id.ID
	trigger triggerAdminEventFunc
}

// Listen is called when a message is received for the admin listener
func (al *adminListener) Listen(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	// Remove the padding
	payloadUnpadded, err := broadcast.DecodeSizedBroadcast(payload)
	if err != nil {
		jww.WARN.Printf(
			"Failed to strip the padding on User Message on channel %s", al.chID)
		return
	}

	// Get the message ID
	msgID := channel.MakeMessageID(payloadUnpadded)

	// Decode the message as a channel message
	cm := &ChannelMessage{}
	if err = proto.Unmarshal(payloadUnpadded, cm); err != nil {
		jww.WARN.Printf("Failed to unmarshal Channel Message from Admin on "+
			"channel %s", al.chID)
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure that the message is not a replay
	if id.Round(cm.RoundID) != round.ID {
		jww.WARN.Printf("The round message %s send on %s referenced "+
			"(%d) was not the same as the round the message was found on (%d)",
			msgID, al.chID, cm.RoundID, round.ID)
		return
	}

	// Submit the message to the event model for listening
	al.trigger(al.chID, cm, msgID, receptionID, round)

	return
}
