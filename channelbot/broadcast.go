package channelbot

import (
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/crypto/cyclic"
)

type Sender interface {
	Send(messageInterface format.MessageInterface)
}

type APISender struct {}

func (s APISender) Send(messageInterface format.MessageInterface) {
	api.Send(messageInterface)
}

func BroadcastMessage(message format.MessageInterface, sendFunc Sender,
	senderID uint64) {
	speakerID := cyclic.NewIntFromBytes(message.GetSender()).Uint64()
	messages := NewSerializedChannelbotMessages(1,
		speakerID, message.GetPayload())

	for _, message := range messages {
		for _, subscriber := range subscribers {
			sendFunc.Send(&api.APIMessage{
				Payload:     message,
				SenderID:    senderID,
				RecipientID: subscriber})
		}
	}
}
