package context

import "gitlab.com/xx_network/primitives/id"

type Message struct {
	Recipient   *id.ID
	Payload     []byte
	MessageType int32
}
