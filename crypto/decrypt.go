////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/e2e"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Decrypt decrypts messages
func Decrypt(nodeKey *cyclic.Int, g *cyclic.Group,
	cmixMsg *pb.CmixMessage) (
	*format.Message, error) {

	// Receive and decrypt a message
	payload := cyclic.NewIntFromBytes(cmixMsg.Payload)
	associatedData := cyclic.NewIntFromBytes(cmixMsg.AssociatedData)

	// perform the decryption
	g.Mul(payload, nodeKey, payload)
	g.Mul(associatedData, nodeKey, associatedData)

	// unpack the message from a MessageBytes
	var message format.Message
	payloadSerial := payload.LeftpadBytes(uint64(format.TOTAL_LEN))
	ADSerial := associatedData.LeftpadBytes(uint64(format.TOTAL_LEN))
	message.Payload = format.DeserializePayload(payloadSerial)
	message.AssociatedData = format.DeserializeAssociatedData(ADSerial)

	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256(cyclic.NewIntFromBytes(message.
		GetKeyFingerprint()), message.GetTimestamp())
	if err != nil {
		jww.ERROR.Panicf(err.Error())
	}
	// TODO deserialize this somewhere along the line and provide methods
	// to mobile developers on the bindings to interact with the timestamps
	message.SetTimestamp(decryptedTimestamp)

	// TODO Should salt be []byte instead of cyclic.Int?
	// TODO Should the method return []byte instead of cyclic.Int?
	// That might give better results in the case that the key happens to be
	// not the correct length in bytes. Unlikely, but possible.
	clientKey := e2e.Keygen(g, nil, nil)
	// Assuming that result of e2e.Keygen() will be nil for non-e2e messages
	if clientKey != nil {
		// Decrypt e2e
		decryptedPayload, err := e2e.Decrypt(*g, clientKey, payloadSerial)
		if err != nil {
			return nil, errors.New(err.Error() +
				"Failed to decrypt e2e message despite non" +
				"-nil client key result")
		}
		// Check MAC with inner message
		if hash.VerifyHMAC(decryptedPayload, message.GetMAC(),
			clientKey.LeftpadBytes(uint64(format.TOTAL_LEN))) {
			return &message, nil
		} else {
			return nil, errors.New("HMAC failed for end-to-end message")
		}
	} else {
		// Check MAC for non-e2e
		if hash.VerifyHMAC(payloadSerial, message.GetMAC(), message.GetKeyFingerprint()) {
			return &message, nil
		} else {
			return nil, errors.New("HMAC failed for plaintext message")
		}
	}
}
