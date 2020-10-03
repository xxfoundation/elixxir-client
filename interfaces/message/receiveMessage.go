package message

import (
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type Receive struct {
	Payload     []byte
	MessageType Type
	Sender      *id.ID
	Timestamp   time.Time
	Encryption  EncryptionType
	ID          e2e.MessageID
}
