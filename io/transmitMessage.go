////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/comms/mixclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
)

// Send a cMix message to the server
func TransmitMessage(addr string, messageBytes *globals.MessageBytes) error {

	globals.TransmissionMutex.Lock()

	cmixmsg := &pb.CmixMessage{
		SenderID:       globals.Session.GetCurrentUser().UserID,
		MessagePayload: messageBytes.Payload.Bytes(),
		RecipientID:    messageBytes.Recipient.Bytes(),
	}

	_, err := mixclient.SendMessageToServer(addr, cmixmsg)

	globals.TransmissionMutex.Unlock()

	return err
}
