////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"testing"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/cyclic"
)

var broadcastedRecipients map[uint64]struct{}

type TestSender struct{}

func (s TestSender) Send(messageInterface format.MessageInterface) {
	// put the recipient in the map of recipients
	broadcastedRecipients[cyclic.NewIntFromBytes(messageInterface.GetRecipient()).Uint64()] = struct{}{}
}

func TestBroadcastMessage(t *testing.T) {
	broadcastedRecipients = make(map[uint64]struct{})

	// Make sure that when a message is broadcast,
	// it gets send to the right people
	// Only powers of two will be subscribers
	sender := uint64(4)
	users = map[uint64]AccessControl{
		1:  &OwnerAccess{},
		2:  &OwnerAccess{},
		4:  &OwnerAccess{},
		8:  &OwnerAccess{},
		16: &OwnerAccess{},
	}

	message := "This cheese is neat"
	BroadcastMessage(&api.APIMessage{message, sender, 30}, &TestSender{}, 0)
	for i := range users {
		// Each subscriber should be in the map of recipients that received
		// the broadcast
		_, found := broadcastedRecipients[i]

		if !found {
			t.Errorf("Couldn't find %v in the map of users who received the"+
				" broadcast", users[i])
		}
	}
}
