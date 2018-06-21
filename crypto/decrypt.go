////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"errors"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/crypto/verification"
	"gitlab.com/privategrity/client/user"
)

// Decrypt decrypts messages
func Decrypt(g *cyclic.Group, cmixMsg *pb.CmixMessage) (
	*format.Message, error) {

	var err error

	// Receive and decrypt a message
	message := &format.MessageSerial{
		Payload:   cyclic.NewIntFromBytes(cmixMsg.MessagePayload),
		Recipient: cyclic.NewIntFromBytes(cmixMsg.RecipientID),
	}

	// Get inverse reception key to decrypt the message
	keys := user.TheSession.GetKeys()
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
			decryptedMessage.GetData().LeftpadBytes(format.DATA_LEN),
		}

	// FIXME: This should not be done here. Do it as part of the receive/display.
	success := verification.CheckMic(payloadMicList,
		decryptedMessage.GetPayloadMIC().LeftpadBytes(format.PMIC_LEN))

	if !success {
		err = errors.New("MIC did not match")
	}

	return &decryptedMessage, err
}
