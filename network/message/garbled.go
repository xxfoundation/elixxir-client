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
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// Messages can arrive in the network out of order. When message handling fails
// to decrypt a message, it is added to the garbled message buffer (which is
// stored on disk) and the message decryption is retried here whenever triggered.

// This can be triggered through the CheckGarbledMessages on the network manager
// and is used in the /keyExchange package on successful rekey triggering

// Triggers Garbled message checking if the queue is not full
// Exposed on the network manager
func (m *Manager) CheckGarbledMessages() {
	select {
	case m.triggerGarbled <- struct{}{}:
	default:
		jww.WARN.Println("Failed to check garbled messages " +
			"due to full channel")
	}
}

//long running thread which processes garbled messages
func (m *Manager) processGarbledMessages(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-m.triggerGarbled:
			jww.INFO.Printf("[GARBLE] Checking Garbled messages")
			m.handleGarbledMessages()
		}
	}
}

//handler for a single run of garbled messages
func (m *Manager) handleGarbledMessages() {
	garbledMsgs := m.Session.GetGarbledMessages()
	e2eKv := m.Session.E2e()
	var failedMsgs []format.Message
	//try to decrypt every garbled message, excising those who's counts are too high
	for grbldMsg, count, timestamp, has := garbledMsgs.Next(); has; grbldMsg, count, timestamp, has = garbledMsgs.Next() {
		//if it exists, check against all in the list
		grbldContents := grbldMsg.GetContents()
		identity := m.Session.GetUser().ReceptionID
		_, forMe, _ := m.Session.GetEdge().Check(identity, grbldMsg.GetIdentityFP(), grbldContents)
		if forMe {
			fingerprint := grbldMsg.GetKeyFP()
			// Check if the key is there, process it if it is
			if key, isE2E := e2eKv.PopKey(fingerprint); isE2E {
				jww.INFO.Printf("[GARBLE] Check E2E for %s, KEYFP: %s",
					grbldMsg.Digest(), grbldMsg.GetKeyFP())
				// Decrypt encrypted message
				msg, err := key.Decrypt(grbldMsg)
				if err == nil {
					// get the sender
					sender := key.GetSession().GetPartner()
					//remove from the buffer if decryption is successful
					garbledMsgs.Remove(grbldMsg)

					jww.INFO.Printf("[GARBLE] message decoded as E2E from "+
						"%s, msgDigest: %s", sender, grbldMsg.Digest())

					//handle the successfully decrypted message
					xxMsg, ok := m.partitioner.HandlePartition(sender, message.E2E,
						msg.GetContents(),
						key.GetSession().GetRelationshipFingerprint())
					if ok {
						m.Switchboard.Speak(xxMsg)
						continue
					}
				}
			} else {
				// todo: figure out how to get the ephermal reception id in here.
				// we have the raw data, but do not know what address space was
				// used int he round
				// todo: figure out how to get the round id, the recipient id, and the round timestamp
				/*
					ephid, err := ephemeral.Marshal(garbledMsg.GetEphemeralRID())
					if err!=nil{
						jww.WARN.Printf("failed to get the ephemeral id for a garbled " +
							"message, clearing the message: %+v", err)
						garbledMsgs.Remove(garbledMsg)
						continue
					}

					ephid.Clear(m.)*/

				raw := message.Receive{
					Payload:        grbldMsg.Marshal(),
					MessageType:    message.Raw,
					Sender:         &id.ID{},
					EphemeralID:    ephemeral.Id{},
					Timestamp:      time.Unix(0, 0),
					Encryption:     message.None,
					RecipientID:    &id.ID{},
					RoundId:        0,
					RoundTimestamp: time.Unix(0, 0),
				}
				im := fmt.Sprintf("[GARBLE] RAW Message reprocessed: keyFP: %v, "+
					"msgDigest: %s", grbldMsg.GetKeyFP(), grbldMsg.Digest())
				jww.INFO.Print(im)
				m.Internal.Events.Report(1, "MessageReception", "Garbled", im)
				m.Session.GetGarbledMessages().Add(grbldMsg)
				m.Switchboard.Speak(raw)
			}
		}

		// fail the message if any part of the decryption fails,
		// unless it is the last attempts and has been in the buffer long
		// enough, in which case remove it
		if count == m.param.MaxChecksGarbledMessage &&
			netTime.Since(timestamp) > m.param.GarbledMessageWait {
			garbledMsgs.Remove(grbldMsg)
		} else {
			failedMsgs = append(failedMsgs, grbldMsg)
		}
	}
	for _, grbldMsg := range failedMsgs {
		garbledMsgs.Failed(grbldMsg)
	}
}
