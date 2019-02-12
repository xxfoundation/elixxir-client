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
	MakeInitVect(message.GetPayloadInitVect())
	MakeInitVect(message.GetRecipientInitVect())

	payloadMicList :=
		[][]byte{message.GetPayloadInitVect(),
			message.GetSenderID(),
			message.GetData(),
		}

	payloadMic := verification.GenerateMIC(payloadMicList, format.PMIC_LEN)
	copy(message.GetPayloadMIC(), payloadMic)

	recipientMicList :=
		[][]byte{message.GetRecipientInitVect(),
			message.GetRecipientID(),
		}

	copy(message.GetRecipientMIC(), verification.GenerateMIC(recipientMicList, format.RMIC_LEN))

	result := message.SerializeMessage()

	// perform the encryption
	resultPayload := cyclic.NewIntFromBytes(result.Payload)
	resultRecipient := cyclic.NewIntFromBytes(result.Recipient)
	g.Mul(resultPayload, key, resultPayload)
	g.Mul(resultRecipient, key, resultRecipient)

	return &result
}
