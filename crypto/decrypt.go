////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"errors"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/crypto/verification"
)

func Decrypt(g *cyclic.Group, encryptedMessage *globals.MessageBytes) (
	*globals.Message, error) {

	var err error

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
	g.Mul(encryptedMessage.Payload, inverseReceptionKeys, encryptedMessage.Payload)
	g.Mul(encryptedMessage.Recipient, inverseReceptionKeys, encryptedMessage.Recipient)

	// unpack the message from a MessageBytes
	message := encryptedMessage.DeconstructMessageBytes()

	payloadMicList :=
		[][]byte{message.GetPayloadInitVector().LeftpadBytes(globals.IV_LEN),
			message.GetSenderID().LeftpadBytes(globals.SID_LEN),
			message.GetPayload().LeftpadBytes(globals.PAYLOAD_LEN),
		}

	success := verification.CheckMic(payloadMicList,
		message.GetPayloadMIC().LeftpadBytes(globals.PMIC_LEN))

	if !success {
		err = errors.New("MIC did not match")
	}

	return message, err
}
