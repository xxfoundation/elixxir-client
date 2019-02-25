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
	messageRecipient := cyclic.NewIntFromBytes(cmixMsg.AssociatedData)

	// perform the decryption
	g.Mul(messagePayload, key, messagePayload)
	g.Mul(messageRecipient, key, messageRecipient)

	// unpack the message from a MessageBytes
	decryptedMessage := format.DeserializeMessage(format.MessageSerial{
		MessagePayload:   messagePayload.LeftpadBytes(format.TOTAL_LEN),
		RecipientPayload: messageRecipient.LeftpadBytes(format.TOTAL_LEN),
	})

	payloadMicList :=
		[][]byte{decryptedMessage.GetMessageInitVect(),
			decryptedMessage.GetSenderID(),
			decryptedMessage.GetData(),
		}

	// Note: Don't check the recipient MIC here. The recipient MIC is currently
	// only for the server to know who to send the message to. If the message
	// has ended up here, it's probably because it's for you. So don't worry
	// about it!
	// FIXME: This should not be done here. Do it as part of the receive/display.
	success := verification.CheckMic(payloadMicList,
		decryptedMessage.GetPayloadMIC())

	if !success {
		err = errors.New("Payload MIC did not match")
	}

	return &decryptedMessage, err
}
