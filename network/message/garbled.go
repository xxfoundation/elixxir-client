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
	//try to decrypt every garbled message, excising those who's counts are too high
	for grbldMsg, count, timestamp, has := m.garbledStore.Next(); has; grbldMsg, count, timestamp, has = m.garbledStore.Next() {
		//if it exists, check against all in the list
		grbldContents := grbldMsg.GetContents()
		identity := m.session.GetUser().ReceptionID

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
}
