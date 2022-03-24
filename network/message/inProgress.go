///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// Messages can arrive in the network out of order. When message handling fails
// to decrypt a message, it is added to the garbled message buffer (which is
// stored on disk) and the message decryption is retried here whenever triggered.

// This can be triggered through the CheckInProgressMessages on the network
// handler and is used in the /keyExchange package on successful rekey triggering.

// CheckInProgressMessages triggers rechecking all in progress messages if the
// queue is not full Exposed on the network handler.
func (p *handler) CheckInProgressMessages() {
	select {
	case p.checkInProgress <- struct{}{}:
	default:
		jww.WARN.Print("Failed to check garbled messages due to full channel.")
	}
}

// recheckInProgressRunner is a long-running thread which processes messages
// that need to be checked.
func (p *handler) recheckInProgressRunner(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case <-p.checkInProgress:
			jww.INFO.Printf("[GARBLE] Checking Garbled messages")
			p.recheckInProgress()
		}
	}
}

// recheckInProgress is the handler for a single run of recheck messages.
func (p *handler) recheckInProgress() {
	// Try to decrypt every garbled message, excising those whose counts are too
	// high
	for grbldMsg, ri, identity, has := p.inProcess.Next(); has; grbldMsg, ri, identity, has = p.inProcess.Next() {
		bundle := Bundle{
			Round:     id.Round(ri.ID),
			RoundInfo: ri,
			Messages:  []format.Message{grbldMsg},
			Finish:    func() {},
			Identity:  identity,
		}

		select {
		case p.messageReception <- bundle:
		default:
			jww.WARN.Printf("Failed to send bundle, channel full.")
		}
	}
}
