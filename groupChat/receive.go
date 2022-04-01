///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Error messages.
const (
	newDecryptKeyErr        = "failed to generate key for decrypting group payload: %+v"
	unmarshalInternalMsgErr = "failed to unmarshal group internal message: %+v"
	unmarshalSenderIdErr    = "failed to unmarshal sender ID: %+v"
	unmarshalPublicMsgErr   = "failed to unmarshal group cMix message contents: %+v"
	findGroupKeyFpErr       = "no group with key fingerprint %s"
	genCryptKeyMacErr       = "failed to generate encryption key for group " +
		"cMix message because MAC verification failed (epoch %d could be off)"
)

// Adheres to network.Manager interface for reception processing
type receptionProcessor struct {
	m *Manager
	g gs.Group
}

// Process incoming group chat messages
func (p *receptionProcessor) Process(message format.Message, receptionID receptionID.EphemeralIdentity, round historical.Round) {
	jww.TRACE.Print("Group message reception received cMix message.")

	// Attempt to read the message
	roundTimeStamp := round.Timestamps[states.QUEUED]
	msgID, timestamp, senderID, msg, err := decryptMessage(p.g, message, roundTimeStamp)
	if err != nil {
		jww.WARN.Printf("Group message reception failed to read "+
			"cMix message: %+v", err)
		return
	}

	jww.DEBUG.Printf("Received group message with ID %s from sender "+
		"%s in group %s with ID %s at %s.", msgID, senderID, p.g.Name,
		p.g.ID, timestamp)

	// If the message was read correctly, send it to the callback
	go p.m.receiveFunc(MessageReceive{
		GroupID:        p.g.ID,
		ID:             msgID,
		Payload:        msg,
		SenderID:       senderID,
		RecipientID:    receptionID.Source,
		EphemeralID:    receptionID.EphId,
		Timestamp:      timestamp,
		RoundID:        round.ID,
		RoundTimestamp: roundTimeStamp,
	})
}

// decryptMessage decrypts the group message payload and returns its message ID,
// timestamp, sender ID, and message contents.
func decryptMessage(g gs.Group, cMixMsg format.Message, roundTimestamp time.Time) (
	group.MessageID, time.Time, *id.ID, []byte, error) {

	// Unmarshal cMix message contents to get public message format
	pubMsg, err := unmarshalPublicMsg(cMixMsg.GetContents())
	if err != nil {
		return group.MessageID{}, time.Time{}, nil, nil,
			errors.Errorf(unmarshalPublicMsgErr, err)
	}

	key, err := getCryptKey(g.Key, pubMsg.GetSalt(), cMixMsg.GetMac(),
		pubMsg.GetPayload(), g.DhKeys, roundTimestamp)
	if err != nil {
		return group.MessageID{}, time.Time{}, nil, nil, err
	}

	// Decrypt internal message
	decryptedPayload := group.Decrypt(key, cMixMsg.GetKeyFP(),
		pubMsg.GetPayload())

	// Unmarshal internal message
	intlMsg, err := unmarshalInternalMsg(decryptedPayload)
	if err != nil {
		return group.MessageID{}, time.Time{}, nil, nil,
			errors.Errorf(unmarshalInternalMsgErr, err)
	}

	// Unmarshal sender ID
	senderID, err := intlMsg.GetSenderID()
	if err != nil {
		return group.MessageID{}, time.Time{}, nil, nil,
			errors.Errorf(unmarshalSenderIdErr, err)
	}

	messageID := group.NewMessageID(g.ID, intlMsg.Marshal())

	return messageID, intlMsg.GetTimestamp(), senderID, intlMsg.GetPayload(), nil
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
