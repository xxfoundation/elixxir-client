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

func TransmitMessage(addr string, messageBytes, recipientBytes *[]byte) {

	cmixmsg := &pb.CmixMessage{
		SenderID:       globals.Session.GetCurrentUser().UID,
		MessagePayload: *messageBytes,
		RecipientID:    *recipientBytes,
	}

	mixclient.SendMessageToServer(addr, cmixmsg)

}
