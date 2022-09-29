////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/fileTransfer/sentRoundTracker"
	"gitlab.com/elixxir/client/fileTransfer/store"
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"time"
)

// Error messages.
const (
	// generateRandomPacketSize
	getRandomNumPartsRandPanic = "[FT] Failed to generate random number of file parts to send: %+v"

	// manager.sendCmix
	errNoMoreRetries = "file transfer failed: ran our of retries."
)

const (
	// Duration to wait for round to finish before timing out.
	roundResultsTimeout = 15 * time.Second

	// Age when rounds that files were sent from are deleted from the tracker.
	clearSentRoundsAge = 10 * time.Second

	// Number of concurrent sending threads
	workerPoolThreads = 4

	// Tag that prints with cMix sending logs.
	cMixDebugTag = "FT.Part"

	// Prefix used for the name of a stoppable used for a sending thread
	sendThreadStoppableName = "FilePartSendingThread#"
)

// startSendingWorkerPool initialises a worker pool of file part sending
// threads.
func (m *manager) startSendingWorkerPool(multiStop *stoppable.Multi) {
	// Set up cMix sending parameters
	m.params.Cmix.SendTimeout = m.params.SendTimeout
	m.params.Cmix.ExcludedRounds =
		sentRoundTracker.NewManager(clearSentRoundsAge)

	if m.params.Cmix.DebugTag == cmix.DefaultDebugTag ||
		m.params.Cmix.DebugTag == "" {
		m.params.Cmix.DebugTag = cMixDebugTag
	}

	for i := 0; i < workerPoolThreads; i++ {
		stop := stoppable.NewSingle(sendThreadStoppableName + strconv.Itoa(i))
		go func(single *stoppable.Single) {
			m.sendingThread(single)
		}(stop)
		jww.INFO.Printf("Adding stoppable %s", stop.Name())
		multiStop.Add(stop)
	}

}

// sendingThread sends part packets that become available oin the send queue.
func (m *manager) sendingThread(stop *stoppable.Single) {
	healthChan := make(chan bool, 10)
	healthChanID := m.cmix.AddHealthCallback(func(b bool) { healthChan <- b })
	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Printf("[FT] Stopping file part sending thread (%s): "+
				"stoppable triggered.", stop.Name())
			m.cmix.RemoveHealthCallback(healthChanID)
			stop.ToStopped()
			return
		case healthy := <-healthChan:
			for !healthy {
				jww.INFO.Printf("not healthy, waiting for health update")
				select {
				// Wait for health update or a quit signal
				case <-stop.Quit():
					jww.DEBUG.Printf("[FT] Stopping file part sending thread (%s): "+
						"stoppable triggered.", stop.Name())
					m.cmix.RemoveHealthCallback(healthChanID)
					stop.ToStopped()
					return
				case healthy = <-healthChan:
					jww.INFO.Printf("received health update, it is now set to %s", healthy)

				}
			}
		case packet := <-m.sendQueue:
			jww.INFO.Printf("sending pack")
			m.sendCmix(packet)
		}
	}
}

// sendCmix sends the parts in the packet via Cmix.SendMany.
func (m *manager) sendCmix(packet []store.Part) {
	// validParts will contain all parts in the original packet excluding those
	// that return an error from GetEncryptedPart
	validParts := make([]store.Part, 0, len(packet))

	// Encrypt each part and to a TargetedCmixMessage
	messages := make([]cmix.TargetedCmixMessage, 0, len(packet))
	for _, p := range packet {
		encryptedPart, mac, fp, err :=
			p.GetEncryptedPart(m.cmix.GetMaxMessageLength())
		if err != nil {
			jww.ERROR.Printf("[FT] File transfer %s (%q) failed: %+v",
				p.TransferID(), p.FileName(), err)
			m.callbacks.Call(p.TransferID(), errors.New(errNoMoreRetries))
			continue
		}

		validParts = append(validParts, p)

		messages = append(messages, cmix.TargetedCmixMessage{
			Recipient:   p.Recipient(),
			Payload:     encryptedPart,
			Fingerprint: fp,
			Service:     message.Service{},
			Mac:         mac,
		})
	}

	// Clear all old rounds from the sent rounds list
	m.params.Cmix.ExcludedRounds.(*sentRoundTracker.Manager).RemoveOldRounds()

	jww.DEBUG.Printf("[FT] Sending %d file parts via SendManyCMIX",
		len(messages))

	rid, _, err := m.cmix.SendMany(messages, m.params.Cmix)
	if err != nil {
		jww.WARN.Printf("[FT] Failed to send %d file parts via "+
			"SendManyCMIX: %+v", len(messages), err)

		for _, p := range validParts {
			m.batchQueue <- p
		}
	}

	err = m.cmix.GetRoundResults(
		roundResultsTimeout, m.roundResultsCallback(validParts), rid)
}

// roundResultsCallback generates a network.RoundEventCallback that handles
// all parts in the packet once the round succeeds or fails.
func (m *manager) roundResultsCallback(packet []store.Part) cmix.RoundEventCallback {
	// Group file parts by transfer
	grouped := map[ftCrypto.TransferID][]store.Part{}
	for _, p := range packet {
		if _, exists := grouped[*p.TransferID()]; exists {
			grouped[*p.TransferID()] = append(grouped[*p.TransferID()], p)
		} else {
			grouped[*p.TransferID()] = []store.Part{p}
		}
	}

	return func(
		allRoundsSucceeded, _ bool, rounds map[id.Round]cmix.RoundResult) {
		// Get round ID
		var rid id.Round
		for rid = range rounds {
			break
		}

		if allRoundsSucceeded {
			jww.DEBUG.Printf("[FT] %d file parts delivered on round %d (%v)",
				len(packet), rid, grouped)

			// If the round succeeded, then mark all parts as arrived and report
			// each transfer's progress on its progress callback
			for tid, parts := range grouped {
				for _, p := range parts {
					p.MarkArrived()
				}

				// Call the progress callback after all parts have been marked
				// so that the progress reported included all parts in the batch
				m.callbacks.Call(&tid, nil)
			}
		} else {
			jww.DEBUG.Printf("[FT] %d file parts failed on round %d (%v)",
				len(packet), rid, grouped)

			// If the round failed, then add each part into the send queue
			for _, p := range packet {
				m.batchQueue <- p
			}
		}
	}
}
