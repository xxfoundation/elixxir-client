////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"errors"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// CMIX Decrypt performs the decryption
// of a message from a team of nodes
// It returns a new message
func CMIXDecrypt(session user.Session,
	msg *pb.CmixMessage) *format.Message {
	salt := msg.Salt
	nodeKeys := session.GetKeys()
	baseKeys := make([]*cyclic.Int, len(nodeKeys))
	for i, key := range nodeKeys {
		baseKeys[i] = key.ReceptionKey
		//TODO: Add KMAC verification here
	}

	newMsg := format.NewMessage()
	newMsg.Payload = format.DeserializePayload(
		msg.MessagePayload)
	newMsg.AssociatedData = format.DeserializeAssociatedData(
		msg.AssociatedData)

	return cmix.ClientEncryptDecrypt(false, session.GetGroup(), newMsg, salt, baseKeys)
}

// E2EDecrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// or in case of a decryption error (related to padding)
// If it succeeds, it modifies the passed message
func E2EDecrypt(grp *cyclic.Group, key *cyclic.Int,
	msg *format.Message) error {
	// First thing to do is check MAC
	if !hash.VerifyHMAC(msg.SerializePayload(),
		msg.GetMAC(), key.Bytes()) {
		return errors.New("HMAC verification failed for E2E message")
	}
	var iv [e2e.AESBlockSize]byte
	fp := msg.GetKeyFingerprint()
	copy(iv[:], fp[:e2e.AESBlockSize])
	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256WithIV(
		key.Bytes(), iv, msg.GetTimestamp())
	if err != nil {
		return errors.New("Timestamp decryption failed for E2E message: " + err.Error())
	}
	// TODO deserialize this somewhere along the line and provide methods
	// to mobile developers on the bindings to interact with the timestamps
	msg.SetTimestamp(decryptedTimestamp)
	// Decrypt e2e
	decryptedPayload, err := e2e.Decrypt(grp, key, msg.SerializePayload())
	if err != nil {
		return errors.New("Failed to decrypt E2E message: " + err.Error())
	}
	msg.SetSplitPayload(decryptedPayload)
	return nil
}

// E2EDecryptUnsafe uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// It doesn't expect the payload to be padded
// If it succeeds, it modifies the passed message
func E2EDecryptUnsafe(grp *cyclic.Group, key *cyclic.Int,
	msg *format.Message) error {
	// First thing to do is check MAC
	if !hash.VerifyHMAC(msg.SerializePayload(),
		msg.GetMAC(), key.Bytes()) {
		return errors.New("HMAC verification failed for E2E message")
	}
	var iv [e2e.AESBlockSize]byte
	fp := msg.GetKeyFingerprint()
	copy(iv[:], fp[:e2e.AESBlockSize])
	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256WithIV(
		key.Bytes(), iv, msg.GetTimestamp())
	if err != nil {
		return errors.New("Timestamp decryption failed for E2E message: " + err.Error())
	}
	// TODO deserialize this somewhere along the line and provide methods
	// to mobile developers on the bindings to interact with the timestamps
	msg.SetTimestamp(decryptedTimestamp)
	// Decrypt e2e
	decryptedPayload := e2e.DecryptUnsafe(grp, key, msg.SerializePayload())
	msg.SetSplitPayload(decryptedPayload)
	return nil
}
