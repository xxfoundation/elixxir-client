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
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/crypto/verification"
)

// Decrypt decrypts messages
func Decrypt(key *cyclic.Int, g *cyclic.Group, cmixMsg *pb.CmixMessage) (
	*format.Message, error) {

	var err error

	// Receive and decrypt a message
	message := &format.MessageSerial{
		Payload:   cyclic.NewIntFromBytes(cmixMsg.MessagePayload),
		Recipient: cyclic.NewIntFromBytes(cmixMsg.RecipientID),
	}

	// perform the decryption
	g.Mul(message.Payload, key, message.Payload)
	g.Mul(message.Recipient, key, message.Recipient)

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
