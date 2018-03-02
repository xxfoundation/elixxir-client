////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/client/globals"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixclient"
)

func TransmitMessage(addr string, messageBytes *globals.MessageBytes) {

	cmixmsg := &pb.CmixMessage{
	    SenderID: 		globals.Session.GetCurrentUser().Id,
		MessagePayload: messageBytes.Payload.Bytes(),
		RecipientID:    messageBytes.Recipient.Bytes(),
	}

	mixclient.SendMessageToServer(addr, cmixmsg)

}
