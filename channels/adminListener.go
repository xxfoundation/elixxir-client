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
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// adminListener adheres to the [broadcast.ListenerFunc] interface and is used
// when admin messages are received on the channel.
type adminListener struct {
	chID      *id.ID
	trigger   triggerAdminEventFunc
	checkSent messageReceiveFunc
}

// Listen is called when a message is received for the admin listener.
func (al *adminListener) Listen(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// Get the message ID
	msgID := channel.MakeMessageID(payload, al.chID)

	// Decode the message as a channel message
	cm := &ChannelMessage{}
	if err := proto.Unmarshal(payload, cm); err != nil {
		jww.WARN.Printf("Failed to unmarshal Channel Message from Admin on "+
			"channel %s", al.chID)
		return
	}

	// Check if we sent the message, ignore triggering if we sent
	if al.checkSent(msgID, round) {
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure that the message is not a replay
	if id.Round(cm.RoundID) != round.ID {
		jww.WARN.Printf("The round message %s send on %s referenced (%d) was "+
			"not the same as the round the message was found on (%d)",
			msgID, al.chID, cm.RoundID, round.ID)
		return
	}

	// Replace the timestamp on the message if it is outside the allowable range
	ts := vetTimestamp(time.Unix(0, cm.LocalTimestamp),
		round.Timestamps[states.QUEUED], msgID)

	// Submit the message to the event model for listening
	if uuid, err := al.trigger(al.chID, cm, ts, msgID, receptionID,
		round, Delivered); err != nil {
		jww.WARN.Printf("Error in passing off trigger for admin "+
			"message (UUID: %d): %+v", uuid, err)
	}

	return
}
