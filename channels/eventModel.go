package channels

import (
	"errors"
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

var (
	MessageTypeAlreadyRegistered = errors.New("the given message type has " +
		"already been registered")
)

type EventModel interface {
	JoinChannel(channel cryptoBroadcast.Channel)
	LeaveChannel(ChannelID *id.ID)

	ReceiveTextMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID,
		messageType MessageType, SenderUsername string, Content []byte,
		timestamp time.Time, lease time.Duration, round rounds.Round)
	ReceiveAdminTextMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID,
		messageType MessageType, SenderUsername string, Content []byte,
		timestamp time.Time, lease time.Duration, round rounds.Round)
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

func initEvents(model EventModel) *events {
	e := &events{
		model:      model,
		registered: make(map[MessageType]MessageTypeReceiveMessage),
		mux:        sync.RWMutex{},
	}

	//set up default message types
	e.registered[Text] = e.model.ReceiveTextMessage
	e.registered[AdminText] = e.model.ReceiveAdminTextMessage
	return e
}

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

func (e *events) hear(chID *id.ID, umi *UserMessageInternal,
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
