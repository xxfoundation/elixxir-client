package crypto

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
)

func Decrypt(g *cyclic.Group, message *globals.MessageBytes) *globals.Message {

	// Get inverse reception key to decrypt the message
	keys := globals.Session.GetKeys()
	// TODO move this allocation somewhere sensible
	sharedKeyStorage := make([]byte, 0, 8192)

	// generate the product of the inverse transmission keys for encryption
	sharedReceptionKey := cyclic.NewMaxInt()
	inverseReceptionKeys := cyclic.NewInt(1)
	for _, key := range keys {
		// modify key for the next node
		forward.GenerateSharedKey(g, key.ReceptionKeys.Base,
			key.ReceptionKeys.Recursive, sharedReceptionKey, sharedKeyStorage)
		g.Inverse(sharedReceptionKey, sharedReceptionKey)
		g.Mul(inverseReceptionKeys, sharedReceptionKey, inverseReceptionKeys)
	}

	// perform the decryption
	g.Mul(message.Payload, inverseReceptionKeys, message.Payload)
	g.Mul(message.Recipient, inverseReceptionKeys, message.Recipient)

	// unpack the message from a MessageBytes
	result := message.DeconstructMessageBytes()
	return result
}
