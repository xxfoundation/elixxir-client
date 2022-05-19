///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/crypto/e2e"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

func (m *Manager) handleMessages(stop *stoppable.Single) {
	preimageList := m.Session.GetEdge()
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case bundle := <-m.messageReception:
			for _, msg := range bundle.Messages {
				m.handleMessage(msg, bundle, preimageList)
			}
			bundle.Finish()
		}
	}

}

func (m *Manager) handleMessage(ecrMsg format.Message, bundle Bundle, edge *edge.Store) {
	// We've done all the networking, now process the message
	fingerprint := ecrMsg.GetKeyFP()
	msgDigest := ecrMsg.Digest()
	identity := bundle.Identity

	e2eKv := m.Session.E2e()

	var sender *id.ID
	var msg format.Message
	var encTy message.EncryptionType
	var err error
	var relationshipFingerprint []byte

	//if it exists, check against all in the list
	ecrMsgContents := ecrMsg.GetContents()
	has, forMe, _ := m.Session.GetEdge().Check(identity.Source, ecrMsg.GetIdentityFP(), ecrMsgContents)
	if !has {
		jww.INFO.Printf("checking backup %v", preimage.MakeDefault(identity.Source))
		//if it doesnt exist, check against the default fingerprint for the identity
		forMe = fingerprint2.CheckIdentityFP(ecrMsg.GetIdentityFP(),
			ecrMsgContents, preimage.MakeDefault(identity.Source))
	}

	if !forMe {
		if jww.GetLogThreshold() == jww.LevelTrace {
			expectedFP := fingerprint2.IdentityFP(ecrMsgContents,
				preimage.MakeDefault(identity.Source))
			jww.TRACE.Printf("Message for %d (%s) failed identity "+
				"check: %v (expected-default) vs %v (received)", identity.EphId,
				identity.Source, expectedFP, ecrMsg.GetIdentityFP())
		}
		im := fmt.Sprintf("Garbled/RAW Message: keyFP: %v, round: %d"+
			"msgDigest: %s, not determined to be for client", ecrMsg.GetKeyFP(), bundle.Round, ecrMsg.Digest())
		m.Internal.Events.Report(1, "MessageReception", "Garbled", im)
		m.Session.GetGarbledMessages().Add(ecrMsg)
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
			msg := fmt.Sprintf("Failed to decrypt message with "+
				"fp %s from partner %s: %s", key.Fingerprint(),
				sender, err)
			jww.WARN.Printf(msg)
			m.Internal.Events.Report(9, "MessageReception",
				"DecryptionError", msg)
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
		// if it doesn't match any form of encrypted, hear it as a raw message
		// and add it to garbled messages to be handled later
		msg = ecrMsg
		raw := message.Receive{
			Payload:     msg.Marshal(),
			MessageType: message.Raw,
			Sender:      &id.ID{},
			EphemeralID: identity.EphId,
			Timestamp:   time.Time{},
			Encryption:  message.None,
			RecipientID: identity.Source,
			RoundId:     id.Round(bundle.RoundInfo.ID),
			// We use PRECOMPUTING here because all Rounds have that timestamp available to them
			// QUEUED can be missing sometimes and cause a lot of hidden problems further down the line
			RoundTimestamp: time.Unix(0, int64(bundle.RoundInfo.Timestamps[states.PRECOMPUTING])),
		}
		im := fmt.Sprintf("Received message of type Garbled/RAW: keyFP: %v, round: %d, "+
			"msgDigest: %s", msg.GetKeyFP(), bundle.Round, msg.Digest())
		jww.INFO.Print(im)
		m.Internal.Events.Report(1, "MessageReception", "Garbled", im)
		m.Session.GetGarbledMessages().Add(msg)
		m.Switchboard.Speak(raw)
		return
	}

	// Process the decrypted/unencrypted message partition, to see if
	// we get a full message
	xxMsg, ok := m.partitioner.HandlePartition(sender, encTy, msg.GetContents(),
		relationshipFingerprint)

	im := fmt.Sprintf("Received message of ecr type %s and msg type "+
		"%d from %s in round %d,msgDigest: %s, keyFP: %v", encTy,
		xxMsg.MessageType, sender, bundle.Round, msgDigest, msg.GetKeyFP())
	jww.INFO.Print(im)
	m.Internal.Events.Report(2, "MessageReception", "MessagePart", im)

	// If the reception completed a message, hear it on the switchboard
	if ok {
		//Set the identities
		xxMsg.RecipientID = identity.Source
		xxMsg.EphemeralID = identity.EphId
		xxMsg.Encryption = encTy
		xxMsg.RoundId = id.Round(bundle.RoundInfo.ID)
		xxMsg.RoundTimestamp = time.Unix(0, int64(bundle.RoundInfo.Timestamps[states.QUEUED]))
		if xxMsg.MessageType == message.Raw {
			rm := fmt.Sprintf("Received a message of type 'Raw' from %s."+
				"Message Ignored, 'Raw' is a reserved type. Message supressed.",
				xxMsg.ID)
			jww.WARN.Print(rm)
			m.Internal.Events.Report(10, "MessageReception",
				"Error", rm)
		} else {
			m.Switchboard.Speak(xxMsg)
		}
	}
}
