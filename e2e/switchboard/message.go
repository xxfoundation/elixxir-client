package switchboard

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type Receive struct {
	MessageType catalog.MessageType
	ID          e2e.MessageID
	Payload     []byte

	Sender         *id.ID
	RecipientID    *id.ID
	EphemeralID    ephemeral.Id
	RoundId        id.Round
	RoundTimestamp time.Time
	Timestamp      time.Time // Message timestamp of when the user sent
	Encrypted      bool
}
