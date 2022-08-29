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

// the adminListener adheres to the broadcast listener interface and is used
// when admin messages are received on the channel
type adminListener struct {
	name   NameService
	events *events
	chID   *id.ID
}

func (al *adminListener) Listen(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	//Remove the padding
	payloadUnpadded, err := broadcast.DecodeSizedBroadcast(payload)
	if err != nil {
		jww.WARN.Printf("Failed to strip the padding on User Message "+
			"on channel %s", al.chID)
		return
	}

	//get the message ID
	msgID := channel.MakeMessageID(payloadUnpadded)

	//Decode the message as a channel message
	var cm *ChannelMessage
	if err = proto.Unmarshal(payloadUnpadded, cm); err != nil {
		jww.WARN.Printf("Failed to unmarshal Channel Message from Admin"+
			" on channel %s", al.chID)
		return
	}

	/*CRYPTOGRAPHICALLY RELEVANT CHECKS*/

	// check the round to ensure the message is not a replay
	if id.Round(cm.RoundID) != round.ID {
		jww.WARN.Printf("The round message %s send on %s referenced "+
			"(%d) was not the same as the round the message was found on (%d)",
			msgID, al.chID, cm.RoundID, round.ID)
		return
	}

	//Submit the message to the event model for listening
	al.events.triggerAdminEvent(al.chID, cm, msgID, receptionID, round)

	return
}
