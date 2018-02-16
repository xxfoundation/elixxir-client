package crypto

import (
	"gitlab.com/privategrity/client/globals"
)

//De constructs message
func Decrypt(message *[]byte) *globals.Message {
	return globals.ConstructMessageFromBytes(message)
}
