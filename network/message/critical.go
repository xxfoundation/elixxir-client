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
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Critical Messages are protocol layer communications that must succeed. These
// are added to the persistent critical messages store.  This thread waits for
// network access to move from unhealthy to healthy and the sends all critical
// messages.
// Health is tracked by registering with the Health
// Tracker (/network/Health/Tracker.g0)

//Thread loop for processing critical messages
func (m *Manager) processCriticalMessages(quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case isHealthy := <-m.networkIsHealthy:
			if isHealthy {
				m.criticalMessages()
			}
		}
	}
}

// processes all critical messages
func (m *Manager) criticalMessages() {
	critMsgs := m.Session.GetCriticalMessages()
	// try to send every message in the critical messages and the raw critical
	// messages buffer in parallel

	//critical messages
	for msg, param, has := critMsgs.Next(); has; msg, param, has = critMsgs.Next() {
		go func(msg message.Send, param params.E2E) {
			jww.INFO.Printf("Resending critical message to %s ",
				msg.Recipient)
			//send the message
			rounds, _, err := m.SendE2E(msg, param)
			//if the message fail to send, notify the buffer so it can be handled
			//in the future and exit
			if err != nil {
				jww.ERROR.Printf("Failed to send critical message to %s "+
					" on notification of healthy network: %+v", msg.Recipient,
					err)
				critMsgs.Failed(msg)
				return
			}
			//wait on the results to make sure the rounds were successful
			sendResults := make(chan ds.EventReturn, len(rounds))
			roundEvents := m.Instance.GetRoundEvents()
			for _, r := range rounds {
				roundEvents.AddRoundEventChan(r, sendResults, 1*time.Minute,
					states.COMPLETED, states.FAILED)
			}
			success, numTimeOut, numRoundFail := utility.TrackResults(sendResults, len(rounds))
			if !success {
				jww.ERROR.Printf("critical message send to %s failed "+
					"to transmit transmit %v/%v paritions on rounds %d: %v "+
					"round failures, %v timeouts", msg.Recipient,
					numRoundFail+numTimeOut, len(rounds), rounds, numRoundFail, numTimeOut)
				critMsgs.Failed(msg)
				return
			}

			jww.INFO.Printf("Sucesfull resend of critical message "+
				"to %s on rounds %d", msg.Recipient, rounds)
			critMsgs.Succeeded(msg)
		}(msg, param)
	}

	critRawMsgs := m.Session.GetCriticalRawMessages()
	param := params.GetDefaultCMIX()
	//raw critical messages
	for msg, rid, has := critRawMsgs.Next(); has; msg, rid, has = critRawMsgs.Next() {
		localRid := rid.DeepCopy()
		go func(msg format.Message, rid *id.ID) {
			jww.INFO.Printf("Resending critical raw message to %s "+
				"(msgDigest: %s)", rid, msg.Digest())
			//send the message
			round, _, err := m.SendCMIX(msg, rid, param)
			//if the message fail to send, notify the buffer so it can be handled
			//in the future and exit
			if err != nil {
				jww.ERROR.Printf("Failed to send critical raw message on "+
					"notification of healthy network: %+v", err)
				critRawMsgs.Failed(msg, rid)
				return
			}

			//wait on the results to make sure the rounds were successful
			sendResults := make(chan ds.EventReturn, 1)
			roundEvents := m.Instance.GetRoundEvents()

			roundEvents.AddRoundEventChan(round, sendResults, 1*time.Minute,
				states.COMPLETED, states.FAILED)

			success, numTimeOut, _ := utility.TrackResults(sendResults, 1)
			if !success {
				if numTimeOut > 0 {
					jww.ERROR.Printf("critical raw message resend to %s "+
						"(msgDigest: %s) on round %d failed to transmit due to "+
						"timeout", rid, msg.Digest(), round)
				} else {
					jww.ERROR.Printf("critical raw message resend to %s "+
						"(msgDigest: %s) on round %d failed to transmit due to "+
						"send failure", rid, msg.Digest(), round)
				}

				critRawMsgs.Failed(msg, rid)
				return
			}

			jww.INFO.Printf("Sucesfull resend of critical raw message "+
				"to %s (msgDigest: %s) on round %d", rid, msg.Digest(), round)

			critRawMsgs.Succeeded(msg, rid)
		}(msg, localRid)
	}

}
