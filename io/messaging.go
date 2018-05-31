////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io asynchronous sending functionality. This is managed by an outgoing
// messages channel and managed by the sender thread kicked off during
// initialization.
package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/comms/client"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/format"
	"sync"
	"time"
)

type messaging struct{}

// Messaging implements the Communications interface
var Messaging Communications = &messaging{}

// SendAddress is the address of the server to send messages
var SendAddress string

// ReceiveAddress is the address of the server to receive messages from
var ReceiveAddress string

// UseGateway decides if we should use the gateway calls instead of
// the deprecated server RPCs
var UseGateway = false

// BlockTransmissions will use a mutex to prevent multiple threads from sending
// messages at the same time.
var BlockTransmissions = true

// TransmitDelay is the minimum delay between transmissions.
var TransmitDelay = 1000 * time.Millisecond

// Map that holds a record of the messages that this client successfully
// received during this session
var ReceivedMessages map[string]struct{}
var lastReceivedMessageID string = ""

var sendLock sync.Mutex

// MessageListener allows threads to listen for messages from specific
// users
type messageListener struct {
	SenderID uint64
	Messages chan *format.Message
}

var listeners []messageListener
var listenersLock sync.Mutex

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
func (m *messaging) SendMessage(recipientID uint64, message string) error {
	// FIXME: We should really bring the plaintext parts of the NewMessage logic
	// into this module, then have an EncryptedMessage type that is sent to/from
	// the cMix network. This NewMessage does way too many things: break the
	// message into parts, generate mic's, etc -- the crypto library should only
	// know about the crypto and advertise a max message payload size

	// TBD: Is there a really good reason why we'd ever have more than one user
	// in this library? why not pass a sender object instead?
	userID := globals.Session.GetCurrentUser().UserID
	messages, err := format.NewMessage(userID, recipientID, message)

	if err != nil {
		return err
	}
	for i := range messages {
		err = send(userID, &messages[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// send actually sends the message to the server
func send(senderID uint64, message *format.Message) error {
	// Enable transmission blocking if enabled
	if BlockTransmissions {
		sendLock.Lock()
		defer func() {
			time.Sleep(TransmitDelay)
			sendLock.Unlock()
		}()
	}

	// TBD: Is there a really good reason we have to specify the Grp and not a
	// key? Should we even be doing the encryption here?
	encryptedMessage := crypto.Encrypt(crypto.Grp, message)
	msgPacket := &pb.CmixMessage{
		SenderID:       senderID,
		MessagePayload: encryptedMessage.Payload.Bytes(),
		RecipientID:    encryptedMessage.Recipient.Bytes(),
	}

	var err error
	if UseGateway {
		jww.FATAL.Println("Sending put message to gateway")
		err = client.SendPutMessage(SendAddress, msgPacket)
	} else {
		_, err = client.SendMessageToServer(SendAddress, msgPacket)
	}

	return err
}

// Listen adds a listener to the receiver thread
func (m *messaging) Listen(senderID uint64) chan *format.Message {
	listenersLock.Lock()
	defer listenersLock.Unlock()
	if listeners == nil {
		listeners = make([]messageListener, 0)
	}
	listenerCh := make(chan *format.Message, 10)
	listener := messageListener{
		SenderID: senderID,
		Messages: listenerCh,
	}
	listeners = append(listeners, listener)
	return listenerCh
}

// StopListening closes and deletes the listener
func (m *messaging) StopListening(listenerCh chan *format.Message) {
	listenersLock.Lock()
	defer listenersLock.Unlock()
	for i := range listeners {
		if listeners[i].Messages == listenerCh {
			close(listeners[i].Messages)
			if len(listeners) == i+1 {
				listeners = listeners[:i]
			} else {
				listeners = append(listeners[:i], listeners[i+1:]...)
			}
		}
	}
}

// MessageReceiver is a polling thread for receiving messages -- again.. we
// should be passing this a user object with some keys, and maybe a shared
// list for the listeners?
// Accessing all of these global variables is extremely problematic for this
// kind of thread.
func (m *messaging) MessageReceiver(delay time.Duration) {
	// FIXME: It's not clear we should be doing decryption here.
	if globals.Session == nil {
		jww.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientPollMessage{
		UserID: globals.Session.GetCurrentUser().UserID,
	}

	for {
		time.Sleep(delay)
		if len(listeners) == 0 {
			jww.FATAL.Panicf("No listeners for receiver thread!")
		}
		if UseGateway {
			jww.FATAL.Printf("Attempting to receive message from gateway")
			decryptedMessage := m.receiveMessageFromGateway(&pollingMessage)
			if decryptedMessage != nil {
				broadcastMessageReception(decryptedMessage)
			}
		} else {
			m.receiveMessageFromServer(&pollingMessage)
		}
	}
}

func (m *messaging) receiveMessageFromGateway(
	pollingMessage *pb.ClientPollMessage) *format.Message {
	pollingMessage.MessageID = lastReceivedMessageID
	messages, err := client.SendCheckMessages(globals.Session.GetGWAddress(),
		pollingMessage)

	if err != nil {
		jww.WARN.Printf("CheckMessages error during polling: %v", err.Error())
		return nil
	}

	jww.FATAL.Printf("Checking novelty of %v messages", len(messages.MessageIDs))

	if ReceivedMessages == nil {
		ReceivedMessages = make(map[string]struct{})
	}

	for _, messageID := range messages.MessageIDs {
		// Get the first unseen message from the list of IDs
		_, received := ReceivedMessages[messageID]
		if !received {
			jww.FATAL.Printf("Got a message waiting on the gateway: %v", messageID)
			// We haven't seen this message before.
			// So, we should retrieve it from the gateway.
			newMessage, err := client.SendGetMessage(globals.
				Session.GetGWAddress(),
				&pb.ClientPollMessage{
					UserID:    globals.Session.GetCurrentUser().UserID,
					MessageID: messageID,
				})
			if err != nil {
				jww.WARN.Printf(
					"Couldn't receive message with ID %v while"+
						" polling gateway", messageID)
			} else {
				if newMessage.MessagePayload == nil &&
					newMessage.RecipientID == nil &&
					newMessage.SenderID == 0 {
					jww.FATAL.Println("Message fields not populated")
					return nil
				}
				decryptedMsg, err2 := crypto.Decrypt(crypto.Grp, newMessage)
				jww.FATAL.Println(decryptedMsg.GetPayload())
				if err2 != nil {
					jww.WARN.Printf("Message did not decrypt properly: %v", err2.Error())
				}
				ReceivedMessages[messageID] = struct{}{}
				lastReceivedMessageID = messageID

				return decryptedMsg
			}
		}
	}
	return nil
}

func (m *messaging) receiveMessageFromServer(pollingMessage *pb.ClientPollMessage) {
	encryptedMsg, err := client.SendClientPoll(ReceiveAddress, pollingMessage)
	if err != nil {
		jww.WARN.Printf("MessageReceiver error during Polling: %v", err.Error())
		return
	}
	if encryptedMsg.MessagePayload == nil &&
		encryptedMsg.RecipientID == nil &&
		encryptedMsg.SenderID == 0 {
		return
	}

	decryptedMsg, err2 := crypto.Decrypt(crypto.Grp, encryptedMsg)
	if err2 != nil {
		jww.WARN.Printf("Message did not decrypt properly: %v", err2.Error())
	}

	broadcastMessageReception(decryptedMsg)
}

func broadcastMessageReception(decryptedMsg *format.Message) {
	jww.FATAL.Println("Attempting to broadcast received message")
	senderID := decryptedMsg.GetSenderIDUint()
	listenersLock.Lock()
	for i := range listeners {
		// Skip if not 0 or not senderID matched
		if listeners[i].SenderID != 0 && listeners[i].SenderID != senderID {
			continue
		}
		listeners[i].Messages <- decryptedMsg
	}

	// FIXME: RemovRe this.
	// Send the message to any global listener
	if globals.UsingReceiver() {
		jww.WARN.Printf("This client implemenation is using the deprecated " +
			"globals.Receiver API. This will stop working shortly.")
		err := globals.Receive(decryptedMsg)
		if err != nil {
			jww.ERROR.Printf("Could not call global receiver: %s", err.Error())
		}
	}
	listenersLock.Unlock()
}
