////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/emoji"

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
		nickname, text string, pubKey ed25519.PublicKey, dmToken uint32,
		codeset uint8, timestamp time.Time, lease time.Duration,
		round rounds.Round, mType MessageType, status SentStatus) uint64

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
	ReceiveReply(channelID *id.ID, messageID cryptoChannel.MessageID,
		reactionTo cryptoChannel.MessageID, nickname, text string,
		pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
		timestamp time.Time, lease time.Duration, round rounds.Round,
		mType MessageType, status SentStatus) uint64

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
	ReceiveReaction(channelID *id.ID, messageID cryptoChannel.MessageID,
		reactionTo cryptoChannel.MessageID, nickname, reaction string,
		pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
		timestamp time.Time, lease time.Duration, round rounds.Round,
		mType MessageType, status SentStatus) uint64

	// UpdateSentStatus is called whenever the sent status of a message has
	// changed.
	//
	// messageID, timestamp, and round are all nillable and may be updated based
	// upon the UUID at a later date. A time of time.Time{} will be passed for a
	// nilled timestamp. If a nil value is passed, make no update.
	UpdateSentStatus(uuid uint64, messageID cryptoChannel.MessageID,
		timestamp time.Time, round rounds.Round, status SentStatus)

	// unimplemented
	// IgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	// UnIgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	// PinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID, end time.Time)
	// UnPinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
}

// MessageTypeReceiveMessage defines handlers for messages of various message
// types. Default ones for Text, Reaction, and AdminText.
//
// A unique UUID must be returned by which the message can be referenced later
// via [EventModel.UpdateSentStatus].
//
// It must return a unique UUID for the message by which it can be referenced
// later.
type MessageTypeReceiveMessage func(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType,
	nickname string, content []byte, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration,
	round rounds.Round, status SentStatus) uint64

// updateStatusFunc is a function type for EventModel.UpdateSentStatus so it can
// be mocked for testing where used.
type updateStatusFunc func(uuid uint64, messageID cryptoChannel.MessageID,
	timestamp time.Time, round rounds.Round, status SentStatus)

// events is an internal structure that processes events and stores the handlers
// for those events.
type events struct {
	model      EventModel
	registered map[MessageType]MessageTypeReceiveMessage
	mux        sync.RWMutex
}

// initEvents initializes the event model and registers default message type
// handlers.
func initEvents(model EventModel) *events {
	e := &events{
		model:      model,
		registered: make(map[MessageType]MessageTypeReceiveMessage),
		mux:        sync.RWMutex{},
	}

	// set up default message types
	e.registered[Text] = e.receiveTextMessage
	e.registered[AdminText] = e.receiveTextMessage
	e.registered[Reaction] = e.receiveReaction
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

type triggerEventFunc func(chID *id.ID, umi *userMessageInternal, ts time.Time,
	receptionID receptionID.EphemeralIdentity, round rounds.Round,
	status SentStatus) (uint64, error)

// triggerEvent is an internal function that is used to trigger message
// reception on a message received from a user (symmetric encryption).
//
// It will call the appropriate MessageTypeHandler assuming one exists.
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
		errStr := fmt.Sprintf("Received message from %x on channel %s in "+
			"round %d which could not be handled due to unregistered message "+
			"type %s; Contents: %v",
			um.ECCPublicKey, chID, round.ID, messageType, cm.Payload)
		jww.WARN.Printf(errStr)
		return 0, errors.New(errStr)
	}

	// Call the listener. This is already in an instanced event; no new thread
	// is needed.
	uuid := listener(
		chID, umi.GetMessageID(), messageType, cm.Nickname, cm.Payload,
		um.ECCPublicKey, cm.DMToken, 0, ts, time.Duration(cm.Lease), round,
		status)
	return uuid, nil
}

type triggerAdminEventFunc func(chID *id.ID, cm *ChannelMessage, ts time.Time,
	messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
	round rounds.Round, status SentStatus) (uint64, error)

// triggerAdminEvent is an internal function that is used to trigger message
// reception on a message received from the admin (asymmetric encryption).
//
// It will call the appropriate MessageTypeHandler assuming one exists.
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
		errStr := fmt.Sprintf("Received Admin message from %s on channel %s "+
			"in round %d which could not be handled due to unregistered "+
			"message type %s; Contents: %v",
			AdminUsername, chID, round.ID, messageType, cm.Payload)
		jww.WARN.Printf(errStr)
		return 0, errors.New(errStr)
	}

	// Call the listener. This is already in an instanced event; no new thread
	// is needed.
	uuid := listener(
		chID, messageID, messageType, AdminUsername, cm.Payload,
		AdminFakePubKey, cm.DMToken, 0, ts, time.Duration(cm.Lease), round,
		status)
	return uuid, nil
}

// receiveTextMessage is the internal function that handles the reception of
// text messages. It handles both messages and replies and calls the correct
// function on the event model.
//
// If the message has a reply, but it is malformed, it will drop the reply and
// write to the log.
func (e *events) receiveTextMessage(channelID *id.ID,
	messageID cryptoChannel.MessageID, messageType MessageType,
	nickname string, content []byte, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus) uint64 {
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
			return e.model.ReceiveReply(channelID, messageID, replyTo, nickname,
				txt.Text, pubKey, dmToken, codeset, timestamp, lease, round,
				Text, status)

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
		pubKey, dmToken, codeset, timestamp, lease, round, Text, status)
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
	nickname string, content []byte, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	status SentStatus) uint64 {
	react := &CMIXChannelReaction{}
	if err := proto.Unmarshal(content, react); err != nil {
		jww.ERROR.Printf("Failed to text unmarshal message %s from %x on "+
			"channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	// check that the reaction is a single emoji and ignore if it isn't
	if err := emoji.ValidateReaction(react.Reaction); err != nil {
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
			react.Reaction, pubKey, dmToken, codeset, timestamp, lease, round, Reaction,
			status)
	} else {
		jww.ERROR.Printf("Failed process reaction %s from public key %v "+
			"(codeset %d) on channel %s, type %s, ts: %s, lease: %s, "+
			"round: %d, reacting to invalid message, ignoring reaction",
			messageID, pubKey, codeset, channelID, messageType, timestamp,
			lease, round.ID)
	}
	return 0
}
