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
	"fmt"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"time"
)

// Messaging implements the Communications interface
type Messaging struct {
	nextId func() []byte
	// SendAddress is the address of the server to send messages
	SendAddress string
	// ReceiveAddress is the address of the server to receive messages from
	ReceiveAddress string
	// BlockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	BlockTransmissions bool
	// TransmitDelay is the minimum delay between transmissions.
	TransmitDelay time.Duration
	// Map that holds a record of the messages that this client successfully
	// received during this session
	ReceivedMessages map[string]struct{}
	sendLock sync.Mutex
}

func NewMessenger() *Messaging {
	return &Messaging{
		nextId: parse.IDCounter(),
		BlockTransmissions: true,
		TransmitDelay: 1000 * time.Millisecond,
		ReceivedMessages: make(map[string]struct{}),
	}
}

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
// TODO This method would be cleaner if it took a parse.Message (particularly
// w.r.t. generating message IDs for multi-part messages.)
func (m *Messaging) SendMessage(session user.Session,
	recipientID *id.User,
	message []byte) error {
	// FIXME: We should really bring the plaintext parts of the NewMessage logic
	// into this module, then have an EncryptedMessage type that is sent to/from
	// the cMix network. This NewMessage does way too many things: break the
	// message into parts, generate mic's, etc -- the crypto library should only
	// know about the crypto and advertise a max message payload size

	// TBD: Is there a really good reason why we'd ever have more than one user
	// in this library? why not pass a sender object instead?
	globals.Log.DEBUG.Printf("Sending message to %q: %q", *recipientID, message)
	userID := session.GetCurrentUser().User
	grp := session.GetGroup()
	parts, err := parse.Partition([]byte(message),
		m.nextId())
	if err != nil {
		return fmt.Errorf("SendMessage Partition() error: %v", err.Error())
	}
	// Every part should have the same timestamp
	now := time.Now()
	// TODO Is it better to use Golang's binary timestamp format, or
	// use the 2 int64 technique with Unix seconds+nanoseconds?
	// 2 int64s is 128 bits, which is as much as can fit in the timestamp field,
	// but the binary serialization is 15 bytes, which is slightly smaller but
	// not smaller enough to make a difference.
	// The binary serialized timestamp also includes zone data, which could be
	// a feature, but might compromise a little bit of anonymity.
	// Using binary timestamp format for now.
	// TODO BC: It is actually better to use the 15 byte version since this will
	// allow the encrypted timestamp to fit in 16 bytes instead of 32, by using
	// the key fingerprint as the IV for AES encryption
	nowBytes, err := now.MarshalBinary()
	if err != nil {
		return fmt.Errorf("SendMessage MarshalBinary() error: %v", err.Error())
	}
	for i := range parts {
		message := format.NewMessage()
		message.SetSender(userID)
		message.SetRecipient(recipientID)
		// The timestamp will be encrypted later
		message.SetTimestamp(nowBytes)
		message.SetPayloadData(parts[i])
		err = m.send(session, userID, message, grp)
		if err != nil {
			return fmt.Errorf("SendMessage send() error: %v", err.Error())
		}
	}
	return nil
}

// send actually sends the message to the server
func (m *Messaging) send(session user.Session,
	senderID *id.User, message *format.Message, grp *cyclic.Group) error {
	// Enable transmission blocking if enabled
	if m.BlockTransmissions {
		m.sendLock.Lock()
		defer func() {
			time.Sleep(m.TransmitDelay)
			m.sendLock.Unlock()
		}()
	}

	salt := cmix.NewSalt(csprng.Source(&csprng.SystemRNG{}), 16)

	// TBD: Add key macs to this message
	macs := make([][]byte, 0)

	// Generate a compound encryption key
	encryptionKey := grp.NewInt(1)
	for _, key := range session.GetKeys() {
		baseKey := key.TransmissionKeys.Base
		partialEncryptionKey := cmix.NewEncryptionKey(salt, baseKey, grp)
		grp.Mul(encryptionKey, partialEncryptionKey, encryptionKey)
		//TODO: Add KMAC generation here
	}

	// TBD: Is there a really good reason we have to specify the Grp and not a
	// key? Should we even be doing the encryption here?
	// TODO: Use salt here / generate n key map
	e2eKey := e2e.Keygen(grp, nil, nil)
	associatedData, payload := crypto.Encrypt(encryptionKey, grp,
		message, e2eKey)
	msgPacket := &pb.CmixMessage{
		SenderID:       senderID.Bytes(),
		MessagePayload: payload,
		AssociatedData: associatedData,
		Salt:           salt,
		KMACs:          macs,
	}

	var err error
	globals.Log.INFO.Println("Sending put message to gateway")
	err = client.SendPutMessage(m.SendAddress, msgPacket)

	return err
}

// MessageReceiver is a polling thread for receiving messages -- again.. we
// should be passing this a user object with some keys, and maybe a shared
// list for the listeners?
// Accessing all of these global variables is extremely problematic for this
// kind of thread.
func (m *Messaging) MessageReceiver(session user.Session,
	sw *switchboard.Switchboard,
	delay time.Duration, quit chan bool) {
	// FIXME: It's not clear we should be doing decryption here.
	if session == nil {
		globals.Log.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientPollMessage{
		UserID: session.GetCurrentUser().User.Bytes(),
	}

	for {
		select {
		case <-quit:
			close(quit)
			return
		default:
			time.Sleep(delay)
			globals.Log.INFO.Printf("Attempting to receive message from gateway")
			decryptedMessages := m.receiveMessagesFromGateway(session, &pollingMessage)
			if decryptedMessages != nil {
				for i := range decryptedMessages {
					assembledMessage := GetCollator().AddMessage(
						decryptedMessages[i], time.Minute)
					if assembledMessage != nil {
						// we got a fully assembled message. let's broadcast it
						broadcastMessageReception(assembledMessage, sw)
					}
				}
			}
		}
	}
}

func (m *Messaging) receiveMessagesFromGateway(session user.Session,
	pollingMessage *pb.ClientPollMessage) []*format.Message {
	if session != nil {
		pollingMessage.MessageID = session.GetLastMessageID()
		messages, err := client.SendCheckMessages(session.GetGWAddress(),
			pollingMessage)

		if err != nil {
			globals.Log.WARN.Printf("CheckMessages error during polling: %v", err.Error())
			return nil
		}

		globals.Log.INFO.Printf("Checking novelty of %v messages", len(messages.MessageIDs))

		results := make([]*format.Message, 0, len(messages.MessageIDs))
		grp := session.GetGroup()
		for _, messageID := range messages.MessageIDs {
			// Get the first unseen message from the list of IDs
			_, received := m.ReceivedMessages[messageID]
			if !received {
				globals.Log.INFO.Printf("Got a message waiting on the gateway: %v",
					messageID)
				// We haven't seen this message before.
				// So, we should retrieve it from the gateway.
				newMessage, err := client.SendGetMessage(
					session.GetGWAddress(),
					&pb.ClientPollMessage{
						UserID: session.GetCurrentUser().User.
							Bytes(),
						MessageID: messageID,
					})
				if err != nil {
					globals.Log.WARN.Printf(
						"Couldn't receive message with ID %v while"+
							" polling gateway", messageID)
				} else {
					if newMessage.MessagePayload == nil ||
						newMessage.AssociatedData == nil {
						globals.Log.INFO.Println("Message fields not populated")
						continue
					}

					// Generate a compound decryption key
					salt := newMessage.Salt
					decryptionKey := grp.NewInt(1)
					for _, key := range session.GetKeys() {
						baseKey := key.ReceptionKeys.Base
						partialDecryptionKey := cmix.NewDecryptionKey(salt, baseKey,
							grp)
						grp.Mul(decryptionKey, partialDecryptionKey, decryptionKey)
						//TODO: Add KMAC verification here
					}

					globals.Log.INFO.Printf(
						"Adding message ID %v to received message IDs", messageID)
					m.ReceivedMessages[messageID] = struct{}{}
					session.SetLastMessageID(messageID)
					session.StoreSession()

					decryptedMsg, err2 := crypto.Decrypt(decryptionKey, grp,
						newMessage)
					if err2 != nil {
						globals.Log.WARN.Printf(
							"Message did not decrypt properly, "+
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
