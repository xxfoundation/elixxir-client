////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/client/v5/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// Messages can arrive in the network out of order. When message handling fails
// to decrypt a message, it is added to the garbled message buffer (which is
// stored on disk) and the message decryption is retried here whenever
// triggered.

// This can be triggered through the CheckInProgressMessages on the network
// handler and is used in the /keyExchange package on successful rekey
// triggering.

// CheckInProgressMessages triggers rechecking all in progress messages if the
// queue is not full Exposed on the network handler.
func (h *handler) CheckInProgressMessages() {
	select {
	case h.checkInProgress <- struct{}{}:
		jww.DEBUG.Print("[Garbled] Sent signal to check garbled " +
			"message queue...")
	default:
		jww.WARN.Print("Failed to check garbled messages due to full channel.")
	}
}

// recheckInProgressRunner is a long-running thread which processes messages
// that need to be checked.
func (h *handler) recheckInProgressRunner(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-h.checkInProgress:
			jww.INFO.Printf("[GARBLE] Checking Garbled messages")
			h.recheckInProgress()
		}
	}
}

// recheckInProgress is the handler for a single run of recheck messages.
func (h *handler) recheckInProgress() {
	// Try to decrypt every garbled message, excising those whose counts are too
	// high
	for grbldMsg, ri, identity, has := h.inProcess.Next(); has; grbldMsg, ri, identity, has = h.inProcess.Next() {
		bundleMsgs := []format.Message{grbldMsg}
		bundle := Bundle{
			Round:     id.Round(ri.ID),
			RoundInfo: rounds.MakeRound(ri),
			Messages:  bundleMsgs,
			Finish:    func() {},
			Identity:  identity,
		}

		select {
		case h.messageReception <- bundle:
			jww.INFO.Printf("[GARBLE] Sent %d messages to process",
				len(bundleMsgs))
		default:
			jww.WARN.Printf("Failed to send bundle, channel full.")
		}
	}
}
