////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/verification"
	"gitlab.com/elixxir/primitives/format"
)

// Decrypt decrypts messages
func Decrypt(key *cyclic.Int, g *cyclic.Group, cmixMsg *pb.CmixMessage) (
	*format.Message, error) {

	var err error

	// Receive and decrypt a message
	messagePayload := cyclic.NewIntFromBytes(cmixMsg.MessagePayload)
	messageRecipient := cyclic.NewIntFromBytes(cmixMsg.RecipientID)

	// perform the decryption
	g.Mul(messagePayload, key, messagePayload)
	g.Mul(messageRecipient, key, messageRecipient)

	// unpack the message from a MessageBytes
	decryptedMessage := format.DeserializeMessage(format.MessageSerial{
		Payload:   messagePayload.LeftpadBytes(format.TOTAL_LEN),
		Recipient: messageRecipient.LeftpadBytes(format.TOTAL_LEN),
	})

	payloadMicList :=
		[][]byte{decryptedMessage.GetPayloadInitVect(),
			decryptedMessage.GetSenderID(),
			decryptedMessage.GetData(),
		}

	// FIXME: This should not be done here. Do it as part of the receive/display.
	success := verification.CheckMic(payloadMicList,
		decryptedMessage.GetPayloadMIC())

	if !success {
		err = errors.New("Payload MIC did not match")
	}

	recipientMicList :=
		[][]byte{decryptedMessage.GetRecipientInitVect(),
			decryptedMessage.GetRecipientID(),
		}

	success = verification.CheckMic(recipientMicList,
		decryptedMessage.GetRecipientMIC())

	if !success {
		if err == nil {
			err = errors.New("Recipient MIC did not match")
		} else {
			err = errors.New("Payload and recipient MIC did not match")
		}
	}

	return &decryptedMessage, err
}
