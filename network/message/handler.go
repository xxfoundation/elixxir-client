package message

import (
	"bytes"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

func (m *Manager) handleMessages(quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case bundle := <-m.messageReception:
			for _, msg := range bundle.Messages {
				m.handleMessage(msg)
			}
			bundle.Finish()
		}
	}

}

func (m *Manager) handleMessage(ecrMsg format.Message) {
	// We've done all the networking, now process the message
	fingerprint := ecrMsg.GetKeyFP()

	e2eKv := m.Session.E2e()

	var sender *id.ID
	var msg format.Message
	var encTy message.EncryptionType
	var err error
	var relationshipFingerprint []byte

	// try to get the key fingerprint, process as e2e encryption if
	// the fingerprint is found
	if key, isE2E := e2eKv.PopKey(fingerprint); isE2E {
		jww.INFO.Printf("is e2e message")
		// Decrypt encrypted message
		msg, err = key.Decrypt(ecrMsg)
		// get the sender
		sender = key.GetSession().GetPartner()
		relationshipFingerprint = key.GetSession().GetRelationshipFingerprint()

		//drop the message is decryption failed
		if err != nil {
			//if decryption failed, print an error
			jww.WARN.Printf("Failed to decrypt message with fp %s "+
				"from partner %s: %s", key.Fingerprint(), sender, err)
			return
		}
		//set the type as E2E encrypted
		encTy = message.E2E
	} else if isUnencrypted, uSender := IsUnencrypted(ecrMsg); isUnencrypted {
		jww.INFO.Printf("is unencrypted")
		// if the key fingerprint does not match, try to treat it as an
		// unencrypted message
		sender = uSender
		msg = ecrMsg
		encTy = message.None
	} else {
		jww.INFO.Printf("is raw")
		// if it doesnt match any form of encrypted, hear it as a raw message
		// and add it to garbled messages to be handled later
		msg = ecrMsg
		raw := message.Receive{
			Payload:     msg.Marshal(),
			MessageType: message.Raw,
			Sender:      msg.GetRecipientID(),
			Timestamp:   time.Time{},
			Encryption:  message.None,
		}
		jww.INFO.Printf("Garbled/RAW Message: %v", msg.GetKeyFP())
		m.Session.GetGarbledMessages().Add(msg)
		m.Switchboard.Speak(raw)
		return
	}

	// Process the decrypted/unencrypted message partition, to see if
	// we get a full message
	xxMsg, ok := m.partitioner.HandlePartition(sender, encTy, msg.GetContents(),
		relationshipFingerprint)
	// If the reception completed a message, hear it on the switchboard
	if ok {
		if xxMsg.MessageType == message.Raw {
			jww.WARN.Panicf("Recieved a message of type 'Raw' from %s."+
				"Message Ignored, 'Raw' is a reserved type. Message supressed.",
				xxMsg.ID)
		} else {
			m.Switchboard.Speak(xxMsg)
		}
	}
}

const macMask = 0b00111111

// IsUnencrypted determines if the message is unencrypted by comparing the hash
// of the message payload to the MAC. Returns true if the message is unencrypted
// and false otherwise.
// the highest bit of the recpient ID is stored in the highest bit of the MAC
// field. This is accounted for and the id is reassembled, with a presumed user
// type
func IsUnencrypted(m format.Message) (bool, *id.ID) {

	expectedMac := makeUnencryptedMAC(m.GetContents())
	receivedMac := m.GetMac()
	idHighBit := (receivedMac[0] & 0b01000000) << 1
	receivedMac[0] &= macMask

	//return false if the message is not unencrypted
	if !bytes.Equal(expectedMac, receivedMac) {
		jww.INFO.Printf("Failed isUnencrypted! Expected: %v; " +
			" Received: %v", expectedMac, receivedMac)
		return false, nil
	}

	//extract the user ID
	idBytes := m.GetKeyFP()
	idBytes[0] |= idHighBit
	uid := id.ID{}
	copy(uid[:], idBytes[:])
	uid.SetType(id.User)

	// Return true if the byte slices are equal
	return true, &uid
}

// SetUnencrypted sets up the condition where the message would be determined to
// be unencrypted by setting the MAC to the hash of the message payload.
func SetUnencrypted(m format.Message, uid *id.ID) {
	mac := makeUnencryptedMAC(m.GetContents())

	//copy in the high bit of the userID for storage
	mac[0] |= (uid[0] & 0b10000000) >> 1

	// Set the MAC
	m.SetMac(mac)

	//remove the type byte off of the userID and clear the highest bit so
	//it can be stored in the fingerprint
	fp := format.Fingerprint{}
	copy(fp[:], uid[:format.KeyFPLen])
	fp[0] &= 0b01111111

	m.SetKeyFP(fp)
}

// returns the mac, fingerprint, and the highest byte
func makeUnencryptedMAC(payload []byte)[]byte{
	// Create new hash
	h, err := hash.NewCMixHash()

	if err != nil {
		jww.ERROR.Panicf("Failed to create hash: %v", err)
	}

	// Hash the message payload
	h.Write(payload)
	payloadHash := h.Sum(nil)

	//set the first bit as zero to ensure everything stays in the group
	payloadHash[0] &= macMask

	return payloadHash
}