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
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"time"
)

// Error messages.
const (

	// manager.Send
	newNoGroupErr   = "no group found with ID %s"
	newCmixMsgErr   = "failed to generate cMix messages for group chat %q (%s): %+v"
	sendManyCmixErr = "failed to send group chat message from member %s to group %q (%s): %+v"

	// newCmixMsg
	messageLenErr = "message length %d is greater than maximum payload size %d"
	newKeyErr     = "failed to generate key for encrypting group payload"

	// newMessageParts
	newPublicMsgErr   = "failed to create new public group message for cMix message: %+v"
	newInternalMsgErr = "failed to create new internal group message for cMix message: %+v"

	// newSalt
	saltReadErr       = "failed to generate salt for group message: %+v"
	saltReadLengthErr = "length of generated salt %d != %d required"
)

// Send sends a message to all group members using Cmix.SendMany.
// The send fails if the message is too long.
func (m *manager) Send(groupID *id.ID, tag string, message []byte) (
	id.Round, time.Time, group.MessageID, error) {

	if tag == "" {
		tag = defaultServiceTag
	}

	// Get the relevant group
	g, exists := m.GetGroup(groupID)
	if !exists {
		return 0, time.Time{}, group.MessageID{},
			errors.Errorf(newNoGroupErr, groupID)
	}

	// Get the current time stripped of the monotonic clock
	timeNow := netTime.Now().Round(0)

	// Create a cMix message for each group member
	groupMessages, err := m.newMessages(g, tag, message, timeNow)
	if err != nil {
		return 0, time.Time{}, group.MessageID{},
			errors.Errorf(newCmixMsgErr, g.Name, g.ID, err)
	}

	// Obtain message ID
	msgId, err := getGroupMessageId(
		m.getE2eGroup(), groupID, m.getReceptionIdentity().ID, timeNow, message)
	if err != nil {
		return 0, time.Time{}, group.MessageID{}, err
	}

	// Send all the groupMessages
	param := cmix.GetDefaultCMIXParams()
	param.DebugTag = "group.Message"
	rid, _, err := m.getCMix().SendMany(groupMessages, param)
	if err != nil {
		return 0, time.Time{}, group.MessageID{},
			errors.Errorf(sendManyCmixErr, m.getReceptionIdentity().ID, g.Name, g.ID, err)
	}

	jww.DEBUG.Printf("[GC] Sent message to %d members in group %s at %s.",
		len(groupMessages), groupID, timeNow)
	return rid, timeNow, msgId, nil
}

// newMessages builds a list of messages, one for each group chat member.
func (m *manager) newMessages(g gs.Group, tag string, msg []byte,
	timestamp time.Time) ([]cmix.TargetedCmixMessage, error) {

	// Create list of cMix messages
	messages := make([]cmix.TargetedCmixMessage, 0, len(g.Members))
	rng := m.getRng().GetStream()
	defer rng.Close()

	// Create cMix messages in parallel
	for _, member := range g.Members {
		// Do not send to the sender
		if m.getReceptionIdentity().ID.Cmp(member.ID) {
			continue
		}

		// Add cMix message to list
		cMixMsg, err := newCmixMsg(g, tag, msg, timestamp, member, rng,
			m.getReceptionIdentity().ID, m.getCMix().GetMaxMessageLength())
		if err != nil {
			return nil, err
		}
		messages = append(messages, cMixMsg)
	}

	return messages, nil
}

// newCmixMsg generates a new cmix.TargetedCmixMessage for the given group
// member
func newCmixMsg(g gs.Group, tag string, msg []byte, timestamp time.Time,
	mem group.Member, rng io.Reader, senderId *id.ID, maxCmixMessageSize int) (
	cmix.TargetedCmixMessage, error) {

	// Initialize targeted message
	cmixMsg := cmix.TargetedCmixMessage{
		Recipient: mem.ID,
		Service: message.Service{
			Identifier: g.ID[:],
			Tag:        makeServiceTag(tag),
			Metadata:   g.ID[:],
		},
	}

	// Create three message layers
	pubMsg, intlMsg, err := newMessageParts(maxCmixMessageSize)
	if err != nil {
		return cmixMsg, err
	}

	// Return an error if the message is too large to fit in the payload
	if intlMsg.GetPayloadMaxSize() < len(msg) {
		return cmixMsg, errors.Errorf(
			messageLenErr, len(msg), intlMsg.GetPayloadMaxSize())
	}

	// Generate 256-bit salt
	salt, err := newSalt(rng)
	if err != nil {
		return cmixMsg, err
	}

	// Generate key fingerprint
	cmixMsg.Fingerprint = group.NewKeyFingerprint(g.Key, salt, mem.ID)

	// Generate key
	key, err := group.NewKdfKey(g.Key, group.ComputeEpoch(timestamp), salt)
	if err != nil {
		return cmixMsg, errors.WithMessage(err, newKeyErr)
	}

	// Generate internal message
	payload := setInternalPayload(intlMsg, timestamp, senderId, msg)

	// Encrypt internal message
	encryptedPayload := group.Encrypt(key, cmixMsg.Fingerprint, payload)

	// Generate public message
	cmixMsg.Payload = setPublicPayload(pubMsg, salt, encryptedPayload)

	// Generate MAC
	cmixMsg.Mac = group.NewMAC(key, encryptedPayload, g.DhKeys[*mem.ID])

	return cmixMsg, nil
}

// getGroupMessageId builds the group message ID.
func getGroupMessageId(grp *cyclic.Group, groupId, senderId *id.ID,
	timestamp time.Time, msg []byte) (group.MessageID, error) {
	cmixMsg := format.NewMessage(grp.GetP().ByteLen())
	_, intlMsg, err := newMessageParts(cmixMsg.ContentsSize())
	if err != nil {
		return group.MessageID{}, errors.WithMessage(err,
			"Failed to make message parts for message ID")
	}
	return group.NewMessageID(groupId,
		setInternalPayload(intlMsg, timestamp, senderId, msg)), nil
}

// newMessageParts generates a public payload message and the internal payload
// message. An error is returned if the messages cannot fit in the payloadSize.
func newMessageParts(payloadSize int) (publicMsg, internalMsg, error) {
	pubMsg, err := newPublicMsg(payloadSize)
	if err != nil {
		return pubMsg, internalMsg{}, errors.Errorf(newPublicMsgErr, err)
	}

	intlMsg, err := newInternalMsg(pubMsg.GetPayloadSize())
	if err != nil {
		return pubMsg, intlMsg, errors.Errorf(newInternalMsgErr, err)
	}

	return pubMsg, intlMsg, nil
}

// newSalt generates a new salt of the specified size.
func newSalt(rng io.Reader) ([group.SaltLen]byte, error) {
	var salt [group.SaltLen]byte
	n, err := rng.Read(salt[:])
	if err != nil {
		return salt, errors.Errorf(saltReadErr, err)
	} else if n != group.SaltLen {
		return salt, errors.Errorf(saltReadLengthErr, group.SaltLen, n)
	}

	return salt, nil
}

// setInternalPayload sets the timestamp, sender ID, and message of the
// internalMsg and returns the marshal bytes.
func setInternalPayload(internalMsg internalMsg, timestamp time.Time,
	sender *id.ID, msg []byte) []byte {
	// Set timestamp, sender ID, and message to the internalMsg
	internalMsg.SetTimestamp(timestamp)
	internalMsg.SetSenderID(sender)
	internalMsg.SetPayload(msg)

	// Return the payload marshaled
	return internalMsg.Marshal()
}

// setPublicPayload sets the salt and encrypted payload of the publicMsg and
// returns the marshal bytes.
func setPublicPayload(publicMsg publicMsg, salt [group.SaltLen]byte,
	encryptedPayload []byte) []byte {
	// Set salt and payload
	publicMsg.SetSalt(salt)
	publicMsg.SetPayload(encryptedPayload)

	return publicMsg.Marshal()
}
