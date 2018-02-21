package io

import (
	"gitlab.com/privategrity/client/globals"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
)

func TransmitMessage(addr string, messageBytes, recipientBytes *[]byte) {

	cmixmsg := &pb.CmixMessage{
		SenderID:       globals.Session.GetCurrentUser().Id,
		MessagePayload: *messageBytes,
		RecipientID:    *recipientBytes,
	}

	message.SendMessageToServer(addr, cmixmsg)

}
