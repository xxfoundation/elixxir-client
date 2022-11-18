////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
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
	rounds.Round, time.Time, group.MessageID, error) {

	if tag == "" {
		tag = defaultServiceTag
	}

	// Get the relevant group
	g, exists := m.GetGroup(groupID)
	if !exists {
		return rounds.Round{}, time.Time{}, group.MessageID{},
			errors.Errorf(newNoGroupErr, groupID)
	}

	// Get the current time stripped of the monotonic clock
	timeNow := netTime.Now().Round(0)

	// Create a cMix message for each group member
	groupMessages, msgId, err := m.newMessages(g, tag, message, timeNow)
	if err != nil {
		return rounds.Round{}, time.Time{}, group.MessageID{},
			errors.Errorf(newCmixMsgErr, g.Name, g.ID, err)
	}

	// Send all the groupMessages
	param := cmix.GetDefaultCMIXParams()
	param.DebugTag = "group.Message"
	rid, _, err := m.getCMix().SendMany(groupMessages, param)
	if err != nil {
		return rounds.Round{}, time.Time{}, group.MessageID{},
			errors.Errorf(sendManyCmixErr, m.getReceptionIdentity().ID, g.Name, g.ID, err)
	}

	jww.INFO.Printf("[GC] Sent message to %d members in group %s at %s.",
		len(groupMessages), groupID, timeNow)
	return rid, timeNow, msgId, nil
}

// newMessages builds a list of messages, one for each group chat member.
func (m *manager) newMessages(g gs.Group, tag string, msg []byte,
	timestamp time.Time) ([]cmix.TargetedCmixMessage, group.MessageID, error) {

	// Create list of cMix messages
	messages := make([]cmix.TargetedCmixMessage, 0, len(g.Members))
	rng := m.getRng().GetStream()
	defer rng.Close()

	// Generate initial internal message
	maxCmixMessageLength := m.getCMix().GetMaxMessageLength()

	// Generate public message to determine what length internal message can be
	pubMsg, err := newPublicMsg(maxCmixMessageLength)
	if err != nil {
		return nil, group.MessageID{}, errors.Errorf(newPublicMsgErr, err)
	}

	// Generate internal message
	intlMsg, err := newInternalMsg(pubMsg.GetPayloadSize())
	if err != nil {
		return nil, group.MessageID{}, errors.Errorf(newInternalMsgErr, err)
	}

	// Return an error if the message is too large to fit in the payload
	if intlMsg.GetPayloadMaxSize() < len(msg) {
		return nil, group.MessageID{}, errors.Errorf(
			messageLenErr, len(msg), intlMsg.GetPayloadMaxSize())
	}

	// Generate internal message
	internalMessagePayload := setInternalPayload(intlMsg, timestamp,
		m.getReceptionIdentity().ID, msg)

	// Create cMix messages
	for _, member := range g.Members {
		// Do not send to the sender
		if m.getReceptionIdentity().ID.Cmp(member.ID) {
			continue
		}

		// Add cMix message to list
		cMixMsg, err := newCmixMsg(g, tag, timestamp, member, rng, maxCmixMessageLength,
			internalMessagePayload)
		if err != nil {
			return nil, group.MessageID{}, err
		}
		messages = append(messages, cMixMsg)
	}

	return messages, group.NewMessageID(g.ID, internalMessagePayload), nil
}

// newCmixMsg generates a new cmix.TargetedCmixMessage for the given group
// member
func newCmixMsg(g gs.Group, tag string, timestamp time.Time,
	mem group.Member, rng io.Reader, maxCmixMessageSize int,
	internalMessagePayload []byte) (
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

	// Generate public message
	pubMsg, err := newPublicMsg(maxCmixMessageSize)
	if err != nil {
		return cmixMsg, err
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

	// Encrypt internal message
	encryptedPayload := group.Encrypt(key, cmixMsg.Fingerprint,
		internalMessagePayload)

	// Generate public message
	cmixMsg.Payload = setPublicPayload(pubMsg, salt, encryptedPayload)

	// Generate MAC
	cmixMsg.Mac = group.NewMAC(key, encryptedPayload, g.DhKeys[*mem.ID])

	return cmixMsg, nil
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
