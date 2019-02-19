////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/verification"
	"gitlab.com/elixxir/primitives/format"
)

// Encrypt uses the encryption key to encrypt a message
func Encrypt(key *cyclic.Int, g *cyclic.Group, message *format.Message) *format.MessageSerial {

	// TODO: This is all MIC code and should be moved outside the encrypt
	//       function.
	MakeInitVect(message.GetMessageInitVect())
	MakeInitVect(message.GetRecipientInitVect())

	payloadMicList :=
		[][]byte{message.GetMessageInitVect(),
			message.GetSenderID(),
			message.GetData(),
		}

	payloadMic := verification.GenerateMIC(payloadMicList, format.MMIC_LEN)
	copy(message.GetPayloadMIC(), payloadMic)

	recipientMicList :=
		[][]byte{message.GetRecipientInitVect(),
			message.GetRecipientID(),
		}

	copy(message.GetRecipientMIC(), verification.GenerateMIC(recipientMicList, format.RMIC_LEN))

	result := message.SerializeMessage()

	// perform the encryption
	resultPayload := cyclic.NewIntFromBytes(result.MessagePayload)
	resultRecipient := cyclic.NewIntFromBytes(result.RecipientPayload)
	g.Mul(resultPayload, key, resultPayload)
	g.Mul(resultRecipient, key, resultRecipient)

	// write back encrypted message into result
	copy(result.MessagePayload, resultPayload.LeftpadBytes(format.TOTAL_LEN))
	copy(result.RecipientPayload, resultRecipient.LeftpadBytes(format.TOTAL_LEN))

	return &result
}
