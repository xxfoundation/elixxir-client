////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/crypto/verification"
)

func Encrypt(g *cyclic.Group, message *format.Message) *format.MessageSerial {

	keys := globals.Session.GetKeys()

	globals.MakeInitVect(message.GetPayloadInitVect())
	globals.MakeInitVect(message.GetRecipientInitVect())

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

	// TODO move this allocation somewhere sensible
	sharedKeyStorage := make([]byte, 0, 8192)

	// generate the product of the inverse transmission keys for encryption
	sharedTransmissionKey := cyclic.NewMaxInt()
	inverseTransmissionKeys := cyclic.NewInt(1)
	for _, key := range keys {
		// modify keys for next node
		forward.GenerateSharedKey(g, key.TransmissionKeys.Base,
			key.TransmissionKeys.Recursive, sharedTransmissionKey,
			sharedKeyStorage)
		g.Inverse(sharedTransmissionKey, sharedTransmissionKey)
		g.Mul(inverseTransmissionKeys, sharedTransmissionKey,
			inverseTransmissionKeys)

	}

	// perform the encryption
	g.Mul(result.Payload, inverseTransmissionKeys, result.Payload)
	g.Mul(result.Recipient, inverseTransmissionKeys, result.Recipient)

	return &result
}
