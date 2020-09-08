package message

import "gitlab.com/xx_network/primitives/id"

type Send struct {
	Recipient   *id.ID
	Payload     []byte
	MessageType Type
}
