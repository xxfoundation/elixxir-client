////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/verification"
	"gitlab.com/elixxir/primitives/format"
	"time"
)

// Encrypt uses the encryption key to encrypt the passed message and populate
// the associated data
// You must also encrypt the message for the nodes
func Encrypt(key *cyclic.Int, g *cyclic.Group,
	message *format.Message, e2eKey *cyclic.Int) (encryptedAssociatedData []byte,
	encryptedPayload []byte) {
	e2eKeyBytes := e2eKey.LeftpadBytes(uint64(format.TOTAL_LEN))
	// Key fingerprint is 256 bits given by H(e2ekey)
	// For now use Blake2B
	h, _ := hash.NewCMixHash()
	h.Write(e2eKeyBytes)
	keyFp := format.NewFingerprint(h.Sum(nil))
	message.SetKeyFingerprint(*keyFp)

	// Encrypt the timestamp using the e2ekey
	// TODO BC: this will produce a 32 byte ciphertext, where the first 16 bytes
	// is the IV internally generated AES. This is fine right now since there are 32 bytes
	// of space in Associated Data for the timestamp.
	// If we want to decrease that to 16 bytes, we need to use the key fingerprint
	// as the IV for AES encryption
	// TODO: timestamp like this is kinda hacky, maybe it should be set right here
	// However, this would lead to parts of same message having potentially different timestamps
	// Get relevant bytes from timestamp by unmarshalling and then marshalling again
	timestamp := time.Time{}
	timestamp.UnmarshalBinary(message.GetTimestamp())
	timeBytes, _ := timestamp.MarshalBinary()
	var iv [e2e.AESBlockSize]byte
	copy(iv[:], keyFp[:e2e.AESBlockSize])
	encryptedTimestamp, err := e2e.EncryptAES256WithIV(e2eKeyBytes, iv, timeBytes)
	// Make sure the encrypted timestamp fits
	if len(encryptedTimestamp) != format.AD_TIMESTAMP_LEN || err != nil {
		jww.ERROR.Panicf(err.Error())
	}
	message.SetTimestamp(encryptedTimestamp)

	// E2E encrypt the message
	encPayload, err := e2e.Encrypt(*g, e2eKey, message.GetPayload())
	if len(encPayload) != format.TOTAL_LEN || err != nil {
		jww.ERROR.Panicf(err.Error())
	}
	message.SetPayload(encPayload)

	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, e2eKeyBytes)
	message.SetMAC(MAC)

	recipientMicList := [][]byte{
		message.AssociatedData.GetRecipientID(),
		keyFp[:],
		message.AssociatedData.GetTimestamp(),
		message.AssociatedData.GetMAC(),
	}
	mic := verification.GenerateMIC(recipientMicList, uint64(format.AD_RMIC_LEN))
	message.SetRecipientMIC(mic)

	// perform the CMIX encryption
	resultPayload := g.NewIntFromBytes(message.SerializePayload())
	resultAssociatedData := g.NewIntFromBytes(message.SerializeAssociatedData())
	g.Mul(resultPayload, key, resultPayload)
	g.Mul(resultAssociatedData, key, resultAssociatedData)

	return resultAssociatedData.LeftpadBytes(uint64(format.TOTAL_LEN)),
		resultPayload.LeftpadBytes(uint64(format.TOTAL_LEN))
}
