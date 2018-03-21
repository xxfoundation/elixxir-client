////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/crypto/verification"
	"gitlab.com/privategrity/crypto/format"
)

func Encrypt(g *cyclic.Group, plainMessage *format.Message) *format.
MessageSerial {

	keys := globals.Session.GetKeys()

	globals.MakeInitVect(plainMessage.GetPayloadInitVect())
	globals.MakeInitVect(plainMessage.GetRecipientInitVect())

	payloadMicList :=
		[][]byte{plainMessage.GetPayloadInitVect().LeftpadBytes(format.PIV_LEN),
			plainMessage.GetSenderID().LeftpadBytes(format.SID_LEN),
			plainMessage.GetData().LeftpadBytes(format.DATA_LEN),
		}

	plainMessage.GetPayloadMIC().SetBytes(verification.GenerateMIC(payloadMicList,
		format.PMIC_LEN))

	recipientMicList :=
		[][]byte{plainMessage.GetRecipientInitVect().LeftpadBytes(format.RIV_LEN),
			plainMessage.GetRecipientID().LeftpadBytes(format.RID_LEN),
		}

	plainMessage.GetRecipientMIC().SetBytes(verification.GenerateMIC(recipientMicList,
		format.RMIC_LEN))

	result := plainMessage.SerializeMessage()

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
