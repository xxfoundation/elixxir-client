package switchboard

import (
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/xx_network/primitives/id"
)

// ID to respond to any message type
const AnyType = message.NoType

//ID to respond to any user
func AnyUser() *id.ID {
	return &id.ZeroUser
}
