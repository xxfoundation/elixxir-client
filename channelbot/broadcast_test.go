package channelbot

import (
	"testing"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/cyclic"
)

var broadcastedRecipients []uint64

type TestSender struct{}

func (s TestSender) Send(messageInterface format.MessageInterface) {
	broadcastedRecipients = append(broadcastedRecipients,
		cyclic.NewIntFromBytes(messageInterface.GetRecipient()).Uint64())
}

func TestBroadcastMessage(t *testing.T) {
	broadcastedRecipients = make([]uint64, 0)

	// Make sure that when a message is broadcast,
	// it gets send to the right people
	// Only powers of two will be subscribers
	sender := uint64(4)
	subscribers = []uint64{1, 2, 4, 8, 16}
	message := "This cheese is neat"
	BroadcastMessage(&api.APIMessage{message, sender, 30}, &TestSender{}, 0)
	for i := range subscribers {
		found := false
		// Each subscriber should be in the list of recipients that received
		// the broadcast
		for j := range broadcastedRecipients {
			if broadcastedRecipients[j] == subscribers[i] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find %v in the list of users who received the"+
				" broadcast", subscribers[i])
		}
	}
}
