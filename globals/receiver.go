////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/format"
)

// An object implementing this interface can be called back when the client
// gets a message
type ReceiverProto struct {
	ReceiveMethod func(messageInterface format.MessageInterface)
}

func (rp ReceiverProto) Receive(messageInterface format.MessageInterface) {
	rp.ReceiveMethod(messageInterface)
}

var CurrentReceiver *ReceiverProto

func Receive(message format.MessageInterface) {
	CurrentReceiver.Receive(message)
}
