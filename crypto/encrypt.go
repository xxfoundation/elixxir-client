////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/crypto/verification"
)

// Encrypt uses the encryption key to encrypt a message
func Encrypt(key *cyclic.Int, g *cyclic.Group, message *format.Message) *format.MessageSerial {

	// TODO: This is all MIC code and should be moved outside the encrypt
	//       function.
	MakeInitVect(message.GetPayloadInitVect())
	MakeInitVect(message.GetRecipientInitVect())

	payloadMicList :=
		[][]byte{message.GetPayloadInitVect().LeftpadBytes(format.PIV_LEN),
			message.GetSenderID().LeftpadBytes(format.SID_LEN),
			message.GetData().LeftpadBytes(format.DATA_LEN),
		}

	message.GetPayloadMIC().SetBytes(verification.GenerateMIC(payloadMicList,
		format.PMIC_LEN))

	recipientMicList :=
		[][]byte{message.GetRecipientInitVect().LeftpadBytes(format.RIV_LEN),
			message.GetRecipientID().LeftpadBytes(format.RID_LEN),
		}

	message.GetRecipientMIC().SetBytes(verification.GenerateMIC(recipientMicList,
		format.RMIC_LEN))

	result := message.SerializeMessage()

	// perform the encryption
	g.Mul(result.Payload, key, result.Payload)
	g.Mul(result.Recipient, key, result.Recipient)

	return &result
}
