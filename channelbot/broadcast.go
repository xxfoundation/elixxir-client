////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/cyclic"
	jww "github.com/spf13/jwalterweatherman"
)

type Sender interface {
	Send(messageInterface format.MessageInterface)
}

func BroadcastMessage(message format.MessageInterface, sendFunc Sender,
	senderID uint64) {
	speakerID := cyclic.NewIntFromBytes(message.GetSender()).Uint64()
	if users[speakerID].CanSend() {
		payloads := NewSerializedChannelbotMessages(1,
			speakerID, message.GetPayload())

		// broadcast the message to all subscribers
		for _, payload := range payloads {
			for subscriber, access := range users {
				// only send to users that can receive
				if access.CanReceive() {
					preparedMessages, err := format.NewMessage(senderID,
						subscriber, payload)
					if err == nil {
						// there should only be one serialized message per slice
						for _, preparedMessage := range preparedMessages {
							sendFunc.Send(preparedMessage)
						}
					} else {
						jww.ERROR.Printf("Couldn't construct format messages" +
							": %v", err.Error())
					}
				}
			}
		}
	}
}
