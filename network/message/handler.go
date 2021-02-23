///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/crypto/e2e"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
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
				m.handleMessage(msg, bundle.Identity)
			}
			bundle.Finish()
		}
	}

}

func (m *Manager) handleMessage(ecrMsg format.Message, identity reception.IdentityUse) {
	// We've done all the networking, now process the message
	fingerprint := ecrMsg.GetKeyFP()

	e2eKv := m.Session.E2e()

	var sender *id.ID
	var msg format.Message
	var encTy message.EncryptionType
	var err error
	var relationshipFingerprint []byte

	//check if the identity fingerprint matches
	forMe, err := fingerprint2.CheckIdentityFP(ecrMsg.GetIdentityFP(),
		ecrMsg.GetContents(), identity.Source)
	if err != nil {
		jww.FATAL.Panicf("Could not check IdentityFIngerprint: %+v", err)
	}
	if !forMe {
		return
	}

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
		msg = ecrMsg
		if err != nil {
			jww.DEBUG.Printf("Failed to unmarshal ephemeral ID "+
				"on unknown message: %+v", err)
		}
		raw := message.Receive{
			Payload:     msg.Marshal(),
			MessageType: message.Raw,
			Sender:      &id.ID{},
			EphemeralID: identity.EphId,
			Timestamp:   time.Time{},
			Encryption:  message.None,
			RecipientID: identity.Source,
		}
		jww.INFO.Printf("Garbled/RAW Message: %v", msg.GetKeyFP())
		m.Session.GetGarbledMessages().Add(msg)
		m.Switchboard.Speak(raw)
		return
	}

	jww.INFO.Printf("Received message of type %s from %s," +
		" msgDigest: %s", encTy, sender, msg.Digest())

	// Process the decrypted/unencrypted message partition, to see if
	// we get a full message
	xxMsg, ok := m.partitioner.HandlePartition(sender, encTy, msg.GetContents(),
		relationshipFingerprint)

	//Set the identities
	xxMsg.RecipientID = identity.Source
	xxMsg.EphemeralID = identity.EphId

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
