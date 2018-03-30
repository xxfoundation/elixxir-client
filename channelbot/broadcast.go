package channelbot

import (
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/client/globals"
)

func BroadcastMessage(messageInterface format.MessageInterface) {
	speakerID := cyclic.NewIntFromBytes(messageInterface.GetSender()).Uint64()
	messages := NewSerializedChannelbotMessages(1,
		speakerID, messageInterface.GetPayload())

	for _, message := range messages {
		for _, subscriber := range subscribers {
			api.Send(&api.APIMessage{
				Payload:     message,
				SenderID:    globals.Session.GetCurrentUser().UserID,
				RecipientID: subscriber})
		}
	}
}
