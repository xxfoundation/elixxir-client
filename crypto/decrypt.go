////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// Decrypt decrypts messages
func Decrypt(nodeKey *cyclic.Int, grp *cyclic.Group,
	cmixMsg *pb.CmixMessage) (
	*format.Message, error) {

	// Receive and decrypt a message
	payload := grp.NewIntFromBytes(cmixMsg.MessagePayload)

	// perform the CMIX decryption
	grp.Mul(payload, nodeKey, payload)

	// unpack the message from a MessageBytes
	var message format.Message
	payloadSerial := payload.LeftpadBytes(uint64(format.TOTAL_LEN))
	message.AssociatedData = format.DeserializeAssociatedData(cmixMsg.AssociatedData)
	message.Payload = format.DeserializePayload(payloadSerial)

	// TODO Should salt be []byte instead of cyclic.Int?
	// TODO Should the method return []byte instead of cyclic.Int?
	// That might give better results in the case that the key happens to be
	// not the correct length in bytes. Unlikely, but possible.
	clientKey := e2e.Keygen(grp, nil, nil)
	// Assuming that result of e2e.Keygen() will be nil for non-e2e messages
	// TODO BC: why is this assumption valid ?
	if clientKey != nil {
		clientKeyBytes := clientKey.LeftpadBytes(uint64(format.TOTAL_LEN))
		// First thing to do is check MAC
		if !hash.VerifyHMAC(payloadSerial, message.GetMAC(), clientKeyBytes) {
			return nil, errors.New("HMAC failed for end-to-end message")
		}
		var iv [e2e.AESBlockSize]byte
		fp := message.GetKeyFingerprint()
		copy(iv[:], fp[:e2e.AESBlockSize])
		// decrypt the timestamp in the associated data
		decryptedTimestamp, err := e2e.DecryptAES256WithIV(clientKeyBytes, iv, message.GetTimestamp())
		if err != nil {
			jww.ERROR.Panicf(err.Error())
		}
		// TODO deserialize this somewhere along the line and provide methods
		// to mobile developers on the bindings to interact with the timestamps
		message.SetTimestamp(decryptedTimestamp)
		// Decrypt e2e
		decryptedPayload, err := e2e.Decrypt(grp, clientKey, payloadSerial)
		if err != nil {
			return nil, errors.New(err.Error() +
				"Failed to decrypt e2e message despite non" +
				"-nil client key result")
		}
		if message.SetSplitPayload(decryptedPayload) != len(decryptedPayload) {
			jww.ERROR.Panicf("Error setting decrypted payload")
		}
		return &message, nil
	} else {
		// Check MAC for non-e2e
		fp := message.GetKeyFingerprint()
		if hash.VerifyHMAC(payloadSerial, message.GetMAC(), fp[:]) {
			return &message, nil
		} else {
			return nil, errors.New("HMAC failed for plaintext message")
		}
	}
}
