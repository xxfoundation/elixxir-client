package parse

import (
	"gitlab.com/privategrity/client/user"
)

type Message struct {
	TypedBody
	Sender   user.ID
	Receiver user.ID
}
