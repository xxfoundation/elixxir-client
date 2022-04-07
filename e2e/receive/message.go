package receive

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/historical"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type Message struct {
	MessageType catalog.MessageType
	ID          e2e.MessageID
	Payload     []byte

	Sender      *id.ID
	RecipientID *id.ID
	EphemeralID ephemeral.Id
	Timestamp   time.Time // Message timestamp of when the user sent

	Encrypted bool

	Round historical.Round
}
