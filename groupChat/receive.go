///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
)

// Error messages.
const (
	newDecryptKeyErr        = "failed to generate key for decrypting group payload: %+v"
	unmarshalInternalMsgErr = "failed to unmarshal group internal message: %+v"
	unmarshalSenderIdErr    = "failed to unmarshal sender ID: %+v"
	unmarshalPublicMsgErr   = "failed to unmarshal group cMix message contents: %+v"
	genCryptKeyMacErr       = "failed to generate encryption key for group " +
		"cMix message because MAC verification failed (epoch %d could be off)"
)

// Adheres to cmix.Manager interface for reception processing
type receptionProcessor struct {
	m *Manager
	g gs.Group
}

// Process incoming group chat messages
func (p *receptionProcessor) Process(message format.Message, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.TRACE.Print("Group message reception received cMix message.")
	// Attempt to read the message
	roundTimestamp := round.Timestamps[states.QUEUED]

	// Unmarshal cMix message contents to get public message format
	pubMsg, err := unmarshalPublicMsg(message.GetContents())
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal: %+v", errors.Errorf(unmarshalPublicMsgErr, err))
	}

	// Obtain the cryptKey for the public message
	key, err := getCryptKey(p.g.Key, pubMsg.GetSalt(), message.GetMac(),
		pubMsg.GetPayload(), p.g.DhKeys, roundTimestamp)
	if err != nil {
		jww.WARN.Printf("Unable to getCryptKey: %+v", err)
		return
	}

	// Decrypt the message payload using the cryptKey
	result, err := decryptMessage(p.g, message.GetKeyFP(), key, pubMsg.GetPayload())
	if err != nil {
		jww.WARN.Printf("Group message reception failed to read "+
			"cMix message: %+v", err)
		return
	}
	// Populate remaining fields from the top level
	result.GroupID = p.g.ID
	result.RecipientID = receptionID.Source
	result.EphemeralID = receptionID.EphId
	result.RoundID = round.ID
	result.RoundTimestamp = roundTimestamp

	jww.DEBUG.Printf("Received group message with ID %s from sender "+
		"%s in group %s with ID %s at %s.", result.ID, result.SenderID, p.g.Name,
		p.g.ID, result.Timestamp)

	// If the message was read correctly, send it to the callback
	p.m.receiveFunc(result)
}

func (p *receptionProcessor) String() string {
	return fmt.Sprintf("GroupChatReception(%s)", p.m.receptionId)
}

// decryptMessage decrypts the group message payload and returns its message ID,
// timestamp, sender ID, and message contents.
func decryptMessage(g gs.Group, fingerprint format.Fingerprint, key group.CryptKey, payload []byte) (
	MessageReceive, error) {

	// Decrypt internal message
	decryptedPayload := group.Decrypt(key, fingerprint, payload)

	// Unmarshal internal message
	intlMsg, err := unmarshalInternalMsg(decryptedPayload)
	if err != nil {
		return MessageReceive{}, errors.Errorf(unmarshalInternalMsgErr, err)
	}

	// Unmarshal sender ID
	senderID, err := intlMsg.GetSenderID()
	if err != nil {
		return MessageReceive{}, errors.Errorf(unmarshalSenderIdErr, err)
	}

	return MessageReceive{
		ID:        group.NewMessageID(g.ID, intlMsg.Marshal()),
		Payload:   intlMsg.GetPayload(),
		SenderID:  senderID,
		Timestamp: intlMsg.GetTimestamp(),
	}, nil
}

// getCryptKey generates the decryption key for a group internal message. The
// key is generated using the group key, an epoch, and a salt. The epoch is
// based off the round timestamp. So, to avoid missing the correct epoch, the
// current, past, and next epochs are checked until one of them produces a key
// that matches the message's MAC. The DH key is also unknown, so each member's
// DH key is tried until there is a match.
func getCryptKey(key group.Key, salt [group.SaltLen]byte, mac, payload []byte,
	dhKeys gs.DhKeyList, roundTimestamp time.Time) (group.CryptKey, error) {

	// Compute the current epoch
	epoch := group.ComputeEpoch(roundTimestamp)
	for _, dhKey := range dhKeys {

		// Create a key with the correct epoch
		for _, epoch := range []uint32{epoch, epoch - 1, epoch + 1} {
			// Generate key
			cryptKey, err := group.NewKdfKey(key, epoch, salt)
			if err != nil {
				return group.CryptKey{}, errors.Errorf(newDecryptKeyErr, err)
			}

			// Return the key if the MAC matches
			if group.CheckMAC(mac, cryptKey, payload, dhKey) {
				return cryptKey, nil
			}
		}
	}

	// Return an error if none of the epochs worked
	return group.CryptKey{}, errors.Errorf(genCryptKeyMacErr, epoch)
}
