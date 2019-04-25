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
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"sync"
	"time"
)

// Messaging implements the Communications interface
type Messaging struct {
	nextId   func() []byte
	collator *Collator
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
	sendLock         sync.Mutex
}

func NewMessenger() *Messaging {
	return &Messaging{
		nextId:             parse.IDCounter(),
		collator:           NewCollator(),
		BlockTransmissions: true,
		TransmitDelay:      1000 * time.Millisecond,
		ReceivedMessages:   make(map[string]struct{}),
	}
}

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
// TODO This method would be cleaner if it took a parse.Message (particularly
// w.r.t. generating message IDs for multi-part messages.)
func (m *Messaging) SendMessage(session user.Session,
	recipientID *id.User,
	cryptoType format.CryptoType,
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
	nowBytes, err := now.MarshalBinary()
	if err != nil {
		return fmt.Errorf("SendMessage MarshalBinary() error: %v", err.Error())
	}
	for i := range parts {
		message := format.NewMessage()
		message.SetSender(userID)
		message.SetRecipient(recipientID)
		// The timestamp will be encrypted later
		// NOTE: This sets 15 bytes, not 16
		message.SetTimestamp(nowBytes)
		message.SetPayloadData(parts[i])
		err = m.send(session, cryptoType, message, false)
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
	recipientID *id.User,
	cryptoType format.CryptoType,
	message []byte) error {
	size := len(message)
	if size > format.TOTAL_LEN {
		return fmt.Errorf("SendMessageNoPartition() error: message to be sent is too big")
	}
	userID := session.GetCurrentUser().User
	now := time.Now()
	// GO Timestamp binary serialization is 15 bytes, which
	// allows the encrypted timestamp to fit in 16 bytes
	// using AES encryption
	nowBytes, err := now.MarshalBinary()
	if err != nil {
		return fmt.Errorf("SendMessageNoPartition MarshalBinary() error: %v", err.Error())
	}
	msg := format.NewMessage()
	msg.SetRecipient(recipientID)
	// The timestamp will be encrypted later
	// NOTE: This sets 15 bytes, not 16
	msg.SetTimestamp(nowBytes)
	// If message is bigger than payload size
	// use SenderID space to send it
	if size > format.MP_PAYLOAD_LEN {
		msg.SetSenderID(message[:format.MP_SID_END])
		msg.SetPayloadData(message[format.MP_SID_END:])
	} else {
		msg.SetSender(userID)
		msg.SetPayloadData(message)
	}
	err = m.send(session, cryptoType, msg, true)
	if err != nil {
		return fmt.Errorf("SendMessageNoPartition send() error: %v", err.Error())
	}
	return nil
}

// send actually sends the message to the server
func (m *Messaging) send(session user.Session,
	cryptoType format.CryptoType,
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
	if cryptoType == format.E2E {
		handleE2ESending(session, message, rekey)
	} else {
		padded, err := e2e.Pad(message.GetPayload(), format.TOTAL_LEN)
		if err != nil {
			return err
		}
		message.SetPayload(padded)
		e2e.SetUnencrypted(message)
	}

	// CMIX Encryption
	salt := cmix.NewSalt(csprng.Source(&csprng.SystemRNG{}), 16)
	encMsg := crypto.CMIXEncrypt(session, salt, message)

	msgPacket := &pb.CmixMessage{
		SenderID:       session.GetCurrentUser().User.Bytes(),
		MessagePayload: encMsg.SerializePayload(),
		AssociatedData: encMsg.SerializeAssociatedData(),
		Salt:           salt,
		KMACs:          make([][]byte, 0),
	}

	globals.Log.INFO.Println("Sending put message to gateway")
	return client.SendPutMessage(m.SendAddress, msgPacket)
}

func handleE2ESending(session user.Session,
	message *format.Message,
	rekey bool) {
	recipientID := message.GetRecipient()

	var key *keyStore.E2EKey
	var action keyStore.Action
	if rekey {
		// Get send Rekey
		key, action = session.GetKeyStore().
			TransmissionReKeys.Pop(recipientID)
	} else {
		// Get send key
		key, action = session.GetKeyStore().
			TransmissionKeys.Pop(recipientID)
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
		// Send RekeyTrigger message to switchboard containing partner public key
		// The most recent partner public key will be the one on the
		// Receiving Key Manager for this partner
		km := session.GetRecvKeyManager(recipientID)
		partnerPubKey := km.GetPubKey()
		rekeyMsg := &parse.Message{
			Sender: session.GetCurrentUser().User,
			TypedBody: parse.TypedBody{
				MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
				Body:        partnerPubKey.Bytes(),
			},
			CryptoType: format.None,
			Receiver:   recipientID,
		}
		session.GetSwitchboard().Speak(rekeyMsg)
	}

	globals.Log.DEBUG.Printf("E2E encrypting message")
	if rekey {
		crypto.E2EEncryptUnsafe(session.GetGroup(),
			key.GetKey(),
			key.KeyFingerprint(),
			message)
	} else {
		crypto.E2EEncrypt(session.GetGroup(),
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
	pollingMessage := pb.ClientPollMessage{
		UserID: session.GetCurrentUser().User.Bytes(),
	}

	for {
		select {
		case <-session.GetQuitChan():
			close(session.GetQuitChan())
			return
		default:
			time.Sleep(delay)
			globals.Log.INFO.Printf("Attempting to receive message from gateway")
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
	keyFingerprint := message.GetKeyFingerprint()

	// Lookup reception key
	recpKey := session.GetKeyStore().
		ReceptionKeys.Pop(keyFingerprint)

	rekey := false
	if recpKey == nil {
		// TODO Handle sending error message to SW
		return rekey, fmt.Errorf("E2EKey for matching fingerprint not found, can't process message")
	} else if recpKey.GetOuterType() == format.Rekey {
		// If key type is rekey, the message is a rekey from partner
		rekey = true
	}

	globals.Log.DEBUG.Printf("E2E decrypting message")
	var err error
	if rekey {
		err = crypto.E2EDecryptUnsafe(session.GetGroup(), recpKey.GetKey(), message)
	} else {
		err = crypto.E2EDecrypt(session.GetGroup(), recpKey.GetKey(), message)
	}

	if err != nil {
		// TODO handle Garbled message to SW
	}

	// Get decrypted partner public key from message
	// The most recent own private key will be the one on the
	// Sending Key Manager for this partner
	// Send rekey message to switchboard
	if rekey {
		partner := recpKey.GetManager().GetPartner()
		km := session.GetSendKeyManager(partner)
		ownPrivKey := km.GetPrivKey().LeftpadBytes(uint64(format.TOTAL_LEN))
		partnerPubKey := message.SerializePayload()
		body := append(ownPrivKey, partnerPubKey...)
		rekeyMsg := &parse.Message{
			Sender: partner,
			TypedBody: parse.TypedBody{
				MessageType: int32(cmixproto.Type_NO_TYPE),
				Body:        body,
			},
			CryptoType: format.Rekey,
			Receiver:   session.GetCurrentUser().User,
		}
		session.GetSwitchboard().Speak(rekeyMsg)
	}
	return rekey, err
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

					// CMIX Decryption
					decMsg := crypto.CMIXDecrypt(session, newMessage)

					var err error = nil
					var rekey bool
					var unpadded []byte
					// If message is E2E, handle decryption
					if !e2e.IsUnencrypted(decMsg) {
						rekey, err = handleE2EReceiving(session, decMsg)
					} else {
						// If message is non E2E, need to unpad payload
						unpadded, err = e2e.Unpad(decMsg.SerializePayload())
						if err == nil {
							decMsg.SetSplitPayload(unpadded)
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
						results = append(results, decMsg)
					}

					globals.Log.INFO.Printf(
						"Adding message ID %v to received message IDs", messageID)
					m.ReceivedMessages[messageID] = struct{}{}
					session.SetLastMessageID(messageID)
					session.StoreSession()
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
