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
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/emoji"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

const (
	/* Versions for various message types */
	cmixChannelTextVersion       = 0
	cmixChannelReactionVersion   = 0
	cmixChannelInvitationVersion = 0
	cmixChannelSilentVersion     = 0
	cmixChannelDeleteVersion     = 0
	cmixChannelPinVersion        = 0

	// SendMessageTag is the base tag used when generating a debug tag for
	// sending a message.
	SendMessageTag = "ChMessage"

	// SendReplyTag is the base tag used when generating a debug tag for
	// sending a reply.
	SendReplyTag = "ChReply"

	// SendReactionTag is the base tag used when generating a debug tag for
	// sending a reaction.
	SendReactionTag = "ChReaction"

	// SendInviteTag is the base tag used when generating a debug tag for
	// sending an invitation.
	SendInviteTag = "ChInvite"

	// SendSilentTag is the base tag used when generating a debug tag for
	// sending a silent message.
	SendSilentTag = "ChSilent"

	// SendDeleteTag is the base tag used when generating a debug tag for a
	// delete message.
	SendDeleteTag = "ChDelete"

	// SendPinnedTag is the base tag used when generating a debug tag for a
	// pinned message.
	SendPinnedTag = "ChPinned"

	// SendMuteTag is the base tag used when generating a debug tag for a mute
	// message.
	SendMuteTag = "ChMute"

	// SendAdminReplayTag is the base tag used when generating a debug tag for
	// an admin replay message.
	SendAdminReplayTag = "ChAdminReplay"

	// The size of the nonce used in the message ID.
	messageNonceSize = 4
)

var emptyChannelID = &id.ID{}

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
//
// Pings are a list of ed25519 public keys that will receive notifications
// for this message. They must be in the channel and have notifications enabled
func (m *manager) SendGeneric(channelID *id.ID, messageType MessageType,
	msg []byte, validUntil time.Duration, tracked bool, params cmix.CMIXParams,
	pings []ed25519.PublicKey) (
	message.ID, rounds.Round, ephemeral.Id, error) {

	if hmac.Equal(channelID.Bytes(), emptyChannelID.Bytes()) {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.New("cannot send to channel id with all 0s")
	}

	// Reject the send if the user is muted in the channel they are sending to
	if m.events.mutedUsers.isMuted(channelID, m.me.PubKey) {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("user muted in channel %s", channelID)
	}

	// Reject the send if the user is muted in the channel they are sending to
	if m.events.mutedUsers.isMuted(channelID, m.me.PubKey) {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	nickname, _ := m.GetNickname(channelID)

	// Retrieve token.
	// Note that this may be nil if DM token have not been enabled,
	// which is OK.
	dmToken := m.getDmToken(channelID)

	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
		Payload:        msg,
		Nickname:       nickname,
		Nonce:          make([]byte, messageNonceSize),
		LocalTimestamp: netTime.Now().UnixNano(),
		DMToken:        dmToken,
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		printErr = true
		log += fmt.Sprintf(
			"ERROR Got %d bytes for %d-byte nonce", n, messageNonceSize)
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf(
				"generated %d bytes for %d-byte nonce", n, messageNonceSize)
	}

	// Note: we are not checking if message is too long before trying to find a
	// round

	// Build the function pointer that will build the message
	var messageID message.ID
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
		messageID = message.
			DeriveChannelMessageID(channelID, chMsg.RoundID, chMsgSerial)

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
			return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	} else {
		log += "Message not being tracked; skipping pending send. "
	}

	log += fmt.Sprintf("Broadcasting message at %s. ", timeNow())
	tags := makeUserPingTags(pings)
	r, ephID, err := ch.broadcast.BroadcastWithAssembler(assemble, tags,
		uint16(messageType), params)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Broadcast failed at %s: %s. ", timeNow(), err)

		if errDenote := m.st.failedSend(uuid); errDenote != nil {
			log += fmt.Sprintf("Failed to denote failed broadcast: %s", err)
		}
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	log += fmt.Sprintf(
		"Broadcast succeeded at %s on round %d, success!", timeNow(), r.ID)

	if tracked {
		err = m.st.send(uuid, messageID, r)
		if err != nil {
			printErr = true
			log += fmt.Sprintf("ERROR Local broadcast failed: %s", err)
			return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
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
//
// Pings are a list of ed25519 public keys that will receive notifications
// for this message. They must be in the channel and have notifications enabled
func (m *manager) SendMessage(channelID *id.ID, msg string,
	validUntil time.Duration, params cmix.CMIXParams, pings []ed25519.PublicKey) (
	message.ID, rounds.Round, ephemeral.Id, error) {
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Text, txtMarshaled, validUntil, true, params, pings)
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
//
// Pings are a list of ed25519 public keys that will receive notifications
// for this message. They must be in the channel and have notifications enabled
func (m *manager) SendReply(channelID *id.ID, msg string,
	replyTo message.ID, validUntil time.Duration,
	params cmix.CMIXParams, pings []ed25519.PublicKey) (
	message.ID, rounds.Round, ephemeral.Id, error) {
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Text, txtMarshaled, validUntil, true, params, pings)
}

// SendReaction is used to send a reaction to a message over a channel. The
// reaction must be a single emoji with no other characters, and will be
// rejected otherwise.
//
// Clients will drop the reaction if they do not recognize the reactTo message.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
//
// Pings are a list of ed25519 public keys that will receive notifications
// for this message. They must be in the channel and have notifications enabled
func (m *manager) SendReaction(channelID *id.ID, reaction string,
	reactTo message.ID, validUntil time.Duration, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, []byte(reaction), SendReactionTag)
	jww.INFO.Printf(
		"[CH] [%s] SendReaction on channel %s to %s", tag, channelID, reactTo)

	if err := emoji.ValidateReaction(reaction); err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	react := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	params = params.SetDebugTag(tag)

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Reaction, reactMarshaled, validUntil, true, params,
		nil)
}

// SendInvite is used to send to a channel (invited) an invitation to another
// channel (invitee).
//
// If the channel ID for the invitee channel is not recognized by the Manager,
// then an error will be returned.
//
// See [Manager.SendGeneric] for details on payload size limitations and
// elaboration of pings.
func (m *manager) SendInvite(channelID *id.ID, msg string, inviteTo *id.ID,
	host string, maxUses int, validUntil time.Duration,
	params cmix.CMIXParams) (message.ID, rounds.Round, ephemeral.Id, error) {

	// Formulate custom tag
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, []byte(msg), SendInviteTag)

	// Modify the params for the custom tag
	params = params.SetDebugTag(tag)

	jww.INFO.Printf(
		"[CH] [%s] SendInvite on to channel %s", tag, channelID)

	// Retrieve channel that will be used for the invitation
	ch, err := m.getChannel(inviteTo)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.WithMessage(err,
				"could form invitation for a channel that has not been joined.")

	}

	// Form link for invitation
	inviteUrl, err := ch.broadcast.Get().InviteURL(host, maxUses)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.WithMessage(err, "could not form URL")
	}

	// Construct message
	invitation := &CMIXChannelInvitation{
		Version:    cmixChannelInvitationVersion,
		Text:       msg,
		InviteLink: inviteUrl,
	}

	// Marshal message
	invitationMarshalled, err := proto.Marshal(invitation)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	// Send invitation
	return m.SendGeneric(channelID, Invitation, invitationMarshalled,
		validUntil, true, params, nil)
}

// SendSilent is used to send to a channel a message with no notifications.
// Its primary purpose is to communicate new nicknames without calling
// SendMessage.
//
// It takes no payload intentionally as the message should be very lightweight.
func (m *manager) SendSilent(channelID *id.ID, validUntil time.Duration,
	params cmix.CMIXParams) (message.ID, rounds.Round, ephemeral.Id, error) {
	// Formulate custom tag
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, nil, SendSilentTag)

	// Modify the params for the custom tag
	params = params.SetDebugTag(tag)

	jww.INFO.Printf(
		"[CH] [%s] SendSilent on channel %s", tag, channelID)

	// Construct message
	silent := &CMIXChannelSilentMessage{
		Version: cmixChannelSilentVersion,
	}

	// Marshal message
	silentMarshalled, err := proto.Marshal(silent)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	// Send silent message
	return m.SendGeneric(channelID, Silent, silentMarshalled,
		validUntil, true, params, nil)
}

// replayAdminMessage is used to rebroadcast an admin message asa a norma user.
func (m *manager) replayAdminMessage(channelID *id.ID, encryptedPayload []byte,
	params cmix.CMIXParams) (message.ID,
	rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, encryptedPayload, SendAdminReplayTag)
	jww.INFO.Printf(
		"[CH] [%s] replayAdminMessage in channel %s", tag, channelID)

	// Set validUntil to 0 since the replay message itself is not registered in
	// the lease system (only the message its contains)
	return m.SendGeneric(
		channelID, AdminReplay, encryptedPayload, 0,
		false, params, nil)
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
	message.ID, rounds.Round, ephemeral.Id, error) {

	if hmac.Equal(channelID.Bytes(), emptyChannelID.Bytes()) {
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.New("cannot send to channel id with all 0s")
	}

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
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, NotAnAdminErr
	}

	var messageID message.ID
	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
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
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		printErr = true
		log += fmt.Sprintf(
			"ERROR Got %d bytes for %d-byte nonce", n, messageNonceSize)
		return message.ID{}, rounds.Round{}, ephemeral.Id{},
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

		messageID = message.
			DeriveChannelMessageID(channelID, chMsg.RoundID, chMsgSerial)

		// Check if the message is too long
		if len(chMsgSerial) > ch.broadcast.MaxRSAToPublicPayloadSize() {
			return nil, MessageTooLongErr
		}

		return chMsgSerial, nil
	}

	log += fmt.Sprintf("Broadcasting message at %s. ", timeNow())
	encryptedPayload, r, ephID, err := ch.broadcast.
		BroadcastRSAToPublicWithAssembler(privKey, assemble, nil,
			uint16(messageType), params)
	if err != nil {
		printErr = true
		log += fmt.Sprintf("ERROR Broadcast failed at %s: %s. ", timeNow(), err)
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	var uuid uint64
	if tracked {
		log += fmt.Sprintf("Denoting send at %s. ", timeNow())
		uuid, err = m.st.denotePendingAdminSend(channelID, chMsg, encryptedPayload)
		if err != nil {
			printErr = true
			log += fmt.Sprintf("ERROR Denoting send failed at %s: %s. ", timeNow(), err)
			return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	} else {
		log += "Message not being tracked; skipping pending send. "
	}

	log += fmt.Sprintf(
		"Broadcast succeeded at %s on round %d, success!", timeNow(), r.ID)

	if tracked {
		err = m.st.send(uuid, messageID, r)
		if err != nil {
			printErr = true
			log += fmt.Sprintf("ERROR Local broadcast failed: %s", err)
			return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
		}
	}

	return messageID, r, ephID, nil
}

// DeleteMessage deletes the targeted message from storage. Users may delete
// their own messages but only the channel admin can delete other user's
// messages. If the user is not an admin of the channel or if they are not the
// sender of the targetMessage, then the error NotAnAdminErr is returned.
//
// Clients will drop the deletion if they do not recognize the target message.
func (m *manager) DeleteMessage(channelID *id.ID,
	targetMessage message.ID, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, targetMessage.Bytes(), SendDeleteTag)
	jww.INFO.Printf("[CH] [%s] Delete message %s in channel %s",
		tag, targetMessage, channelID)

	// Load private key from storage. If it does not exist, then check if the
	// user is the sender of the message to delete.
	isChannelAdmin := m.IsChannelAdmin(channelID)
	if !isChannelAdmin {
		msg, err := m.events.model.GetMessage(targetMessage)
		if err != nil {
			return message.ID{}, rounds.Round{}, ephemeral.Id{},
				errors.Errorf(
					"failed to find targeted message %s to delete: %+v",
					targetMessage, err)
		}

		if !bytes.Equal(msg.PubKey, m.me.PubKey) {
			return message.ID{}, rounds.Round{}, ephemeral.Id{},
				errors.Errorf("can only delete message you are sender of " +
					"or if you are the channel admin.")
		}
	}

	deleteMessage := &CMIXChannelDelete{
		Version:   cmixChannelDeleteVersion,
		MessageID: targetMessage.Bytes(),
	}

	params = params.SetDebugTag(tag)

	deleteMarshaled, err := proto.Marshal(deleteMessage)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	if isChannelAdmin {
		return m.SendAdminGeneric(
			channelID, Delete, deleteMarshaled, ValidForever, false, params)
	} else {
		return m.SendGeneric(
			channelID, Delete, deleteMarshaled, ValidForever, false, params, nil)
	}
}

// PinMessage pins the target message to the top of a channel view for all
// users in the specified channel. Only the channel admin can pin user messages.
//
// If undoAction is true, then the targeted message is unpinned.
//
// Clients will drop the pin if they do not recognize the target message.
func (m *manager) PinMessage(channelID *id.ID,
	targetMessage message.ID, undoAction bool,
	validUntil time.Duration, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, targetMessage.Bytes(), SendDeleteTag)

	if !undoAction {
		jww.INFO.Printf("[CH] [%s] Pin message %s in channel %s for %s",
			tag, targetMessage, channelID, validUntil)
	} else {
		jww.INFO.Printf("[CH] [%s] Unpin message %s in channel %s for %s",
			tag, targetMessage, channelID, validUntil)
	}

	pinnedMessage := &CMIXChannelPinned{
		Version:    cmixChannelPinVersion,
		MessageID:  targetMessage.Bytes(),
		UndoAction: undoAction,
	}
	pinnedMarshaled, err := proto.Marshal(pinnedMessage)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	params = params.SetDebugTag(tag)

	return m.SendAdminGeneric(
		channelID, Pinned, pinnedMarshaled, validUntil, false, params)
}

// MuteUser is used to mute a user in a channel. Muting a user will cause all
// future messages from the user being dropped on reception. Muted users are
// also unable to send messages. Only the channel admin can mute a user; if the
// user is not an admin of the channel, then the error NotAnAdminErr is
// returned.
//
// If undoAction is true, then the targeted user will be unmuted.
func (m *manager) MuteUser(channelID *id.ID, mutedUser ed25519.PublicKey,
	undoAction bool, validUntil time.Duration, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, mutedUser, SendMuteTag)

	if !undoAction {
		jww.INFO.Printf("[CH] [%s] Mute user %x in channel %s for %s",
			tag, mutedUser, channelID, validUntil)
	} else {
		jww.INFO.Printf("[CH] [%s] Unmute user %x in channel %s for %s",
			tag, mutedUser, channelID, validUntil)
	}

	muteMessage := &CMIXChannelMute{
		Version:    cmixChannelPinVersion,
		PubKey:     mutedUser,
		UndoAction: undoAction,
	}

	params = params.SetDebugTag(tag)

	mutedMarshaled, err := proto.Marshal(muteMessage)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendAdminGeneric(
		channelID, Mute, mutedMarshaled, validUntil, false, params)
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

func makeUserPingTags(users []ed25519.PublicKey) []string {
	if users == nil || len(users) == 0 {
		return nil
	}
	s := make([]string, len(users))
	for i := 0; i < len(s); i++ {
		s[i] = makeUserPingTag(users[i])
	}
	return s
}

func makeUserPingTag(user ed25519.PublicKey) string {
	return fmt.Sprintf("%x-usrping", user)

}
