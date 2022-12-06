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
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"strconv"
	"sync"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
)

// AdminUsername defines the displayed username of admin messages, which are
// unique users for every channel defined by the channel's private key.
const AdminUsername = "Admin"

// SentStatus represents the current status of a channel message.
type SentStatus uint8

const (
	// Unsent is the status of a message when it is pending to be sent.
	Unsent SentStatus = iota

	// Sent is the status of a message once the round it is sent on completed.
	Sent

	// Delivered is the status of a message once is has been received.
	Delivered

	// Failed is the status of a message if it failed to send.
	Failed
)

// String returns a human-readable version of [SentStatus], used for debugging
// and logging. This function adheres to the [fmt.Stringer] interface.
func (ss SentStatus) String() string {
	switch ss {
	case Unsent:
		return "unsent"
	case Sent:
		return "sent"
	case Delivered:
		return "delivered"
	case Failed:
		return "failed"
	default:
		return "Invalid SentStatus: " + strconv.Itoa(int(ss))
	}
}

var AdminFakePubKey = ed25519.PublicKey{}

// EventModel is an interface which an external party which uses the channels
// system passed an object which adheres to in order to get events on the
// channel.
type EventModel interface {
	// JoinChannel is called whenever a channel is joined locally.
	JoinChannel(channel *cryptoBroadcast.Channel)

	// LeaveChannel is called whenever a channel is left locally.
	LeaveChannel(channelID *id.ID)

	// ReceiveMessage is called whenever a message is received on a given
	// channel. It may be called multiple times on the same message. It is
	// incumbent on the user of the API to filter such called by message ID.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and round are all nillable and may be updated based
	// upon the UUID at a later date. A time of time.Time{} will be passed for a
	// nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// Message type is included in the call; it will always be Text (1) for this
	// call, but it may be required in downstream databases.
	ReceiveMessage(channelID *id.ID, messageID cryptoChannel.MessageID,
		nickname, text string, pubKey ed25519.PublicKey, codeset uint8,
		timestamp time.Time, lease time.Duration, round rounds.Round,
		mType MessageType, status SentStatus, hidden bool) uint64

	// ReceiveReply is called whenever a message is received that is a reply on
	// a given channel. It may be called multiple times on the same message. It
	// is incumbent on the user of the API to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory, can arrive
	// before the initial message. As a result, it may be important to buffer
	// replies.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and round are all nillable and may be updated based
	// upon the UUID at a later date. A time of time.Time{} will be passed for a
	// nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// Message type is included in the call; it will always be Text (1) for this
	// call, but it may be required in downstream databases.
	ReceiveReply(channelID *id.ID, messageID,
		reactionTo cryptoChannel.MessageID, nickname, text string,
		pubKey ed25519.PublicKey, codeset uint8, timestamp time.Time,
		lease time.Duration, round rounds.Round, mType MessageType,
		status SentStatus, hidden bool) uint64

	// ReceiveReaction is called whenever a reaction to a message is received on
	// a given channel. It may be called multiple times on the same reaction. It
	// is incumbent on the user of the API to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory, can arrive
	// before the initial message. As a result, it may be important to buffer
	// replies.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and round are all nillable and may be updated based
	// upon the UUID at a later date. A time of time.Time{} will be passed for a
	// nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// Message type is included in the call; it will always be Text (1) for this
	// call, but it may be required in downstream databases.
	ReceiveReaction(channelID *id.ID, messageID,
		reactionTo cryptoChannel.MessageID, nickname, reaction string,
		pubKey ed25519.PublicKey, codeset uint8, timestamp time.Time,
		lease time.Duration, round rounds.Round, mType MessageType,
		status SentStatus, hidden bool) uint64

	// UpdateFromUUID is called whenever a message at the UUID is modified.
	//
	// messageID, timestamp, round, pinned, and hidden are all nillable and may
	// be updated based upon the UUID at a later date. If a nil value is passed,
	// then make no update.
	UpdateFromUUID(uuid uint64, messageID *cryptoChannel.MessageID,
		timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
		status *SentStatus)

	// UpdateFromMessageID is called whenever a message with the message ID is
	// modified.
	//
	// The API needs to return the UUID of the modified message that can be
	// referenced at a later time.
	//
	// timestamp, round, pinned, and hidden are all nillable and may be updated
	// based upon the UUID at a later date. If a nil value is passed, then make
	// no update.
	UpdateFromMessageID(messageID cryptoChannel.MessageID, timestamp *time.Time,
		round *rounds.Round, pinned, hidden *bool, status *SentStatus) uint64

	// GetMessage returns the message with the given [channel.MessageID].
	GetMessage(messageID cryptoChannel.MessageID) (ModelMessage, error)
}

// ModelMessage contains a message and all of its information.
type ModelMessage struct {
	UUID            uint64                  `json:"uuid"`
	Nickname        string                  `json:"nickname"`
	MessageID       cryptoChannel.MessageID `json:"messageID"`
	ChannelID       *id.ID                  `json:"channelID"`
	ParentMessageID cryptoChannel.MessageID `json:"parentMessageID"`
	Timestamp       time.Time               `json:"timestamp"`
	Lease           time.Duration           `json:"lease"`
	Status          SentStatus              `json:"status"`
	Hidden          bool                    `json:"hidden"`
	Pinned          bool                    `json:"pinned"`
	Content         []byte                  `json:"content"`
	Type            MessageType             `json:"type"`
	Round           id.Round                `json:"round"`
	PubKey          ed25519.PublicKey       `json:"pubKey"`
	CodesetVersion  uint8                   `json:"codesetVersion"`
}

// MessageTypeReceiveMessage defines handlers for messages of various message
// types. Default ones for Text, Reaction, and AdminText.
//
// A unique UUID must be returned by which the message can be referenced later
// via [EventModel.UpdateFromUUID].
//
// If fromAdmin is true, then the message has been verifies to come from the
// channel admin.
type MessageTypeReceiveMessage func(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, fromAdmin, userMuted bool) uint64

// UpdateFromUuidFunc is a function type for EventModel.UpdateFromUUID so it can
// be mocked for testing where used.
type UpdateFromUuidFunc func(uuid uint64, messageID *cryptoChannel.MessageID,
	timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
	status *SentStatus)

// events is an internal structure that processes events and stores the handlers
// for those events.
type events struct {
	model      EventModel
	registered map[MessageType]MessageTypeReceiveMessage
	leases     *actionLeaseList
	mutedUsers *mutedUserManager
	mux        sync.RWMutex
}

// initEvents initializes the event model and registers default message type
// handlers.
func initEvents(model EventModel, kv *versioned.KV) *events {
	e := &events{
		model: model,
		mux:   sync.RWMutex{},
	}

	// Set up default message types
	// TODO: add space checking (admin, user, muted)
	e.registered = map[MessageType]MessageTypeReceiveMessage{
		Text:      e.receiveTextMessage,
		AdminText: e.receiveTextMessage,
		Reaction:  e.receiveReaction,
		Delete:    e.receiveDelete,
		Pinned:    e.receivePinned,
		Mute:      e.receiveMute,
	}

	// Initialise list of message leases
	var err error
	e.leases, err = newOrLoadActionLeaseList(e.triggerActionEvent, kv)
	if err != nil {
		jww.FATAL.Panicf("Failed to initialise lease list: %+v", err)
	}

	// Initialise list of muted users
	e.mutedUsers, err = newOrLoadMutedUserManager(kv)
	if err != nil {
		jww.FATAL.Panicf("Failed to initialise muted user list: %+v", err)
	}

	return e
}

// RegisterReceiveHandler is used to register handlers for non default message
// types s they can be processed by modules. It is important that such modules
// sync up with the event model implementation.
//
// There can only be one handler per message type, and this will return an error
// on a multiple registration.
func (e *events) RegisterReceiveHandler(
	messageType MessageType, listener MessageTypeReceiveMessage) error {
	e.mux.Lock()
	defer e.mux.Unlock()

	// check if the type is already registered
	if _, exists := e.registered[messageType]; exists {
		return MessageTypeAlreadyRegistered
	}

	// register the message type
	e.registered[messageType] = listener
	jww.INFO.Printf("Registered Listener for Message Type %s", messageType)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Message Triggers                                                           //
////////////////////////////////////////////////////////////////////////////////

type triggerEventFunc func(chID *id.ID, umi *userMessageInternal, ts time.Time,
	receptionID receptionID.EphemeralIdentity, round rounds.Round,
	status SentStatus) (uint64, error)

// triggerEvent is an internal function that is used to trigger message
// reception on a message received from a user (symmetric encryption).
//
// It will call the appropriate MessageTypeReceiveMessage, assuming one exists.
//
// This function adheres to the triggerEventFunc type.
func (e *events) triggerEvent(chID *id.ID, umi *userMessageInternal,
	ts time.Time, _ receptionID.EphemeralIdentity, round rounds.Round,
	status SentStatus) (uint64, error) {
	um := umi.GetUserMessage()
	cm := umi.GetChannelMessage()
	messageType := MessageType(cm.PayloadType)

	// Check if the type is already registered
	e.mux.RLock()
	listener, exists := e.registered[messageType]
	e.mux.RUnlock()
	if !exists {
		err := errors.Errorf("Received message from %x on channel %s in "+
			"round %d that could not be handled due to unregistered message "+
			"type %s; Contents: %v",
			um.ECCPublicKey, chID, round.ID, messageType, cm.Payload)
		jww.WARN.Print(err)
		return 0, err
	}

	// Check if the user is muted on this channel
	isMuted := e.mutedUsers.isMuted(chID, um.ECCPublicKey)

	// Call the listener. This is already in an instanced event; no new thread
	// is needed.
	uuid := listener(chID, umi.GetMessageID(), messageType, cm.Nickname,
		cm.Payload, um.ECCPublicKey, 0, ts, time.Duration(cm.Lease), round,
		status, false, isMuted)
	return uuid, nil
}

type triggerAdminEventFunc func(chID *id.ID, cm *ChannelMessage, ts time.Time,
	messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
	round rounds.Round, status SentStatus) (uint64, error)

// triggerAdminEvent is an internal function that is used to trigger message
// reception on a message received from the admin (asymmetric encryption).
//
// It will call the appropriate MessageTypeReceiveMessage, assuming one exists.
//
// This function adheres to the triggerAdminEventFunc type.
func (e *events) triggerAdminEvent(chID *id.ID, cm *ChannelMessage,
	ts time.Time, messageID cryptoChannel.MessageID,
	_ receptionID.EphemeralIdentity, round rounds.Round, status SentStatus) (
	uint64, error) {
	messageType := MessageType(cm.PayloadType)

	// check if the type is already registered
	e.mux.RLock()
	listener, exists := e.registered[messageType]
	e.mux.RUnlock()
	if !exists {
		err := errors.Errorf("Received Admin message from %s on channel %s in "+
			"round %d that could not be handled due to unregistered message "+
			"type %s; Contents: %v",
			AdminUsername, chID, round.ID, messageType, cm.Payload)
		jww.WARN.Print(err)
		return 0, err
	}

	// Call the listener. This is already in an instanced event; no new thread
	// is needed.
	uuid := listener(chID, messageID, messageType, AdminUsername, cm.Payload,
		AdminFakePubKey, 0, ts, time.Duration(cm.Lease), round, status, true,
		false)
	return uuid, nil
}

type triggerActionEventFunc func(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	payload []byte, timestamp time.Time, lease time.Duration,
	round rounds.Round, status SentStatus, fromAdmin bool) (uint64, error)

// triggerActionEvent is an internal function that is used to trigger an action
// on a message.
//
// It will call the appropriate MessageTypeReceiveMessage, assuming one exists.
//
// This function adheres to the triggerActionEventFunc type.
func (e *events) triggerActionEvent(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	payload []byte, timestamp time.Time, lease time.Duration,
	round rounds.Round, status SentStatus, fromAdmin bool) (uint64, error) {

	// Check if the type is already registered
	e.mux.RLock()
	listener, exists := e.registered[messageType]
	e.mux.RUnlock()
	if !exists {
		err := errors.Errorf("Received action trigger message %s from %s on "+
			"channel %s in round %d that could not be handled due to "+
			"unregistered message type %s; Contents: %v",
			messageID, nickname, channelID, round.ID, messageType, payload)
		jww.WARN.Print(err)
		return 0, err
	}

	// Call the listener. This is already in an instanced event; no new thread
	// is needed.
	return listener(
		channelID, messageID, messageType, nickname, payload, AdminFakePubKey,
		0, timestamp, lease, round, status, fromAdmin, false), nil
}

////////////////////////////////////////////////////////////////////////////////
// Message Handlers                                                           //
////////////////////////////////////////////////////////////////////////////////

// receiveTextMessage is the internal function that handles the reception of
// text messages. It handles both messages and replies and calls the correct
// function on the event model.
//
// If the message has a reply, but it is malformed, it will drop the reply and
// write to the log.
func (e *events) receiveTextMessage(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType,
	nickname string, content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, _, userMuted bool) uint64 {
	txt := &CMIXChannelText{}

	if err := proto.Unmarshal(content, txt); err != nil {
		jww.ERROR.Printf("Failed to text unmarshal message %s from %x on "+
			"channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	if txt.ReplyMessageID != nil {

		if len(txt.ReplyMessageID) == cryptoChannel.MessageIDLen {
			var replyTo cryptoChannel.MessageID
			copy(replyTo[:], txt.ReplyMessageID)
			tag := makeChaDebugTag(channelID, pubKey, content, SendReplyTag)
			jww.INFO.Printf("[%s]Channels - Received reply from %s "+
				"to %s on %s", tag, base64.StdEncoding.EncodeToString(pubKey),
				base64.StdEncoding.EncodeToString(txt.ReplyMessageID),
				channelID)
			return e.model.ReceiveReply(
				channelID, messageID, replyTo, nickname, txt.Text, pubKey,
				codeset, timestamp, lease, round, Text, status, userMuted)
		} else {
			jww.ERROR.Printf("Failed process reply to for message %s from "+
				"public key %v (codeset %d) on channel %s, type %s, ts: %s, "+
				"lease: %s, round: %d, returning without reply",
				messageID, pubKey, codeset, channelID, messageType, timestamp,
				lease, round.ID)
			// Still process the message, but drop the reply because it is
			// malformed
		}
	}

	tag := makeChaDebugTag(channelID, pubKey, content, SendMessageTag)
	jww.INFO.Printf("[%s]Channels - Received message from %s "+
		"to %s on %s", tag, base64.StdEncoding.EncodeToString(pubKey),
		base64.StdEncoding.EncodeToString(txt.ReplyMessageID), channelID)

	return e.model.ReceiveMessage(channelID, messageID, nickname, txt.Text,
		pubKey, codeset, timestamp, lease, round, Text, status, userMuted)
}

// receiveReaction is the internal function that handles the reception of
// Reactions.
//
// It does edge checking to ensure the received reaction is just a single emoji.
// If the received reaction is not, the reaction is dropped.
// If the messageID for the message the reaction is to is malformed, the
// reaction is dropped.
func (e *events) receiveReaction(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType,
	nickname string, content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, _, userMuted bool) uint64 {
	react := &CMIXChannelReaction{}
	if err := proto.Unmarshal(content, react); err != nil {
		jww.ERROR.Printf("Failed to text unmarshal message %s from %x on "+
			"channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	// check that the reaction is a single emoji and ignore if it isn't
	if err := ValidateReaction(react.Reaction); err != nil {
		jww.ERROR.Printf("Failed process reaction %s from %x on channel "+
			"%s, type %s, ts: %s, lease: %s, round: %d, due to malformed "+
			"reaction (%s), ignoring reaction",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	if react.ReactionMessageID != nil &&
		len(react.ReactionMessageID) == cryptoChannel.MessageIDLen {
		var reactTo cryptoChannel.MessageID
		copy(reactTo[:], react.ReactionMessageID)

		tag := makeChaDebugTag(channelID, pubKey, content, SendReactionTag)
		jww.INFO.Printf("[%s]Channels - Received reaction from %s "+
			"to %s on %s", tag, base64.StdEncoding.EncodeToString(pubKey),
			base64.StdEncoding.EncodeToString(react.ReactionMessageID),
			channelID)

		return e.model.ReceiveReaction(channelID, messageID, reactTo, nickname,
			react.Reaction, pubKey, codeset, timestamp, lease, round, Reaction,
			status, userMuted)
	} else {
		jww.ERROR.Printf("Failed process reaction %s from public key %v "+
			"(codeset %d) on channel %s, type %s, ts: %s, lease: %s, "+
			"round: %d, reacting to invalid message, ignoring reaction",
			messageID, pubKey, codeset, channelID, messageType, timestamp,
			lease, round.ID)
	}
	return 0
}

// receiveDelete is the internal function that handles the reception of deleted
// messages.
//
// This function adheres to the MessageTypeReceiveMessage type.
func (e *events) receiveDelete(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, fromAdmin, userMuted bool) uint64 {

	msgLog := sprintfReceiveMessage(channelID, messageID, messageType, pubKey,
		codeset, timestamp, lease, round, fromAdmin)

	if userMuted {
		jww.ERROR.Printf("Muted user trying to delete message in %s", msgLog)
	}

	deleteMsg := &CMIXChannelDelete{}
	if err := proto.Unmarshal(content, deleteMsg); err != nil {
		jww.ERROR.Printf("Failed to proto unmarshal %T from payload in %s: %+v",
			deleteMsg, msgLog, err)
		return 0
	}

	deleteMessageID, err := cryptoChannel.UnmarshalMessageID(deleteMsg.MessageID)
	if err != nil {
		jww.ERROR.Printf("Failed unmarshal message ID of message targeted for "+
			"deletion in %s: %+v", msgLog, err)
		return 0
	}

	v := deleteVerb(deleteMsg.UndoAction)
	tag := makeChaDebugTag(channelID, pubKey, content, SendDeleteTag)
	jww.INFO.Printf("[%s]Channels - "+
		"Received message %s from %x to channel %s to %s message %s",
		tag, messageID, pubKey, channelID, v, deleteMessageID)

	// Reject the message deletion if not from original sender or admin
	if !fromAdmin {
		targetMsg, err2 := e.model.GetMessage(deleteMessageID)
		if err2 != nil {
			jww.ERROR.Printf("Failed to find target message %s for deletion "+
				"from %s: %+v", deleteMsg, msgLog, err2)
			return 0
		}
		if !bytes.Equal(targetMsg.PubKey, pubKey) {
			jww.ERROR.Printf("Deletion message must come from original sender "+
				"or admin for %s", msgLog)
			return 0
		}
	}

	deleteMsg.UndoAction = true
	payload, err := proto.Marshal(deleteMsg)
	if err != nil {
		jww.ERROR.Printf("Failed to proto marshal %T from payload in %s: %+v",
			deleteMsg, msgLog, err)
		return 0
	}

	if deleteMsg.UndoAction {
		e.leases.removeMessage(channelID, messageType, payload)
		deleted := false
		return e.model.UpdateFromMessageID(
			messageID, nil, nil, &deleted, nil, nil)
	} else {
		e.leases.addMessage(channelID, messageID, messageType, nickname,
			payload, timestamp, lease, round, status)

		deleted := true
		return e.model.UpdateFromMessageID(
			messageID, nil, nil, &deleted, nil, nil)
	}
}

// receivePinned is the internal function that handles the reception of pinned
// messages.
//
// This function adheres to the MessageTypeReceiveMessage type.
func (e *events) receivePinned(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, fromAdmin, userMuted bool) uint64 {

	msgLog := sprintfReceiveMessage(channelID, messageID, messageType, pubKey,
		codeset, timestamp, lease, round, fromAdmin)

	if userMuted {
		jww.ERROR.Printf("Muted user trying to pin message in %s", msgLog)
	}

	// Reject the message pin if it is not from the admin
	if !fromAdmin {
		jww.ERROR.Printf("Pin message must come from admin for %s", msgLog)
		return 0
	}

	pinnedMsg := &CMIXChannelPinned{}
	if err := proto.Unmarshal(content, pinnedMsg); err != nil {
		jww.ERROR.Printf("Failed to proto unmarshal %T from payload in %s: %+v",
			pinnedMsg, msgLog, err)
		return 0
	}

	pinnedMessageID, err := cryptoChannel.UnmarshalMessageID(pinnedMsg.MessageID)
	if err != nil {
		jww.ERROR.Printf("Failed unmarshal message ID of message targeted for "+
			"pinning in %s: %+v", msgLog, err)
		return 0
	}

	v := pinnedVerb(pinnedMsg.UndoAction)
	tag := makeChaDebugTag(channelID, pubKey, content, SendPinnedTag)
	jww.INFO.Printf("[%s]Channels - "+
		"Received message %s from %x to channel %s to %s message %s",
		tag, messageID, pubKey, channelID, v, pinnedMessageID)

	pinnedMsg.UndoAction = true
	payload, err := proto.Marshal(pinnedMsg)
	if err != nil {
		jww.ERROR.Printf("Failed to proto marshal %T from payload in %s: %+v",
			pinnedMsg, msgLog, err)
		return 0
	}

	if pinnedMsg.UndoAction {
		e.leases.removeMessage(channelID, messageType, payload)
		pinned := false

		return e.model.UpdateFromMessageID(
			messageID, nil, nil, &pinned, nil, nil)
	} else {
		e.leases.addMessage(channelID, messageID, messageType, nickname,
			payload, timestamp, lease, round, status)

		pinned := true
		return e.model.UpdateFromMessageID(
			messageID, nil, nil, &pinned, nil, nil)
	}
}

// receiveMute is the internal function that handles the reception of muted
// users.
//
// This function adheres to the MessageTypeReceiveMessage type.
func (e *events) receiveMute(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType, nickname string,
	content []byte, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus, fromAdmin, userMuted bool) uint64 {

	msgLog := sprintfReceiveMessage(channelID, messageID, messageType, pubKey,
		codeset, timestamp, lease, round, fromAdmin)

	if userMuted {
		jww.ERROR.Printf("Muted user trying to mute user in %s", msgLog)
		return 0
	}

	// Reject the message pin if it is not from the admin
	if !fromAdmin {
		jww.ERROR.Printf("Mute user must come from admin for %s", msgLog)
		return 0
	}

	muteMsg := &CMIXChannelMute{}
	if err := proto.Unmarshal(content, muteMsg); err != nil {
		jww.ERROR.Printf("Failed to proto unmarshal %T from payload in %s: %+v",
			muteMsg, msgLog, err)
		return 0
	}

	if len(muteMsg.PubKey) != ed25519.PublicKeySize {
		jww.ERROR.Printf("Failed unmarshal public key of user targeted for "+
			"pinning in %s: length of %d bytes required, received %d bytes",
			msgLog, ed25519.PublicKeySize, len(muteMsg.PubKey))
		return 0
	}

	var mutedUser ed25519.PublicKey
	copy(mutedUser[:], muteMsg.PubKey)

	tag := makeChaDebugTag(channelID, pubKey, content, SendMuteTag)
	jww.INFO.Printf("[%s]Channels - "+
		"Received message %s from %x to channel %s to %s user %x", tag,
		messageID, pubKey, channelID, muteVerb(muteMsg.UndoAction), mutedUser)

	muteMsg.UndoAction = true
	payload, err := proto.Marshal(muteMsg)
	if err != nil {
		jww.ERROR.Printf("Failed to proto marshal %T from payload in %s: %+v",
			muteMsg, msgLog, err)
		return 0
	}

	if muteMsg.UndoAction {
		e.leases.removeMessage(channelID, messageType, payload)
		e.mutedUsers.unmuteUser(channelID, pubKey)
		return 0
	} else {
		e.leases.addMessage(channelID, messageID, messageType, nickname,
			payload, timestamp, lease, round, status)
		e.mutedUsers.muteUser(channelID, pubKey)
		return 0
	}
}

func sprintfReceiveMessage(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType,
	pubKey ed25519.PublicKey, codeset uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, fromAdmin bool) string {
	return fmt.Sprintf("message %s from %x (codeset %d) on channel %s "+
		"{type:%s timestamp:%s lease:%s round:%d fromAdmin:%t}", messageID,
		pubKey, codeset, channelID, messageType, timestamp.Round(0), lease,
		round.ID, fromAdmin)
}

// deleteVerb returns the correct verb for the delete action to use for logging
// and debugging.
func deleteVerb(b bool) string {
	if b {
		return "delete"
	}
	return "un-delete"
}

// pinnedVerb returns the correct verb for the pinned action to use for logging
// and debugging.
func pinnedVerb(b bool) string {
	if b {
		return "pin"
	}
	return "unpin"
}

// muteVerb returns the correct verb for the mute action to use for logging and
// debugging.
func muteVerb(b bool) string {
	if b {
		return "mute"
	}
	return "unmute"
}
