package message

import (
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type Receive struct {
	Recipient   *id.ID
	Payload     []byte
	MessageType Type
	Sender      *id.ID
	Timestamp   time.Time
	Encryption  EncryptionType
}
