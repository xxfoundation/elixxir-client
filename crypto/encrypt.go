////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// CMIX Encrypt performs the encryption
// of the msg to a team of nodes
// It returns a new msg
func CMIXEncrypt(session user.Session,
	salt []byte,
	msg *format.Message) *format.Message {
	// Generate the encryption key
	nodeKeys := session.GetKeys()
	baseKeys := make([]*cyclic.Int, len(nodeKeys))
	for i, key := range nodeKeys {
		baseKeys[i] = key.TransmissionKey
		//TODO: Add KMAC generation here
	}

	return cmix.ClientEncrypt(session.GetGroup(), msg, salt, baseKeys)
}

// E2EEncrypt uses the E2E key to encrypt msg
// to its intended recipient
// It also properly populates the associated data
// It modifies the passed msg instead of returning a new one
func E2EEncrypt(grp *cyclic.Group,
	key *cyclic.Int, keyFP format.Fingerprint,
	msg *format.Message) {
	msg.SetKeyFP(keyFP)

	// Encrypt the timestamp using key
	// Timestamp bytes were previously stored
	// and GO only uses 15 bytes, so use those
	var iv [e2e.AESBlockSize]byte
	copy(iv[:], keyFP[:e2e.AESBlockSize])
	encryptedTimestamp, err :=
		e2e.EncryptAES256WithIV(key.Bytes(), iv,
			msg.GetTimestamp())
	if err != nil {
		panic(err)
	}
	msg.SetTimestamp(encryptedTimestamp)

	// E2E encrypt the msg
	encPayload, err := e2e.Encrypt(grp, key, msg.Contents.Get())
	if err != nil {
		globals.Log.ERROR.Panicf(err.Error())
	}
	msg.Contents.Set(encPayload)

	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key.Bytes())
	msg.SetMAC(MAC)
}

// E2EEncryptUnsafe uses the E2E key to encrypt msg
// to its intended recipient
// It doesn't apply padding to the payload, so it can be unsafe
// if the payload is small
// It also properly populates the associated data
// It modifies the passed msg instead of returning a new one
func E2EEncryptUnsafe(grp *cyclic.Group,
	key *cyclic.Int, keyFP format.Fingerprint,
	msg *format.Message) {
	msg.SetKeyFP(keyFP)

	// Encrypt the timestamp using key
	// Timestamp bytes were previously stored
	// and GO only uses 15 bytes, so use those
	var iv [e2e.AESBlockSize]byte
	copy(iv[:], keyFP[:e2e.AESBlockSize])
	encryptedTimestamp, _ :=
		e2e.EncryptAES256WithIV(key.Bytes(), iv,
			msg.GetTimestamp())
	msg.SetTimestamp(encryptedTimestamp)

	// E2E encrypt the msg
	encPayload := e2e.EncryptUnsafe(grp, key, msg.Contents.Get())
	msg.Contents.Set(encPayload)

	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key.Bytes())
	msg.SetMAC(MAC)
}
