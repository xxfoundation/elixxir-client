package parse

import "gitlab.com/privategrity/client/globals"

type Message struct {
	TypedBody
	Sender globals.UserID
	Receiver globals.UserID
}
