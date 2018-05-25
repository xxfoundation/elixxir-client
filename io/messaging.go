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
func SendMessage(recipientID uint64, message string) error {
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
		err = client.SendPutMessage(SendAddress, msgPacket)
	} else {
		_, err = client.SendMessageToServer(SendAddress, msgPacket)
	}

	return err
}

// Listen adds a listener to the receiver thread
func Listen(senderID uint64) chan *format.Message {
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

// Delete the listener
func StopListening(listenerCh chan *format.Message) {
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

// Polling thread for receiving messages -- again.. we should be passing
// this a user object with some keys, and maybe a shared list for the listeners?
// Accessing all of these global variables is extremely problematic for this
// kind of thread.
func MessageReceiver(delay time.Duration) {
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
		jww.WARN.Printf("Poll")
		encryptedMsg, err := client.SendClientPoll(ReceiveAddress, &pollingMessage)
		if err != nil {
			jww.WARN.Printf("MessageReceiver error during Polling: %v", err.Error())
			continue
		}
		if encryptedMsg.MessagePayload == nil &&
			encryptedMsg.RecipientID == nil &&
			encryptedMsg.SenderID == 0 {
			continue
		}

		jww.WARN.Printf("DECrypt")

		decryptedMsg, err2 := crypto.Decrypt(crypto.Grp, encryptedMsg)
		if err2 != nil {
			jww.WARN.Printf("Message did not decrypt properly: %v", err2.Error())
		}

		jww.WARN.Printf("POPULATE")
		senderID := decryptedMsg.GetSenderIDUint()
		listenersLock.Lock()
		for i := range listeners {
			jww.WARN.Printf("ADDLIST")

			// Skip if not 0 or not senderID matched
			if listeners[i].SenderID != 0 || listeners[i].SenderID != senderID {
				continue
			}
			listeners[i].Messages <- decryptedMsg
		}
		listenersLock.Unlock()
	}
}
