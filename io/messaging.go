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
	"gitlab.com/privategrity/client/listener"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
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

// BlockTransmissions will use a mutex to prevent multiple threads from sending
// messages at the same time.
var BlockTransmissions = true

// TransmitDelay is the minimum delay between transmissions.
var TransmitDelay = 1000 * time.Millisecond

// Map that holds a record of the messages that this client successfully
// received during this session
var ReceivedMessages map[string]struct{}
var lastReceivedMessageID = ""

var sendLock sync.Mutex

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
func (m *messaging) SendMessage(recipientID user.ID,
	message string) error {
	// FIXME: We should really bring the plaintext parts of the NewMessage logic
	// into this module, then have an EncryptedMessage type that is sent to/from
	// the cMix network. This NewMessage does way too many things: break the
	// message into parts, generate mic's, etc -- the crypto library should only
	// know about the crypto and advertise a max message payload size

	// TBD: Is there a really good reason why we'd ever have more than one user
	// in this library? why not pass a sender object instead?
	userID := user.TheSession.GetCurrentUser().UserID
	messages, err := format.NewMessage(uint64(userID), uint64(recipientID),
		message)

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
func send(senderID user.ID, message *format.Message) error {
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
		SenderID:       uint64(senderID),
		MessagePayload: encryptedMessage.Payload.Bytes(),
		RecipientID:    encryptedMessage.Recipient.Bytes(),
	}

	var err error
	jww.INFO.Println("Sending put message to gateway")
	err = client.SendPutMessage(SendAddress, msgPacket)

	return err
}

// MessageReceiver is a polling thread for receiving messages -- again.. we
// should be passing this a user object with some keys, and maybe a shared
// list for the listeners?
// Accessing all of these global variables is extremely problematic for this
// kind of thread.
func (m *messaging) MessageReceiver(delay time.Duration) {
	// FIXME: It's not clear we should be doing decryption here.
	if user.TheSession == nil {
		jww.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientPollMessage{
		UserID: uint64(user.TheSession.GetCurrentUser().UserID),
	}

	for {
		time.Sleep(delay)
		jww.INFO.Printf("Attempting to receive message from gateway")
		decryptedMessage := m.receiveMessageFromGateway(&pollingMessage)
		if decryptedMessage != nil {
			broadcastMessageReception(decryptedMessage, listener.Listeners)
		}
	}
}

func (m *messaging) receiveMessageFromGateway(
	pollingMessage *pb.ClientPollMessage) *format.Message {
	pollingMessage.MessageID = lastReceivedMessageID
	if user.TheSession != nil {
		messages, err := client.SendCheckMessages(user.TheSession.GetGWAddress(),
			pollingMessage)

		if err != nil {
			jww.WARN.Printf("CheckMessages error during polling: %v", err.Error())
			return nil
		}

		jww.INFO.Printf("Checking novelty of %v messages", len(messages.MessageIDs))

		if ReceivedMessages == nil {
			ReceivedMessages = make(map[string]struct{})
		}

		for _, messageID := range messages.MessageIDs {
			// Get the first unseen message from the list of IDs
			_, received := ReceivedMessages[messageID]
			if !received {
				jww.INFO.Printf("Got a message waiting on the gateway: %v",
					messageID)
				// We haven't seen this message before.
				// So, we should retrieve it from the gateway.
				newMessage, err := client.SendGetMessage(user.
					TheSession.GetGWAddress(),
					&pb.ClientPollMessage{
						UserID:    uint64(user.TheSession.GetCurrentUser().UserID),
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
						jww.INFO.Println("Message fields not populated")
						return nil
					}
					decryptedMsg, err2 := crypto.Decrypt(crypto.Grp, newMessage)
					if err2 != nil {
						jww.WARN.Printf("Message did not decrypt properly: %v", err2.Error())
					}
					ReceivedMessages[messageID] = struct{}{}
					lastReceivedMessageID = messageID

					return decryptedMsg
				}
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

	broadcastMessageReception(decryptedMsg, listener.Listeners)
}

func broadcastMessageReception(decryptedMsg *format.Message,
	listeners *listener.ListenerMap) {
	jww.INFO.Println("Attempting to broadcast received message")
	typedBody, err := parse.Parse([]byte(decryptedMsg.GetPayload()))
	// Panic the error for now
	if err != nil {
		panic(err.Error())
	}
	listeners.Speak(&parse.Message{
		TypedBody: *typedBody,
		Sender:    user.NewIDFromBytes(decryptedMsg.GetSender()),
		Receiver:  user.NewIDFromBytes(decryptedMsg.GetRecipient()),
	})
}
