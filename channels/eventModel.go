package channels

import (
	"errors"
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/primitives/states"
	"sync"
	"time"

	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
)

const AdminUsername = "Admin"

var (
	MessageTypeAlreadyRegistered = errors.New("the given message type has " +
		"already been registered")
)

// EventModel is an interface which an external party which uses the channels
// system passed an object which adheres to in order to get events on the channel
type EventModel interface {
	// JoinChannel is called whenever a channel is joined locally
	JoinChannel(channel cryptoBroadcast.Channel)

	// LeaveChannel is called whenever a channel is left locally
	LeaveChannel(ChannelID *id.ID)

	// ReceiveMessage is called whenever a message is received on a given channel
	// It may be called multiple times on the same message, it is incumbent on
	// the user of the API to filter such called by message ID
	ReceiveMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID,
		SenderUsername string, text string,
		timestamp time.Time, lease time.Duration, round rounds.Round)

	// ReceiveReply is called whenever a message is received which is a reply
	// on a given channel. It may be called multiple times on the same message,
	// it is incumbent on the user of the API to filter such called by message ID
	// Messages may arrive our of order, so a reply in theory can arrive before
	// the initial message, as a result it may be important to buffer replies.
	ReceiveReply(ChannelID *id.ID, MessageID cryptoChannel.MessageID,
		ReplyTo cryptoChannel.MessageID, SenderUsername string,
		text string, timestamp time.Time, lease time.Duration,
		round rounds.Round)

	// ReceiveReaction is called whenever a reaction to a message is received
	// on a given channel. It may be called multiple times on the same reaction,
	// it is incumbent on the user of the API to filter such called by message ID
	// Messages may arrive our of order, so a reply in theory can arrive before
	// the initial message, as a result it may be important to buffer reactions.
	ReceiveReaction(ChannelID *id.ID, MessageID cryptoChannel.MessageID,
		ReactionTo cryptoChannel.MessageID, SenderUsername string,
		Reaction string, timestamp time.Time, lease time.Duration,
		round rounds.Round)

	//unimplemented
	//IgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	//UnIgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	//PinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID, end time.Time)
	//UnPinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
}

type MessageTypeReceiveMessage func(ChannelID *id.ID,
	MessageID cryptoChannel.MessageID, messageType MessageType,
	SenderUsername string, Content []byte, timestamp time.Time,
	lease time.Duration, round rounds.Round)

type events struct {
	model      EventModel
	registered map[MessageType]MessageTypeReceiveMessage
	mux        sync.RWMutex
}

// initEvents initializes the event model and registers default message type
// handlers
func initEvents(model EventModel) *events {
	e := &events{
		model:      model,
		registered: make(map[MessageType]MessageTypeReceiveMessage),
		mux:        sync.RWMutex{},
	}

	//set up default message types
	e.registered[Text] = e.receiveTextMessage
	e.registered[AdminText] = e.receiveTextMessage
	e.registered[Reaction] = e.receiveReaction
	return e
}

// RegisterReceiveHandler is used to register handlers for non default message
// types s they can be processed by modules. it is important that such modules
// sync up with the event model implementation.
// There can only be one handler per message type, and this will return an error
// on a multiple registration.
func (e *events) RegisterReceiveHandler(messageType MessageType,
	listener MessageTypeReceiveMessage) error {
	e.mux.Lock()
	defer e.mux.Unlock()

	//check if the type is already registered
	if _, exists := e.registered[messageType]; exists {
		return MessageTypeAlreadyRegistered
	}

	//register the message type
	e.registered[messageType] = listener
	jww.INFO.Printf("Registered Listener for Message Type %s", messageType)
	return nil
}

// triggerEvent is an internal function which is used to trigger message
// reception on a message received from a user (symmetric encryption)
// It will call the appropriate MessageTypeHandler assuming one exists.
func (e *events) triggerEvent(chID *id.ID, umi *UserMessageInternal,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	um := umi.GetUserMessage()
	cm := umi.GetChannelMessage()
	messageType := MessageType(cm.PayloadType)

	//check if the type is already registered
	e.mux.RLock()
	listener, exists := e.registered[messageType]
	e.mux.RUnlock()
	if !exists {
		jww.WARN.Printf("Received message from %s on channel %s in "+
			"round %d which could not be handled due to unregistered message "+
			"type %s; Contents: %v", um.Username, chID, round.ID, messageType,
			cm.Payload)
	}

	//Call the listener. This is already in an instanced event, no new thread needed.
	listener(chID, umi.GetMessageID(), messageType, um.Username,
		cm.Payload, round.Timestamps[states.QUEUED], time.Duration(cm.Lease), round)
	return
}

// triggerAdminEvent is an internal function which is used to trigger message
// reception on a message received from the admin (asymmetric encryption)
// It will call the appropriate MessageTypeHandler assuming one exists.
func (e *events) triggerAdminEvent(chID *id.ID, cm *ChannelMessage,
	messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	messageType := MessageType(cm.PayloadType)

	//check if the type is already registered
	e.mux.RLock()
	listener, exists := e.registered[messageType]
	e.mux.RUnlock()
	if !exists {
		jww.WARN.Printf("Received Admin message from %s on channel %s in "+
			"round %d which could not be handled due to unregistered message "+
			"type %s; Contents: %v", AdminUsername, chID, round.ID, messageType,
			cm.Payload)
	}

	//Call the listener. This is already in an instanced event, no new thread needed.
	listener(chID, messageID, messageType, AdminUsername,
		cm.Payload, round.Timestamps[states.QUEUED], time.Duration(cm.Lease), round)
	return
}

// receiveTextMessage is the internal function which handles the reception of
// text messages. It handles both messages and replies and calls the correct
// function on the event model.
// If the message has a reply but it is malformed, it will drop the reply and
// write to the log
func (e *events) receiveTextMessage(ChannelID *id.ID,
	MessageID cryptoChannel.MessageID, messageType MessageType,
	SenderUsername string, Content []byte, timestamp time.Time,
	lease time.Duration, round rounds.Round) {
	txt := &CMIXChannelText{}
	if err := proto.Unmarshal(Content, txt); err != nil {
		jww.ERROR.Printf("Failed to text unmarshal message %s from %s on "+
			"channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			MessageID, SenderUsername, ChannelID, messageType, timestamp, lease,
			round.ID, err)
		return
	}

	if txt.ReplyMessageID != nil {
		if len(txt.ReplyMessageID) == cryptoChannel.MessageIDLen {
			var replyTo cryptoChannel.MessageID
			copy(replyTo[:], txt.ReplyMessageID)
			e.model.ReceiveReply(ChannelID, MessageID, replyTo, SenderUsername, txt.Text,
				timestamp, lease, round)
			return

		} else {
			jww.ERROR.Printf("Failed process reply to for message %s from %s on "+
				"channel %s, type %s, ts: %s, lease: %s, round: %d, returning "+
				"without reply",
				MessageID, SenderUsername, ChannelID, messageType, timestamp, lease,
				round.ID)
		}
	}

	e.model.ReceiveMessage(ChannelID, MessageID, SenderUsername, txt.Text,
		timestamp, lease, round)
}

// receiveReaction is the internal function which handles the reception of
// Reactions.
// It does edge chaling
func (e *events) receiveReaction(ChannelID *id.ID,
	MessageID cryptoChannel.MessageID, messageType MessageType,
	SenderUsername string, Content []byte, timestamp time.Time,
	lease time.Duration, round rounds.Round) {
	react := &CMIXChannelReaction{}
	if err := proto.Unmarshal(Content, react); err != nil {
		jww.ERROR.Printf("Failed to text unmarshal message %s from %s on "+
			"channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			MessageID, SenderUsername, ChannelID, messageType, timestamp, lease,
			round.ID, err)
		return
	}

	//check that the reaction is a single emoji and ignore if it isn't
	if err := ValidateReaction(react.Reaction); err != nil {
		jww.ERROR.Printf("Failed process reaction %s from %s on channel "+
			"%s, type %s, ts: %s, lease: %s, round: %d, due to malformed "+
			"reaction (%s), ignoring reaction",
			MessageID, SenderUsername, ChannelID, messageType, timestamp, lease,
			round.ID, err)
	}

	if react.ReactionMessageID != nil && len(react.ReactionMessageID) == cryptoChannel.MessageIDLen {
		var reactTo cryptoChannel.MessageID
		copy(reactTo[:], react.ReactionMessageID)
		e.model.ReceiveReaction(ChannelID, MessageID, reactTo, SenderUsername,
			react.Reaction, timestamp, lease, round)
	} else {
		jww.ERROR.Printf("Failed process reaction %s from %s on channel "+
			"%s, type %s, ts: %s, lease: %s, round: %d, reacting to "+
			"invalid message, ignoring reaction",
			MessageID, SenderUsername, ChannelID, messageType, timestamp, lease,
			round.ID)
	}
}
