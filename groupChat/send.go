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
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"time"
)

// Error messages.
const (
	newCmixMsgErr     = "failed to generate cMix messages for group chat: %+v"
	sendManyCmixErr   = "failed to send group chat message from member %s to group %s: %+v"
	newCmixErr        = "failed to generate cMix message for member %d with ID %s in group %s: %+v"
	messageLenErr     = "message length %d is greater than maximum message space %d"
	newNoGroupErr     = "failed to create message for group %s that cannot be found"
	newKeyErr         = "failed to generate key for encrypting group payload"
	newPublicMsgErr   = "failed to create new public group message for cMix message: %+v"
	newInternalMsgErr = "failed to create new internal group message for cMix message: %+v"
	saltReadErr       = "failed to generate salt for group message: %+v"
	saltReadLengthErr = "length of generated salt %d != %d required"
)

// Send sends a message to all group members using Client.SendManyCMIX. The
// send fails if the message is too long.
func (m *Manager) Send(groupID *id.ID, message []byte) (id.Round, time.Time, group.MessageID,
	error) {
	// Get the current time stripped of the monotonic clock
	timeNow := netTime.Now().Round(0)

	// Create a cMix message for each group member
	messages, msgID, err := m.createMessages(groupID, message, timeNow)
	if err != nil {
		return 0, time.Time{}, group.MessageID{}, errors.Errorf(newCmixMsgErr, err)
	}

	param := params.GetDefaultCMIX()
	param.IdentityPreimage = groupID[:]
	param.DebugTag = "group.Message"

	rid, _, err := m.net.SendManyCMIX(messages, param)
	if err != nil {
		return 0, time.Time{}, group.MessageID{},
			errors.Errorf(sendManyCmixErr, m.gs.GetUser().ID, groupID, err)
	}

	jww.DEBUG.Printf("Sent message to %d members in group %s at %s.",
		len(messages), groupID, timeNow)

	return rid, timeNow, msgID, nil
}

// createMessages generates a list of cMix messages and a list of corresponding
// recipient IDs.
func (m *Manager) createMessages(groupID *id.ID, msg []byte, timestamp time.Time) (
	[]message.TargetedCmixMessage, group.MessageID, error) {

	//make the message ID
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	_, intlMsg, err := newMessageParts(cmixMsg.ContentsSize())
	if err != nil {
		return nil, group.MessageID{}, errors.WithMessage(err, "Failed to make message parts for message ID")
	}
	messageID := group.NewMessageID(groupID, setInternalPayload(intlMsg, timestamp, m.gs.GetUser().ID, msg))

	g, exists := m.gs.Get(groupID)
	if !exists {
		return []message.TargetedCmixMessage{}, group.MessageID{},
			errors.Errorf(newNoGroupErr, groupID)
	}

	NewMessages, err := m.newMessages(g, msg, timestamp)

	return NewMessages, messageID, err
}

// newMessages is a private function that allows the passing in of a timestamp
// and streamGen instead of a fastRNG.StreamGenerator for easier testing.
func (m *Manager) newMessages(g gs.Group, msg []byte, timestamp time.Time) (
	[]message.TargetedCmixMessage, error) {
	// Create list of cMix messages
	messages := make([]message.TargetedCmixMessage, 0, len(g.Members))

	// Create channels to receive messages and errors on
	type msgInfo struct {
		msg format.Message
		id  *id.ID
	}
	msgChan := make(chan msgInfo, len(g.Members)-1)
	errChan := make(chan error, len(g.Members)-1)

	// Create cMix messages in parallel
	for i, member := range g.Members {
		// Do not send to the sender
		if m.gs.GetUser().ID.Cmp(member.ID) {
			continue
		}

		// Start thread to build cMix message
		go func(member group.Member, i int) {
			// Create new stream
			rng := m.rng.GetStream()
			defer rng.Close()

			// Add cMix message to list
			cMixMsg, err := m.newCmixMsg(g, msg, timestamp, member, rng)
			if err != nil {
				errChan <- errors.Errorf(newCmixErr, i, member.ID, g.ID, err)
			}
			msgChan <- msgInfo{cMixMsg, member.ID}

		}(member, i)
	}

	// Wait for messages or errors
	for len(messages) < len(g.Members)-1 {
		select {
		case err := <-errChan:
			// Return on the first error that occurs
			return nil, err
		case info := <-msgChan:
			messages = append(messages, message.TargetedCmixMessage{
				Recipient: info.id,
				Message:   info.msg,
			})
		}
	}

	return messages, nil
}

// newCmixMsg generates a new cMix message to be sent to a group member.
func (m *Manager) newCmixMsg(g gs.Group, msg []byte, timestamp time.Time,
	mem group.Member, rng io.Reader) (format.Message, error) {

	// Create three message layers
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	pubMsg, intlMsg, err := newMessageParts(cmixMsg.ContentsSize())
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
	keyFp := group.NewKeyFingerprint(g.Key, salt, mem.ID)

	// Generate key
	key, err := group.NewKdfKey(g.Key, group.ComputeEpoch(timestamp), salt)
	if err != nil {
		return cmixMsg, errors.WithMessage(err, newKeyErr)
	}

	// Generate internal message
	payload := setInternalPayload(intlMsg, timestamp, m.gs.GetUser().ID, msg)

	// Encrypt internal message
	encryptedPayload := group.Encrypt(key, keyFp, payload)

	// Generate public message
	publicPayload := setPublicPayload(pubMsg, salt, encryptedPayload)

	// Generate MAC
	mac := group.NewMAC(key, encryptedPayload, g.DhKeys[*mem.ID])

	// Construct cMix message
	cmixMsg.SetContents(publicPayload)
	cmixMsg.SetKeyFP(keyFp)
	cmixMsg.SetMac(mac)

	return cmixMsg, nil
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
