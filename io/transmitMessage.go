package io

import (
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
)

func TransmitMessage(addr string, messageBytes, recipientBytes *[]byte) {

	cmixmsg := &pb.CmixMessage{
		MessagePayload: *messageBytes,
		RecipientID:    *recipientBytes,
	}

	message.SendMessageToServer(addr, cmixmsg)

}
