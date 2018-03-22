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
	"gitlab.com/privategrity/crypto/format"
)

func Decrypt(g *cyclic.Group, message *format.MessageSerial) (
	*format.Message, error) {

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
	g.Mul(message.Payload, inverseReceptionKeys, message.Payload)
	g.Mul(message.Recipient, inverseReceptionKeys, message.Recipient)

	// unpack the message from a MessageBytes
	decryptedMessage := format.DeserializeMessage(*message)

	payloadMicList :=
		[][]byte{decryptedMessage.GetPayloadInitVect().LeftpadBytes(format.PIV_LEN),
			decryptedMessage.GetSenderID().LeftpadBytes(format.SID_LEN),
			decryptedMessage.GetData().LeftpadBytes(format.DATA_END),
		}

	success := verification.CheckMic(payloadMicList,
		decryptedMessage.GetPayloadMIC().LeftpadBytes(format.PMIC_LEN))

	if !success {
		err = errors.New("MIC did not match")
	}

	return &decryptedMessage, err
}
