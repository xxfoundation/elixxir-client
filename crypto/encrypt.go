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
	"gitlab.com/elixxir/crypto/messaging"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/e2e"
	jww "github.com/spf13/jwalterweatherman"
)

// Encrypt uses the encryption key to encrypt the passed message and populate
// the associated data
// You must also encrypt the message for the nodes
func Encrypt(key *cyclic.Int, g *cyclic.Group,
	message *format.Message, e2eKey []byte) (encryptedAssociatedData []byte,
		encryptedPayload []byte) {

	// Key fingerprint is full 256 bits
	keyFp := messaging.NewSalt(csprng.Source(&csprng.SystemRNG{}), 32)
	message.AssociatedData.SetKeyFingerprint(keyFp)

	// Encrypt the timestamp
	// FIXME This needs to be hooked in to the new keying system
	// Currently it is trivial to decrypt this on the other side,
	// or in the middle, because the full AES key is right there in the AD of
	// the plaintext. This is for ease of implementation.
	encryptedTimestamp, err := e2e.EncryptAES256(cyclic.NewIntFromBytes(keyFp),
		message.GetTimestamp())
	if err != nil {
		jww.ERROR.Panicf(err.Error())
	}
	message.SetTimestamp(encryptedTimestamp)

	// MAC is HMAC(key, plaintext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(message.SerializePayload(), e2eKey)
	message.SetMAC(MAC)

	// TODO Make sure all of these fields are properly populated before
	// generating the MIC!
	recipientMicList := [][]byte{
		message.AssociatedData.GetRecipientID(),
		message.AssociatedData.GetKeyFingerprint(),
		message.AssociatedData.GetTimestamp(),
		message.AssociatedData.GetMAC(),
	}
	mic := verification.GenerateMIC(recipientMicList, uint64(format.AD_RMIC_LEN))
	copy(message.GetRecipientMIC(), mic)

	// perform the encryption
	resultPayload := cyclic.NewIntFromBytes(message.SerializePayload())
	resultAssociatedData := cyclic.NewIntFromBytes(message.SerializeAssociatedData())
	g.Mul(resultPayload, key, resultPayload)
	g.Mul(resultAssociatedData, key, resultAssociatedData)

	return resultAssociatedData.LeftpadBytes(uint64(format.TOTAL_LEN)),
		resultPayload.LeftpadBytes(uint64(format.TOTAL_LEN))
}
