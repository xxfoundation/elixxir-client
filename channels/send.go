////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

const (
	cmixChannelTextVersion     = 0
	cmixChannelReactionVersion = 0
	cmixChannelDeleteVersion   = 0
	cmixChannelPinVersion      = 0

	// SendMessageTag is the base tag used when generating a debug tag for
	// sending a message.
	SendMessageTag = "ChMessage"

	// SendReplyTag is the base tag used when generating a debug tag for
	// sending a reply.
	SendReplyTag = "ChReply"

	// SendReactionTag is the base tag used when generating a debug tag for
	// sending a reaction.
	SendReactionTag = "ChReaction"

	// SendDeleteTag is the base tag used when generating a debug tag for a
	// delete message.
	SendDeleteTag = "ChDelete"

	// SendPinnedTag is the base tag used when generating a debug tag for a
	// pinned message.
	SendPinnedTag = "ChPinned"

	// SendMuteTag is the base tag used when generating a debug tag for a mute
	// message.
	SendMuteTag = "ChMute"
)

// The size of the nonce used in the message ID.
const messageNonceSize = 4

// Prints current time without the monotonic clock (m=) for easier reading
func dateNow() string { return netTime.Now().Round(0).String() }
func timeNow() string { return netTime.Now().Format("15:04:05.9999999") }

////////////////////////////////////////////////////////////////////////////////
// Normal Sending                                                             //
////////////////////////////////////////////////////////////////////////////////

// SendGeneric is used to send a raw message over a channel. In general, it
// should be wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this will
// return an error. Due to the underlying encoding using compression, it is not
// possible to define the largest payload that can be sent, but it will always
// be possible to send a payload of 802 bytes at minimum.
//
// The meaning of validUntil depends on the use case.
//
// Set tracked to true if the message should be tracked in the sendTracker,
// which allows messages to be shown locally before they are received on the
// network. In general, all messages that will be displayed to the user
// should be tracked while all actions should not be. More technically, any
// messageType that corresponds to a handler that does not return a unique
// ID (i.e., always returns 0) cannot be tracked, or it will cause errors.
func (m *manager) SendGeneric(channelID *id.ID, messageType MessageType,
	msg []byte, validUntil time.Duration, tracked bool, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	// Reject the send if the user is muted in the channel they are sending to
	if m.events.mutedUsers.isMuted(channelID, m.me.PubKey) {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("user muted in channel %s", channelID)
	}

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	log := fmt.Sprintf(
		"[CH] [%s] Sending to channel %s message type %s at %s. ",
		params.DebugTag, channelID, messageType, dateNow())
	var printErr bool
	defer func() {
		if printErr {
			jww.ERROR.Printf(log)
		} else {
			jww.INFO.Print(log)
		}
	}()

	// Find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Failed to get channel: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	nickname, _ := m.GetNickname(channelID)
	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       nickname,
		Nonce:          make([]byte, messageNonceSize),
		LocalTimestamp: netTime.Now().UnixNano(),
	}

	// Generate random nonce to be used for message ID generation. This makes it
	// so two identical messages sent on the same round have different message
	// IDs.
	rng := m.rng.GetStream()
	n, err := rng.Read(chMsg.Nonce)
	rng.Close()
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Failed to generate nonce: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		printErr = true
		log += fmt.Sprintf(
			"ERROR Got %d bytes for %d-byte nonce", n, messageNonceSize)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf(
				"generated %d bytes for %d-byte nonce", n, messageNonceSize)
	}

	// Note: we are not checking if message is too long before trying to find a
	// round

	// Build the function pointer that will build the message
	var messageID cryptoChannel.MessageID
	usrMsg := &UserMessage{ECCPublicKey: m.me.PubKey}
	assemble := func(rid id.Round) ([]byte, error) {
		// Build the message
		chMsg.RoundID = uint64(rid)

		// Serialize the message
		chMsgSerial, err2 := proto.Marshal(chMsg)
		if err2 != nil {
			return nil, err2
		}

		// Make the messageID
		messageID = cryptoChannel.MakeMessageID(chMsgSerial, channelID)

		// Sign the message
		messageSig := ed25519.Sign(*m.me.Privkey, chMsgSerial)

		usrMsg.Message = chMsgSerial
		usrMsg.Signature = messageSig

		// Serialize the user message
		usrMsgSerial, err2 := proto.Marshal(usrMsg)
		if err2 != nil {
			return nil, err2
		}

		return usrMsgSerial, nil
	}

	var uuid uint64
	if tracked {
		log += fmt.Sprintf("Pending send at %s. ", timeNow())
		uuid, err = m.st.denotePendingSend(channelID, &userMessageInternal{
			userMessage:    usrMsg,
			channelMessage: chMsg,
			messageID:      messageID,
		})
		if err != nil {
			printErr = true
			log += fmt.Sprintf(
				"ERROR Pending send failed at %s: %s", timeNow(), err)
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	} else {
		log += "Message not being tracked; skipping pending send. "
	}

	log += fmt.Sprintf("Broadcasting message at %s. ", timeNow())
	r, ephID, err := ch.broadcast.BroadcastWithAssembler(assemble, params)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Broadcast failed at %s: %s. ", timeNow(), err)

		if errDenote := m.st.failedSend(uuid); errDenote != nil {
			log += fmt.Sprintf("Failed to denote failed broadcast: %s", err)
		}
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	log += fmt.Sprintf(
		"Broadcast succeeded at %s on round %d, success!", timeNow(), r.ID)

	if tracked {
		err = m.st.send(uuid, messageID, r)
		if err != nil {
			printErr = true
			log += fmt.Sprintf("ERROR Local broadcast failed: %s", err)
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	}

	return messageID, r, ephID, err
}

// SendMessage is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it is not possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 798 bytes at minimum.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
func (m *manager) SendMessage(channelID *id.ID, msg string,
	validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, []byte(msg), SendMessageTag)
	jww.INFO.Printf("[CH] [%s] SendMessage to channel %s", tag, channelID)

	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: nil,
	}

	params = params.SetDebugTag(tag)

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Text, txtMarshaled, validUntil, false, params)
}

// SendReply is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it is not possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 766 bytes at minimum.
//
// If the message ID that the reply is sent to does not exist, then the other
// side will post the message as a normal message and not as a reply.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
func (m *manager) SendReply(channelID *id.ID, msg string,
	replyTo cryptoChannel.MessageID, validUntil time.Duration,
	params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, []byte(msg), SendReplyTag)
	jww.INFO.Printf(
		"[CH] [%s] SendReply on channel %s to %s", tag, channelID, replyTo)
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: replyTo[:],
	}

	params = params.SetDebugTag(tag)

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Text, txtMarshaled, validUntil, false, params)
}

// SendReaction is used to send a reaction to a message over a channel. The
// reaction must be a single emoji with no other characters, and will be
// rejected otherwise.
//
// Clients will drop the reaction if they do not recognize the reactTo message.
func (m *manager) SendReaction(channelID *id.ID, reaction string,
	reactTo cryptoChannel.MessageID, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, []byte(reaction), SendReactionTag)
	jww.INFO.Printf(
		"[CH] [%s] SendReaction on channel %s to %s", tag, channelID, reactTo)

	if err := ValidateReaction(reaction); err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	react := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	params = params.SetDebugTag(tag)

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Reaction, reactMarshaled, ValidForever, false, params)
}

////////////////////////////////////////////////////////////////////////////////
// Admin Sending                                                              //
////////////////////////////////////////////////////////////////////////////////

// SendAdminGeneric is used to send a raw message over a channel encrypted with
// admin keys, identifying it as sent by the admin. In general, it should be
// wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this will
// return an error. The message must be at most 510 bytes long.
//
// If the user is not an admin of the channel (i.e. does not have a private
// key for the channel saved to storage), then an error is returned.
//
// Set tracked to true if the message should be tracked in the sendTracker,
// which allows messages to be shown locally before they are received on the
// network. In general, all messages that will be displayed to the user should
// be tracked while all actions should not be. More technically, any messageType
// that corresponds to a handler that does not return a unique ID (i.e., always
// returns 0) cannot be tracked, or it will cause errors.
func (m *manager) SendAdminGeneric(channelID *id.ID, messageType MessageType,
	msg []byte, validUntil time.Duration, tracked bool, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	log := fmt.Sprintf(
		"[CH] [%s] Admin sending to channel %s message type %s at %s. ",
		params.DebugTag, channelID, messageType, dateNow())
	var printErr bool
	defer func() {
		if printErr {
			jww.ERROR.Printf(log)
		} else {
			jww.INFO.Print(log)
		}
	}()

	// Find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Failed to get channel: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	// Return an error if the user is not an admin
	log += "Getting channel private key. "
	privKey, err := loadChannelPrivateKey(channelID, m.kv)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Failed to load channel private key: %+v", err)
		if m.kv.Exists(err) {
			jww.WARN.Printf("[CH] Private key for channel ID %s found in "+
				"storage, but an error was encountered while accessing it: %+v",
				channelID, err)
		}
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			NotAnAdminErr
	}

	var messageID cryptoChannel.MessageID
	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       AdminUsername,
		Nonce:          make([]byte, messageNonceSize),
		LocalTimestamp: netTime.Now().UnixNano(),
	}

	// Generate random nonce to be used for message ID generation. This makes it
	// so two identical messages sent on the same round have different message
	// IDs
	rng := m.rng.GetStream()
	n, err := rng.Read(chMsg.Nonce)
	rng.Close()
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Failed to generate nonce: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		printErr = true
		log += fmt.Sprintf(
			"ERROR Got %d bytes for %d-byte nonce", n, messageNonceSize)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf(
				"generated %d bytes for %d-byte nonce", n, messageNonceSize)
	}

	// Note: we are not checking if message is too long before trying to
	// find a round

	// Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {
		// Build the message
		chMsg.RoundID = uint64(rid)

		// Serialize the message
		chMsgSerial, err2 := proto.Marshal(chMsg)
		if err2 != nil {
			return nil, err2
		}

		messageID = cryptoChannel.MakeMessageID(chMsgSerial, channelID)

		// Check if the message is too long
		if len(chMsgSerial) > ch.broadcast.MaxRSAToPublicPayloadSize() {
			return nil, MessageTooLongErr
		}

		return chMsgSerial, nil
	}

	var uuid uint64
	if tracked {
		log += fmt.Sprintf("Pending send at %s. ", timeNow())
		uuid, err = m.st.denotePendingAdminSend(channelID, chMsg)
		if err != nil {
			printErr = true
			log += fmt.Sprintf(
				"ERROR Pending send failed at %s: %s", timeNow(), err)
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	} else {
		log += "Message not being tracked; skipping pending send. "
	}

	log += fmt.Sprintf("Broadcasting message at %s. ", timeNow())
	r, ephID, err := ch.broadcast.BroadcastRSAToPublicWithAssembler(privKey,
		assemble, params)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Broadcast failed at %s: %s. ", timeNow(), err)
		if errDenote := m.st.failedSend(uuid); errDenote != nil {
			log += fmt.Sprintf("Failed to denote failed broadcast: %s", err)
		}
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	log += fmt.Sprintf(
		"Broadcast succeeded at %s on round %d, success!", timeNow(), r.ID)

	if tracked {
		err = m.st.send(uuid, messageID, r)
		if err != nil {
			printErr = true
			log += fmt.Sprintf("ERROR Local broadcast failed: %s", err)
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	}

	return messageID, r, ephID, nil
}

// DeleteMessage deletes the targeted message from user's view. Users may delete
// their own messages but only the channel admin can delete other user's
// messages.
//
// If undoAction is true, then the targeted message is un-deleted.
//
// Clients will drop the deletion if they do not recognize the target message.
func (m *manager) DeleteMessage(channelID *id.ID,
	targetMessage cryptoChannel.MessageID, undoAction bool,
	params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, targetMessage.Bytes(), SendDeleteTag)
	jww.INFO.Printf("[CH] [%s] DeleteMessage in channel %s message %s",
		tag, channelID, targetMessage)

	// Load private key from storage. If it does not exist, then check if the
	// user is the sender of the message to delete.
	isChannelAdmin := m.IsChannelAdmin(channelID)
	if !isChannelAdmin {
		msg, err := m.events.model.GetMessage(targetMessage)
		if err != nil {
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
				errors.Errorf(
					"failed to find targeted message %s to delete: %+v",
					targetMessage, err)
		}

		if !bytes.Equal(msg.PubKey, m.me.PubKey) {
			return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
				errors.Errorf("can only delete message you are sender of " +
					"or if you are the channel admin.")
		}
	}

	deleteMessage := &CMIXChannelDelete{
		Version:    cmixChannelDeleteVersion,
		MessageID:  targetMessage.Bytes(),
		UndoAction: undoAction,
	}

	params = params.SetDebugTag(tag)

	deleteMarshaled, err := proto.Marshal(deleteMessage)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	if isChannelAdmin {
		return m.SendAdminGeneric(
			channelID, Delete, deleteMarshaled, ValidForever, false, params)
	} else {
		return m.SendGeneric(
			channelID, Delete, deleteMarshaled, ValidForever, true, params)
	}
}

// PinMessage pins the target message to the top of a channel view for all
// users in the specified channel. Only the channel admin can pin user messages.
//
// If undoAction is true, then the targeted message is unpinned.
//
// Clients will drop the pin if they do not recognize the target message.
func (m *manager) PinMessage(channelID *id.ID,
	targetMessage cryptoChannel.MessageID, undoAction bool,
	params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, targetMessage.Bytes(), SendDeleteTag)
	jww.INFO.Printf("[CH] [%s] PinMessage in channel %s message %s",
		tag, channelID, targetMessage)

	pinnedMessage := &CMIXChannelPinned{
		Version:    cmixChannelPinVersion,
		MessageID:  targetMessage.Bytes(),
		UndoAction: undoAction,
	}

	params = params.SetDebugTag(tag)

	pinnedMarshaled, err := proto.Marshal(pinnedMessage)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendAdminGeneric(
		channelID, Pinned, pinnedMarshaled, ValidForever, false, params)
}

// MuteUser is used to mute a user in a channel. Muting a user will cause all
// future messages from the user being hidden from view. Muted users are also
// unable to send messages. Only the channel admin can mute a user.
//
// If undoAction is true, then the targeted user will be unmuted.
func (m *manager) MuteUser(channelID *id.ID, mutedUser ed25519.PublicKey,
	undoAction bool, params cmix.CMIXParams) (cryptoChannel.MessageID,
	rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, mutedUser, SendMuteTag)
	jww.INFO.Printf(
		"[CH] [%s] MuteUser in channel %s mute %x", tag, channelID, mutedUser)

	muteMessage := &CMIXChannelMute{
		Version:    cmixChannelPinVersion,
		PubKey:     mutedUser,
		UndoAction: undoAction,
	}

	params = params.SetDebugTag(tag)

	mutedMarshaled, err := proto.Marshal(muteMessage)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendAdminGeneric(
		channelID, Mute, mutedMarshaled, ValidForever, false, params)
}

// makeChaDebugTag is a debug helper that creates non-unique msg identifier.
//
// This is set as the debug tag on messages and enables some level of tracing a
// message (if its contents/chan/type are unique).
func makeChaDebugTag(
	channelID *id.ID, id ed25519.PublicKey, msg []byte, baseTag string) string {

	h, _ := blake2b.New256(nil)
	h.Write(channelID[:])
	h.Write(msg)
	h.Write(id)

	tripCode := base64.RawStdEncoding.EncodeToString(h.Sum(nil))[:12]
	return fmt.Sprintf("%s-%s", baseTag, tripCode)
}
