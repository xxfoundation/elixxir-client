////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/format"
	"errors"
)

type Receiver func (messageInterface format.MessageInterface)

var currentReceiver Receiver

func UsingReceiver() bool {
	return currentReceiver == nil
}

func SetReceiver(receiver Receiver) error {
	//TODO: Add mutex
	currentReceiver = receiver
	return nil
}

func Receive(message format.MessageInterface) error {
	if currentReceiver != nil {
		currentReceiver(message)
		return nil
	} else {
		return errors.New("Couldn't receive using the receiver: No receiver" +
			" is set")
	}
}
