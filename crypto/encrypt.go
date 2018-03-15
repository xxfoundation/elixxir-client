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
)

func Encrypt(g *cyclic.Group, message *globals.Message) *globals.
	MessageBytes {

	keys := globals.Session.GetKeys()

	globals.MakeInitVect(message.GetPayloadInitVector())
	globals.MakeInitVect(message.GetRecipientInitVector())

	payloadMicList :=
		[][]byte{message.GetPayloadInitVector().LeftpadBytes(globals.IV_LEN),
			message.GetSenderID().LeftpadBytes(globals.SID_LEN),
			message.GetPayload().LeftpadBytes(globals.PAYLOAD_LEN),
		}

	message.GetPayloadMIC().SetBytes(verification.GenerateMIC(payloadMicList,
		globals.PMIC_LEN))

	recipientMicList :=
		[][]byte{message.GetRecipientInitVector().LeftpadBytes(globals.IV_LEN),
			message.GetRecipientID().LeftpadBytes(globals.RID_LEN),
		}

	message.GetRecipientMIC().SetBytes(verification.GenerateMIC(recipientMicList,
		globals.RMIC_LEN))

	result := message.ConstructMessageBytes()

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

	return result
}
