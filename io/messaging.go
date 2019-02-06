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
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	cmix "gitlab.com/elixxir/crypto/messaging"
	"sync"
	"time"
	"gitlab.com/elixxir/primitives/userid"
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

var sendLock sync.Mutex

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
// TODO This method would be cleaner if it took a parse.Message (particularly
// w.r.t. generating message IDs for multi-part messages.)
func (m *messaging) SendMessage(recipientID *id.UserID,
	message []byte) error {
	// FIXME: We should really bring the plaintext parts of the NewMessage logic
	// into this module, then have an EncryptedMessage type that is sent to/from
	// the cMix network. This NewMessage does way too many things: break the
	// message into parts, generate mic's, etc -- the crypto library should only
	// know about the crypto and advertise a max message payload size

	// TBD: Is there a really good reason why we'd ever have more than one user
	// in this library? why not pass a sender object instead?
	globals.Log.DEBUG.Printf("Sending message to %q: %q", *recipientID, message)
	userID := user.TheSession.GetCurrentUser().UserID
	parts, err := parse.Partition([]byte(message),
		parse.CurrentCounter.NextID())
	if err != nil {
		return err
	}
	for i := range parts {
		messages, err := format.NewMessage(userID, recipientID, parts[i])
		if err != nil {
			return err
		}
		if len(messages) != 1 {
			globals.Log.ERROR.Printf("Expected one message from already-partitioned"+
				" message of length %v. Got %v messages instead.",
				len(parts[i]), len(messages))
		}
		err = send(userID, &messages[0])
		if err != nil {
			return err
		}
	}
	return nil
}

// send actually sends the message to the server
func send(senderID *id.UserID, message *format.Message) error {
	// Enable transmission blocking if enabled
	if BlockTransmissions {
		sendLock.Lock()
		defer func() {
			time.Sleep(TransmitDelay)
			sendLock.Unlock()
		}()
	}

	salt := cmix.NewSalt(csprng.Source(&csprng.SystemRNG{}), 16)

	// TBD: Add key macs to this message
	macs := make([][]byte, 0)

	// Generate a compound encryption key
	encryptionKey := cyclic.NewInt(1)
	for _, key := range user.TheSession.GetKeys() {
		baseKey := key.TransmissionKeys.Base
		partialEncryptionKey := cmix.NewEncryptionKey(salt, baseKey, crypto.Grp)
		crypto.Grp.Mul(encryptionKey, partialEncryptionKey, encryptionKey)
		//TODO: Add KMAC generation here
	}

	// TBD: Is there a really good reason we have to specify the Grp and not a
	// key? Should we even be doing the encryption here?
	// TODO: Use salt here
	encryptedMessage := crypto.Encrypt(encryptionKey, crypto.Grp, message)
	msgPacket := &pb.CmixMessage{
		SenderID:       senderID.Bytes(),
		MessagePayload: encryptedMessage.Payload.Bytes(),
		RecipientID:    encryptedMessage.Recipient.Bytes(),
		Salt:           salt,
		KMACs:          macs,
	}

	var err error
	globals.Log.INFO.Println("Sending put message to gateway")
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
		globals.Log.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientPollMessage{
		UserID: user.TheSession.GetCurrentUser().UserID.Bytes(),
	}

	for {
		time.Sleep(delay)
		globals.Log.INFO.Printf("Attempting to receive message from gateway")
		decryptedMessages := m.receiveMessagesFromGateway(&pollingMessage)
		if decryptedMessages != nil {
			for i := range decryptedMessages {
				assembledMessage := GetCollator().AddMessage(
					decryptedMessages[i], time.Minute)
				if assembledMessage != nil {
					// we got a fully assembled message. let's broadcast it
					broadcastMessageReception(assembledMessage, switchboard.Listeners)
				}
			}
		}
	}
}

func (m *messaging) receiveMessagesFromGateway(
	pollingMessage *pb.ClientPollMessage) []*format.Message {
	if user.TheSession != nil {
		user.TheSession.LockStorage()
		defer user.TheSession.UnlockStorage()
		pollingMessage.MessageID = user.TheSession.GetLastMessageID()
		messages, err := client.SendCheckMessages(user.TheSession.GetGWAddress(),
			pollingMessage)

		if err != nil {
			globals.Log.WARN.Printf("CheckMessages error during polling: %v", err.Error())
			return nil
		}

		globals.Log.INFO.Printf("Checking novelty of %v messages", len(messages.MessageIDs))

		if ReceivedMessages == nil {
			ReceivedMessages = make(map[string]struct{})
		}

		results := make([]*format.Message, 0, len(messages.MessageIDs))
		for _, messageID := range messages.MessageIDs {
			// Get the first unseen message from the list of IDs
			_, received := ReceivedMessages[messageID]
			if !received {
				globals.Log.INFO.Printf("Got a message waiting on the gateway: %v",
					messageID)
				// We haven't seen this message before.
				// So, we should retrieve it from the gateway.
				newMessage, err := client.SendGetMessage(user.
					TheSession.GetGWAddress(),
					&pb.ClientPollMessage{
						UserID:    user.TheSession.GetCurrentUser().UserID.Bytes(),
						MessageID: messageID,
					})
				if err != nil {
					globals.Log.WARN.Printf(
						"Couldn't receive message with ID %v while"+
							" polling gateway", messageID)
				} else {
					if newMessage.MessagePayload == nil &&
						newMessage.RecipientID == nil &&
						newMessage.SenderID == nil {
						globals.Log.INFO.Println("Message fields not populated")
						continue
					}

					// Generate a compound decryption key
					salt := newMessage.Salt
					decryptionKey := cyclic.NewInt(1)
					for _, key := range user.TheSession.GetKeys() {
						baseKey := key.ReceptionKeys.Base
						partialDecryptionKey := cmix.NewDecryptionKey(salt, baseKey,
							crypto.Grp)
						crypto.Grp.Mul(decryptionKey, partialDecryptionKey, decryptionKey)
						//TODO: Add KMAC verification here
					}

					globals.Log.INFO.Printf(
						"Adding message ID %v to received message IDs", messageID)
					ReceivedMessages[messageID] = struct{}{}
					user.TheSession.SetLastMessageID(messageID)

					decryptedMsg, err2 := crypto.Decrypt(decryptionKey, crypto.Grp,
						newMessage)
					if err2 != nil {
						globals.Log.WARN.Printf(
							"Message did not decrypt properly, " +
								"not adding to results array: %v", err2.Error())
					} else {
						results = append(results, decryptedMsg)
					}
				}
			}
		}
		return results
	}
	return nil
}

func broadcastMessageReception(message *parse.Message,
	listeners *switchboard.Switchboard) {

	listeners.Speak(message)
}
