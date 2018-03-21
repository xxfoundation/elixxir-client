////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/format"
)

type Receiver func (messageInterface format.MessageInterface)

var CurrentReceiver Receiver

func Receive(message format.MessageInterface) {
	CurrentReceiver(message)
}
