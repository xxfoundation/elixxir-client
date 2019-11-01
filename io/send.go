package io

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// SendMessage to the provided Recipient
// TODO: It's not clear why we wouldn't hand off a sender object (with
// the keys) here. I won't touch crypto at this time, though...
// TODO This method would be cleaner if it took a parse.Message (particularly
// w.r.t. generating message IDs for multi-part messages.)
func (cm *CommManager) SendMessage(session user.Session, topology *circuit.Circuit,
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
		cm.nextId())
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
	cm.lock.RLock()
	transmitGateway := id.NewNodeFromBytes(cm.ndf.Nodes[cm.transmissionGatewayIndex].ID).NewGateway()
	for i := range parts {
		message := format.NewMessage()
		message.SetRecipient(recipientID)
		message.SetTimestamp(extendedNowBytes)
		message.Contents.SetRightAligned(parts[i])
		err = cm.send(session, topology, cryptoType, message, false, transmitGateway)
		if err != nil {
			return errors.Wrap( err,"SendMessage send() error:")
		}
	}
	cm.lock.RUnlock()
	return nil
}

// Send Message without doing partitions
// This function will be needed for example to send a Rekey
// message, where a new public key will take up the whole message
func (cm *CommManager) SendMessageNoPartition(session user.Session,
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

	cm.lock.RLock()
	transmitGateway := id.NewNodeFromBytes(cm.ndf.Nodes[cm.transmissionGatewayIndex].ID).NewGateway()
	cm.lock.RUnlock()

	err = cm.send(session, topology, cryptoType, msg, true, transmitGateway)
	if err != nil {
		return fmt.Errorf("SendMessageNoPartition send() error: %v", err.Error())
	}
	return nil
}

// send actually sends the message to the server
func (cm *CommManager) send(session user.Session, topology *circuit.Circuit,
	cryptoType parse.CryptoType,
	message *format.Message,
	rekey bool, transmitGateway *id.Gateway) error {
	// Enable transmission blocking if enabled
	if cm.blockTransmissions {
		cm.sendLock.Lock()
		defer func() {
			time.Sleep(cm.transmitDelay)
			cm.sendLock.Unlock()
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
		message.SetKeyFP(*format.NewFingerprint(session.GetCurrentUser().User.Bytes()))
	}
	// CMIX Encryption
	salt := cmix.NewSalt(csprng.Source(&csprng.SystemRNG{}), 32)
	encMsg, kmacs := crypto.CMIXEncrypt(session, topology, salt, message)

	msgPacket := &pb.Slot{
		SenderID: session.GetCurrentUser().User.Bytes(),
		PayloadA: encMsg.GetPayloadA(),
		PayloadB: encMsg.GetPayloadB(),
		Salt:     salt,
		KMACs:    kmacs,
	}

	return cm.Comms.SendPutMessage(transmitGateway, msgPacket)
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
		partners := session.GetKeyStore().GetPartners()
		globals.Log.INFO.Printf("Valid Partner IDs: %+v", partners)
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
