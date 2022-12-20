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
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// adminListener adheres to the [broadcast.ListenerFunc] interface and is used
// when admin messages are received on the channel.
type adminListener struct {
	chID    *id.ID
	trigger triggerAdminEventFunc
}

// Listen is called when a message is received for the admin listener.
func (al *adminListener) Listen(payload, encryptedPayload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// Get the message ID
	messageID := channel.MakeMessageID(payload, al.chID)

	// Decode the message as a channel message
	cm := &ChannelMessage{}
	if err := proto.Unmarshal(payload, cm); err != nil {
		jww.WARN.Printf("[CH] Failed to unmarshal Channel Message from Admin "+
			"on channel %s", al.chID)
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// No timestamp vetting for admins
	ts := time.Unix(0, cm.LocalTimestamp)

	// Submit the message to the event model for listening
	uuid, err := al.trigger(al.chID, cm, encryptedPayload, ts, messageID,
		receptionID, round, Delivered)
	if err != nil {
		jww.WARN.Printf("[CH] Error in passing off trigger for admin "+
			"message (UUID: %d): %+v", uuid, err)
	}

	return
}
