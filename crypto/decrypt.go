////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// E2EDecrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// or in case of a decryption error (related to padding)
// If it succeeds, it modifies the passed message
func E2EDecrypt(grp *cyclic.Group, key *cyclic.Int,
	msg *format.Message) error {
	// First thing to do is check MAC
	if !hash.VerifyHMAC(msg.Contents.Get(), msg.GetMAC(), key.Bytes()) {
		return errors.New("HMAC verification failed for E2E message")
	}
	var iv [e2e.AESBlockSize]byte
	fp := msg.GetKeyFP()
	copy(iv[:], fp[:e2e.AESBlockSize])
	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256WithIV(
		key.Bytes(), iv, msg.GetTimestamp())
	if err != nil {
		return errors.New("Timestamp decryption failed for E2E message: " + err.Error())
	}
	// TODO deserialize this somewhere along the line and provide methods
	// to mobile developers on the bindings to interact with the timestamps
	decryptedTimestamp = append(decryptedTimestamp, 0)
	msg.SetTimestamp(decryptedTimestamp)
	// Decrypt e2e
	decryptedPayload, err := e2e.Decrypt(grp, key, msg.Contents.Get())

	if err != nil {
		return errors.New("Failed to decrypt E2E message: " + err.Error())
	}
	msg.Contents.SetRightAligned(decryptedPayload)
	return nil
}

// E2EDecryptUnsafe uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// It doesn't expect the payload to be padded
// If it succeeds, it modifies the passed message
func E2EDecryptUnsafe(grp *cyclic.Group, key *cyclic.Int,
	msg *format.Message) error {
	// First thing to do is check MAC
	if !hash.VerifyHMAC(msg.Contents.Get(), msg.GetMAC(), key.Bytes()) {
		return errors.New("HMAC verification failed for E2E message")
	}
	var iv [e2e.AESBlockSize]byte
	fp := msg.GetKeyFP()
	copy(iv[:], fp[:e2e.AESBlockSize])
	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256WithIV(
		key.Bytes(), iv, msg.GetTimestamp())
	if err != nil {
		return errors.New("Timestamp decryption failed for E2E message: " + err.Error())
	}
	// TODO deserialize this somewhere along the line and provide methods
	// to mobile developers on the bindings to interact with the timestamps
	decryptedTimestamp = append(decryptedTimestamp, 0)
	msg.SetTimestamp(decryptedTimestamp)
	// Decrypt e2e
	decryptedPayload := e2e.DecryptUnsafe(grp, key, msg.Contents.Get())
	msg.Contents.Set(decryptedPayload)
	return nil
}
