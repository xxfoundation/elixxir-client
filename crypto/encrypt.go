package crypto

import (
	"encoding/binary"
	"gitlab.com/privategrity/client/globals"
)

//De constructs message
func Encrypt(message *globals.Message, recipientID uint64) (*[]byte, *[]byte) {
	payload := message.DeconstructMessageFromBytes()
	recipient := make([]byte, 504)

	recparr := make([]byte, 8)

	binary.BigEndian.PutUint64(recparr, recipientID)

	recipient = append(recipient, recparr...)

	return payload, &recipient
}
