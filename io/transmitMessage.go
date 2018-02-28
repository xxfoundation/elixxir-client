package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/client/globals"
)

func TransmitMessage(addr string, messageBytes *globals.MessageBytes) {

	cmixmsg := &pb.CmixMessage{
		MessagePayload: messageBytes.Payload.Bytes(),
		RecipientID:    messageBytes.Recipient.Bytes(),
	}

	message.SendMessageToServer(addr, cmixmsg)

}
