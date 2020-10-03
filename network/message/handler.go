package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/crypto/e2e"
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
	} else if isUnencrypted, uSender := e2e.IsUnencrypted(ecrMsg); isUnencrypted {
		// if the key fingerprint does not match, try to treat it as an
		// unencrypted message
		sender = uSender
		msg = ecrMsg
		encTy = message.None
	} else {
		// if it doesnt match any form of encrypted, hear it as a raw message
		// and add it to garbled messages to be handled later
		raw := message.Receive{
			Payload:     msg.GetRawContents(),
			MessageType: message.Raw,
			Sender:      &id.ID{},
			Timestamp:   time.Time{},
			Encryption:  message.None,
		}
		m.Switchboard.Speak(raw)
		m.Session.GetGarbledMessages().Add(msg)
		return
	}

	// Process the decrypted/unencrypted message partition, to see if
	// we get a full message
	xxMsg, ok := m.partitioner.HandlePartition(sender, encTy, msg.GetContents(),
		relationshipFingerprint)
	// If the reception completed a message, hear it on the switchboard
	if ok {
		m.Switchboard.Speak(xxMsg)
	}
}
