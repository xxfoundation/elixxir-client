////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io asynchronous sending functionality. This is managed by an outgoing
// messages channel and managed by the sender thread kicked off during
// initialization.
package io

import (
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"sync"
	"time"
)

type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// Messaging implements the Communications interface
type Messaging struct {
	nextId   func() []byte
	collator *Collator
	// SendAddress is the address of the server to send messages
	SendGateway *id.Gateway
	// ReceiveAddress is the address of the server to receive messages from
	ReceiveGateway *id.Gateway
	// BlockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	BlockTransmissions bool
	// TransmitDelay is the minimum delay between transmissions.
	TransmitDelay time.Duration
	// Map that holds a record of the messages that this client successfully
	// received during this session
	ReceivedMessages map[string]struct{}
	// Comms pointer to send/recv messages
	Comms    *client.ClientComms
	sendLock sync.Mutex
}

func NewMessenger() *Messaging {
	return &Messaging{
		nextId:             parse.IDCounter(),
		collator:           NewCollator(),
		BlockTransmissions: true,
		TransmitDelay:      1000 * time.Millisecond,
		ReceivedMessages:   make(map[string]struct{}),
		Comms:              &client.ClientComms{},
	}
}

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
// TODO This method would be cleaner if it took a parse.Message (particularly
// w.r.t. generating message IDs for multi-part messages.)
func (m *Messaging) SendMessage(session user.Session, topology *circuit.Circuit,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	// FIXME: We should really bring the plaintext parts of the NewMessage logic
	// into this module, then have an EncryptedMessage type that is sent to/from
	// the cMix network. This NewMessage does way too many things: break the
	// message into parts, generate mic's, etc -- the crypto library should only
	// know about the crypto and advertise a max message payload size

	// TBD: Is there a really good reason why we'd ever have more than one user
	// in this library? why not pass a sender object instead?
	globals.Log.DEBUG.Printf("Sending message to %q: %q", *recipientID, message)
	parts, err := parse.Partition([]byte(message),
		m.nextId())
	if err != nil {
		return fmt.Errorf("SendMessage Partition() error: %v", err.Error())
	}
	// Every part should have the same timestamp
	now := time.Now()
	// GO Timestamp binary serialization is 15 bytes, which
	// allows the encrypted timestamp to fit in 16 bytes
	// using AES encryption
	// The timestamp will be encrypted later
	// NOTE: This sets 15 bytes, not 16
	nowBytes, err := now.MarshalBinary()
	if err != nil {
		return fmt.Errorf("SendMessage MarshalBinary() error: %v", err.Error())
	}
	// Add a byte for later encryption (15->16 bytes)
	extendedNowBytes := append(nowBytes, 0)
	for i := range parts {
		message := format.NewMessage()
		message.SetRecipient(recipientID)
		message.SetTimestamp(extendedNowBytes)
		message.Contents.SetRightAligned(parts[i])
		err = m.send(session, topology, cryptoType, message, false)
		if err != nil {
			return fmt.Errorf("SendMessage send() error: %v", err.Error())
		}
	}
	return nil
}

// Send Message without doing partitions
// This function will be needed for example to send a Rekey
// message, where a new public key will take up the whole message
func (m *Messaging) SendMessageNoPartition(session user.Session,
	topology *circuit.Circuit, recipientID *id.User, cryptoType parse.CryptoType,
	message []byte) error {
	size := len(message)
	if size > format.TotalLen {
		return fmt.Errorf("SendMessageNoPartition() error: message to be sent is too big")
	}
	now := time.Now()
	// GO Timestamp binary serialization is 15 bytes, which
	// allows the encrypted timestamp to fit in 16 bytes
	// using AES encryption
	// The timestamp will be encrypted later
	// NOTE: This sets 15 bytes, not 16
	nowBytes, err := now.MarshalBinary()
	if err != nil {
		return fmt.Errorf("SendMessageNoPartition MarshalBinary() error: %v", err.Error())
	}
	msg := format.NewMessage()
	msg.SetRecipient(recipientID)
	// Add a byte to support later encryption (15 -> 16 bytes)
	nowBytes = append(nowBytes, 0)
	msg.SetTimestamp(nowBytes)
	msg.Contents.Set(message)
	globals.Log.DEBUG.Printf("Sending message to %v: %x", *recipientID, message)
	err = m.send(session, topology, cryptoType, msg, true)
	if err != nil {
		return fmt.Errorf("SendMessageNoPartition send() error: %v", err.Error())
	}
	return nil
}

// send actually sends the message to the server
func (m *Messaging) send(session user.Session, topology *circuit.Circuit,
	cryptoType parse.CryptoType,
	message *format.Message,
	rekey bool) error {
	// Enable transmission blocking if enabled
	if m.BlockTransmissions {
		m.sendLock.Lock()
		defer func() {
			time.Sleep(m.TransmitDelay)
			m.sendLock.Unlock()
		}()
	}

	// Check message type
	if cryptoType == parse.E2E {
		handleE2ESending(session, message, rekey)
	} else {
		padded, err := e2e.Pad(message.Contents.GetRightAligned(), format.ContentsLen)
		if err != nil {
			return err
		}
		message.Contents.Set(padded)
		e2e.SetUnencrypted(message)
		message.SetMAC(session.GetCurrentUser().User.Bytes())
	}
	// CMIX Encryption
	salt := cmix.NewSalt(csprng.Source(&csprng.SystemRNG{}), 32)
	encMsg := crypto.CMIXEncrypt(session, topology, salt, message)

	msgPacket := &pb.Slot{
		SenderID:       session.GetCurrentUser().User.Bytes(),
		MessagePayload: encMsg.GetPayloadA(),
		AssociatedData: encMsg.GetPayloadB(),
		Salt:           salt,
		KMACs:          make([][]byte, 0),
	}

	return m.Comms.SendPutMessage(m.SendGateway, msgPacket)
}

func handleE2ESending(session user.Session,
	message *format.Message,
	rekey bool) {
	recipientID := message.GetRecipient()

	var key *keyStore.E2EKey
	var action keyStore.Action
	// Get KeyManager for this partner
	km := session.GetKeyStore().GetSendManager(recipientID)
	if km == nil {
		globals.Log.FATAL.Panicf("Couldn't get KeyManager to E2E encrypt message to"+
			" user %v", *recipientID)
	}

	// FIXME: This is a hack to prevent a crash, this function should be
	//        able to block until this condition is true.
	for end, timeout := false, time.After(60*time.Second); !end; {
		if rekey {
			// Get send Rekey
			key, action = km.PopRekey()
		} else {
			// Get send key
			key, action = km.PopKey()
		}
		if key != nil {
			end = true
		}

		select {
		case <-timeout:
			end = true
		default:
		}
	}

	if key == nil {
		globals.Log.FATAL.Panicf("Couldn't get key to E2E encrypt message to"+
			" user %v", *recipientID)
	} else if action == keyStore.Purge {
		// Destroy this key manager
		km := key.GetManager()
		km.Destroy(session.GetKeyStore())
		globals.Log.WARN.Printf("Destroying E2E Send Keys Manager for partner: %v", *recipientID)
	} else if action == keyStore.Deleted {
		globals.Log.FATAL.Panicf("Key Manager is deleted when trying to get E2E Send Key")
	}

	if action == keyStore.Rekey {
		// Send RekeyTrigger message to switchboard
		rekeyMsg := &parse.Message{
			Sender: session.GetCurrentUser().User,
			TypedBody: parse.TypedBody{
				MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
				Body:        []byte{},
			},
			InferredType: parse.None,
			Receiver:     recipientID,
		}
		go session.GetSwitchboard().Speak(rekeyMsg)
	}

	globals.Log.DEBUG.Printf("E2E encrypting message")
	if rekey {
		crypto.E2EEncryptUnsafe(session.GetE2EGroup(),
			key.GetKey(),
			key.KeyFingerprint(),
			message)
	} else {
		crypto.E2EEncrypt(session.GetE2EGroup(),
			key.GetKey(),
			key.KeyFingerprint(),
			message)
	}
}

// MessageReceiver is a polling thread for receiving messages -- again.. we
// should be passing this a user object with some keys, and maybe a shared
// list for the listeners?
// Accessing all of these global variables is extremely problematic for this
// kind of thread.
func (m *Messaging) MessageReceiver(session user.Session, delay time.Duration) {
	// FIXME: It's not clear we should be doing decryption here.
	if session == nil {
		globals.Log.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientRequest{
		UserID: session.GetCurrentUser().User.Bytes(),
	}

	for {
		select {
		case <-session.GetQuitChan():
			close(session.GetQuitChan())
			return
		default:
			time.Sleep(delay)
			globals.Log.DEBUG.Printf("Attempting to receive message from gateway")
			decryptedMessages := m.receiveMessagesFromGateway(session, &pollingMessage)
			if decryptedMessages != nil {
				for i := range decryptedMessages {
					// TODO Handle messages that do not need partitioning
					assembledMessage := m.collator.AddMessage(
						decryptedMessages[i], time.Minute)
					if assembledMessage != nil {
						// we got a fully assembled message. let's broadcast it
						broadcastMessageReception(assembledMessage, session.GetSwitchboard())
					}
				}
			}
		}
	}
}

func handleE2EReceiving(session user.Session,
	message *format.Message) (bool, error) {
	keyFingerprint := message.GetKeyFP()

	// Lookup reception key
	recpKey := session.GetKeyStore().
		GetRecvKey(keyFingerprint)

	rekey := false
	if recpKey == nil {
		// TODO Handle sending error message to SW
		return false, fmt.Errorf("E2EKey for matching fingerprint not found, can't process message")
	} else if recpKey.GetOuterType() == parse.Rekey {
		// If key type is rekey, the message is a rekey from partner
		rekey = true
	}

	globals.Log.DEBUG.Printf("E2E decrypting message")
	var err error
	if rekey {
		err = crypto.E2EDecryptUnsafe(session.GetE2EGroup(), recpKey.GetKey(), message)
	} else {
		err = crypto.E2EDecrypt(session.GetE2EGroup(), recpKey.GetKey(), message)
	}

	if err != nil {
		// TODO handle Garbled message to SW
	}

	// Get partner from Key Manager of receiving key
	// since there is no space in message for senderID
	// Get decrypted partner public key from message
	// Send rekey message to switchboard
	if rekey {
		partner := recpKey.GetManager().GetPartner()
		partnerPubKey := message.Contents.Get()
		rekeyMsg := &parse.Message{
			Sender: partner,
			TypedBody: parse.TypedBody{
				MessageType: int32(cmixproto.Type_NO_TYPE),
				Body:        partnerPubKey,
			},
			InferredType: parse.Rekey,
			Receiver:     session.GetCurrentUser().User,
		}
		go session.GetSwitchboard().Speak(rekeyMsg)
	}
	return rekey, err
}

func (m *Messaging) receiveMessagesFromGateway(session user.Session,
	pollingMessage *pb.ClientRequest) []*format.Message {
	if session != nil {
		pollingMessage.LastMessageID = session.GetLastMessageID()
		messages, err := m.Comms.SendCheckMessages(m.ReceiveGateway,
			pollingMessage)

		if err != nil {
			globals.Log.WARN.Printf("CheckMessages error during polling: %v", err.Error())
			return nil
		}

		globals.Log.INFO.Printf("Checking novelty of %v messages", len(messages.IDs))

		results := make([]*format.Message, 0, len(messages.IDs))
		for _, messageID := range messages.IDs {
			// Get the first unseen message from the list of IDs
			_, received := m.ReceivedMessages[messageID]
			if !received {
				globals.Log.INFO.Printf("Got a message waiting on the gateway: %v",
					messageID)
				// We haven't seen this message before.
				// So, we should retrieve it from the gateway.
				newMessage, err := m.Comms.SendGetMessage(
					m.ReceiveGateway,
					&pb.ClientRequest{
						UserID: session.GetCurrentUser().User.
							Bytes(),
						LastMessageID: messageID,
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

					msg := format.NewMessage()
					msg.SetPayloadA(newMessage.MessagePayload)
					msg.SetDecryptedPayloadB(newMessage.AssociatedData)
					var err error = nil
					var rekey bool
					var unpadded []byte
					// If message is E2E, handle decryption
					if !e2e.IsUnencrypted(msg) {
						rekey, err = handleE2EReceiving(session, msg)
					} else {
						// If message is non E2E, need to unpad payload
						unpadded, err = e2e.Unpad(msg.Contents.Get())
						if err == nil {
							msg.Contents.SetRightAligned(unpadded)
						}
					}

					if err != nil {
						globals.Log.WARN.Printf(
							"Message did not decrypt properly, "+
								"not adding to results array: %v", err.Error())
					} else if rekey {
						globals.Log.INFO.Printf("Correctly processed rekey message," +
							" not adding to results array")
					} else {
						results = append(results, msg)
					}

					globals.Log.INFO.Printf(
						"Adding message ID %v to received message IDs", messageID)
					m.ReceivedMessages[messageID] = struct{}{}
					session.SetLastMessageID(messageID)
					err = session.StoreSession()
					if err != nil {
						globals.Log.ERROR.Printf("Could not store session "+
							"after message recieved from gateway: %+v", err)
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
