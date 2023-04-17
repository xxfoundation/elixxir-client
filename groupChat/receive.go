////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"gitlab.com/xx_network/primitives/netTime"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
)

// Error messages.
const (
	newDecryptKeyErr        = "failed to generate key for decrypting group payload: %+v"
	unmarshalInternalMsgErr = "failed to unmarshal group internal message: %+v"
	unmarshalSenderIdErr    = "failed to unmarshal sender ID: %+v"
	unmarshalPublicMsgErr   = "[GC] Failed to unmarshal group cMix message contents from %d (%s) on round %d: %+v"
	getDecryptionKeyErr     = "[GC] Unable to get decryption key: %+v"
	decryptMsgErr           = "[GC] Failed to decrypt group message: %+v"
	genCryptKeyMacErr       = "failed to generate encryption key for group " +
		"cMix message because MAC verification failed (epoch %d could be off)"
)

// Adheres to message.Processor interface for reception processing.
type receptionProcessor struct {
	m *manager
	g gs.Group
	p Processor
}

// Process incoming group chat messages.
func (p *receptionProcessor) Process(message format.Message, _ []string, _ []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.TRACE.Printf("[GC] Received group message from %d (%s) on round %d.",
		receptionID.EphId.Int64(), receptionID.Source, round.ID)

	// Unmarshal cMix message contents to get public message format
	pubMsg, err := unmarshalPublicMsg(message.GetContents())
	if err != nil {
		jww.ERROR.Printf(unmarshalPublicMsgErr, receptionID.EphId.Int64(),
			receptionID.Source, round.ID, err)
		return
	}

	// Obtain the decryption key for the public message
	// We use PRECOMPUTING here because all Rounds have that timestamp available to them
	// QUEUED can be missing sometimes and cause a lot of hidden problems further down the line
	key, err := getCryptKey(p.g.Key, pubMsg.GetSalt(), message.GetMac(),
		pubMsg.GetPayload(), p.g.DhKeys, round.Timestamps[states.PRECOMPUTING])
	if err != nil {
		jww.ERROR.Printf(getDecryptionKeyErr, err)
		return
	}

	// Decrypt the message payload using the cryptKey
	result, err := decryptMessage(
		p.g, message.GetKeyFP(), key, pubMsg.GetPayload())
	if err != nil {
		jww.ERROR.Printf(decryptMsgErr, err)
		return
	}

	// Populate remaining fields from the top level
	result.GroupID = p.g.ID

	jww.DEBUG.Printf("[GC] Received group message with ID %s from sender "+
		"%s in group %q with ID %s at %s.", result.ID, result.SenderID,
		p.g.Name, p.g.ID, result.Timestamp)

	// Send the decrypted message and original message to the processor
	p.p.Process(result, message, nil, nil, receptionID, round)
}

func (p *receptionProcessor) String() string {
	if p.p == nil {
		return fmt.Sprintf("GroupChatReception(%s)",
			p.m.getReceptionIdentity().ID)
	}
	return fmt.Sprintf("GroupChatReception(%s)-%s",
		p.m.getReceptionIdentity().ID, p.p)
}

// decryptMessage decrypts the group message payload and returns its message ID,
// timestamp, sender ID, and message contents.
func decryptMessage(g gs.Group, fingerprint format.Fingerprint,
	key group.CryptKey, payload []byte) (MessageReceive, error) {

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

	// If given zero time, try to guesstimate roundTimestamp as right now
	if roundTimestamp.Equal(time.Unix(0, 0)) {
		jww.ERROR.Printf("getCryptKey missing roundTimestamp")
		roundTimestamp = netTime.Now()
	}

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
