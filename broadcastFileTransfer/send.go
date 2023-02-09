////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/sentRoundTracker"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
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
		go m.sendingThread(stop)
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
				healthy = <-healthChan
			}
		case packet := <-m.sendQueue:
			m.sendCmix(packet)
		}
	}
}

// sendCmix sends the parts in the packet via Cmix.SendMany.
func (m *manager) sendCmix(packet []*store.Part) {
	// validParts will contain all parts in the original packet excluding those
	// that return an error from GetEncryptedPart
	validParts := make([]*store.Part, 0, len(packet))

	// Encrypt each part and to a TargetedCmixMessage
	messages := make([]cmix.TargetedCmixMessage, 0, len(packet))
	for _, p := range packet {
		encryptedPart, mac, fp, err :=
			p.GetEncryptedPart(m.cmix.GetMaxMessageLength())
		if err != nil {
			jww.ERROR.Printf("[FT] File transfer %s (%q) failed: %+v",
				p.FileID(), p.FileName(), err)
			m.callbacks.Call(p.FileID(), errors.New(errNoMoreRetries))
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

	jww.DEBUG.Printf("[FT] Sending %d file parts via SendManyCMIX: %s",
		len(messages), validParts)

	rid, _, err := m.cmix.SendMany(messages, m.params.Cmix)
	if err != nil {
		jww.WARN.Printf("[FT] Failed to send %d file parts via "+
			"SendManyCMIX (%s): %+v", len(messages), validParts, err)

		for _, p := range validParts {
			m.batchQueue <- p
		}
	}

	m.cmix.GetRoundResults(
		roundResultsTimeout, m.roundResultsCallback(validParts), rid.ID)
}

func printGrouped(g map[ftCrypto.ID][]*store.Part) string {
	transfers := make([]string, 0, len(g))
	for fid, parts := range g {
		partNums := make([]string, len(parts))
		for i, part := range parts {
			partNums[i] = strconv.Itoa(int(part.PartNum()))
		}
		transfers = append(
			transfers, fid.String()+":["+strings.Join(partNums, ", ")+"]")
	}

	return strings.Join(transfers, " ")
}

// roundResultsCallback generates a network.RoundEventCallback that handles
// all parts in the packet once the round succeeds or fails.
func (m *manager) roundResultsCallback(
	packet []*store.Part) cmix.RoundEventCallback {
	// Group file parts by transfer
	grouped := map[ftCrypto.ID][]*store.Part{}
	for _, p := range packet {
		if _, exists := grouped[p.FileID()]; exists {
			grouped[p.FileID()] = append(grouped[p.FileID()], p)
		} else {
			grouped[p.FileID()] = []*store.Part{p}
		}
	}

	return func(
		allRoundsSucceeded, _ bool, rounds map[id.Round]cmix.RoundResult) {
		// Get round ID
		var rid id.Round
		for rid = range rounds {
			break
		}

		// Get send completion time
		var sendTimestamp time.Time
		for _, r := range rounds {
			sendTimestamp = r.Round.Timestamps[states.COMPLETED]
			break
		}

		if allRoundsSucceeded {
			jww.DEBUG.Printf("[FT] %d file parts sent on round %d (%s)",
				len(packet), rid, printGrouped(grouped))

			// If the round succeeded, then mark all parts as arrived and report
			// each transfer's progress on its progress callback
			for fid, parts := range grouped {
				for _, p := range parts {
					p.MarkSent()
				}

				// Call the progress callback after all parts have been marked
				// so that the progress reported included all parts in the batch
				m.callbacks.Call(fid, nil)
			}

			m.sentQueue <- &sentPartPacket{packet, sendTimestamp, false}
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

// resendUnreceived checks that all successfully sent file have been received
// after a set amount of times. Any unsent file parts are added back to the
// queue for sending.
func (m *manager) resendUnreceived(stop *stoppable.Multi) {
	jww.DEBUG.Printf("[FT] Starting resend unreceived file part thread.")
	mainThreadStop := stoppable.NewSingle("FilePartResendThread")
	stop.Add(mainThreadStop)

	for {
		select {
		case <-mainThreadStop.Quit():
			jww.DEBUG.Printf("[FT] Stopping file part resend thread (%s): "+
				"stoppable triggered.", stop.Name())
			mainThreadStop.ToStopped()
			return
		case sentMessages := <-m.sentQueue:
			jww.DEBUG.Printf("[FT] Received sentMessages: %+v", sentMessages)
			sendQueueStop := stoppable.NewSingle("FilePartResendBatch")
			stop.Add(sendQueueStop)

			go func(stop *stoppable.Single, sm *sentPartPacket) {
				waitTime := calcWaitTime(m.params.ResendWait, netTime.Now(), sm)

				jww.DEBUG.Printf("[FT] Scheduled check for resend for %d "+
					"parts in %s: %v", len(sm.packet), waitTime, sm.packet)
				select {
				case <-stop.Quit():
					jww.DEBUG.Printf("[FT] Stopping file part batch resend "+
						"thread (%s): stoppable triggered.", stop.Name())
					stop.ToStopped()
					return
				case <-time.After(waitTime):
					var resentParts []*store.Part
					for _, p := range sm.packet {
						status := p.GetStatus()
						if status == store.SentPart {
							m.batchQueue <- p
							resentParts = append(resentParts, p)
						}
					}
					if len(resentParts) > 0 {
						jww.DEBUG.Printf("[FT] %d of %d parts not received; "+
							"resending parts: %v",
							len(resentParts), len(sm.packet), resentParts)
					} else {
						jww.DEBUG.Printf("[FT] %d out of %d parts received.",
							len(sm.packet), len(sm.packet))
					}
				}
			}(sendQueueStop, sentMessages)
		}
	}
}

// calcWaitTime calculates the amount of time to wait before checking if the
// parts have been received.
func calcWaitTime(
	delay time.Duration, timeNow time.Time, spp *sentPartPacket) time.Duration {
	if spp.loaded {
		return delay
	}

	timeSinceSend := timeNow.Sub(spp.sentTime)
	if timeSinceSend >= delay {
		return 0
	}

	return delay - timeSinceSend
}

// checkedReceivedParts returns a ReceivedProgressCallback that when passed to
// the received transfer handler, tracks which file parts have been received and
// updates their status in the store.SentTransfer. It also updates the sent
// progress callback on updates and clears the file from memory when the
// transfer completed.
func (m *manager) checkedReceivedParts(st *store.SentTransfer, ti *TransferInfo,
	completeCB SendCompleteCallback) ReceivedProgressCallback {
	return func(
		_ bool, _, _ uint16, rt ReceivedTransfer, t FilePartTracker, err error) {
		// Propagate the error to the sent progress callback
		if err != nil {
			m.callbacks.Call(st.FileID(), err)
			return
		}

		// Mark each received part as received in the SentTransfer
		var partsChanges []uint16
		for _, p := range st.GetUnsentParts() {
			if t.GetPartStatus(p.PartNum()) == FpReceived {
				p.MarkReceived()
				partsChanges = append(partsChanges, p.PartNum())
			}
		}
		for _, p := range st.GetSentParts() {
			if t.GetPartStatus(p.PartNum()) == FpReceived {
				p.MarkReceived()
				partsChanges = append(partsChanges, p.PartNum())
			}
		}

		// Call the progress callback if any parts were updated
		if len(partsChanges) > 0 {
			jww.DEBUG.Printf(
				"[FT] %d file parts set as received for file %s (%d)",
				len(partsChanges), st.FileID(), partsChanges)
			m.callbacks.Call(st.FileID(), nil)
		}

		// Once the transfer is complete, close out both the sent and received
		// sides of the transfer
		if st.Status() == store.Completed {
			jww.DEBUG.Printf("[FT] Completed sending and receiving file %s.",
				st.FileID())
			if err = m.CloseSend(st.FileID()); err != nil {
				jww.ERROR.Printf("Failed to close file transfer send: %+v", err)
			}
			if _, err = m.Receive(rt.FileID()); err != nil {
				jww.ERROR.Printf("Failed to receive file transfer: %+v", err)
			}

			go completeCB(*ti)
		}
	}
}
