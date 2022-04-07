////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// Error messages.
const (
	// Manager.sendParts
	sendManyCmixWarn   = "[FT] Failed to send %d file parts %v via SendManyCMIX: %+v"
	setInProgressErr   = "[FT] Failed to set parts %v to in-progress for transfer %s: %+v"
	getRoundResultsErr = "[FT] Failed to get round results for round %d for file transfers %v: %+v"

	// Manager.buildMessages
	noSentTransferWarn = "[FT] Could not get transfer %s for part %d: %+v"
	maxRetriesErr      = "Stopping message transfer: %+v"
	newCmixMessageErr  = "[FT] Failed to assemble cMix message for file part %d on transfer %s: %+v"

	// Manager.makeRoundEventCallback
	finishPassNoTransferErr = "[FT] Failed to mark in-progress parts as finished on success of round %d for transfer %s: %+v"
	finishTransferErr       = "[FT] Failed to set part(s) to finished for transfer %s: %+v"
	finishedEndE2eMsfErr    = "[FT] Failed to send E2E message to %s on completion of file transfer %s: %+v"
	roundFailureWarn        = "[FT] Failed to send file parts for file transfers %v on round %d: round %s"
	finishFailNoTransferErr = "[FT] Failed to requeue in-progress parts on failure of round %d for transfer %s: %+v"
	unsetInProgressErr      = "[FT] Failed to remove parts from in-progress list for transfer %s: round %s: %+v"

	// Manager.sendEndE2eMessage
	endE2eGetPartnerErr = "failed to get file transfer partner %s: %+v"
	endE2eHealthTimeout = "waiting for network to become healthy timed out after %s."
	endE2eSendErr       = "failed to send end file transfer message via E2E to recipient %s: %+v"

	// getRandomNumParts
	getRandomNumPartsRandPanic = "[FT] Failed to generate random number of file parts to send: %+v"
)

const (
	// Duration to wait for round to finish before timing out.
	roundResultsTimeout = 15 * time.Second

	// Duration to wait for send batch to fill before sending partial batch.
	pollSleepDuration = 100 * time.Millisecond

	// Age when rounds that files were sent from are deleted from the tracker.
	clearSentRoundsAge = 10 * time.Second

	// Duration to wait for network to become healthy to send end E2E message
	// before timing out.
	sendEndE2eHealthTimeout = 5 * time.Second

	// Tag that prints with cMix sending logs.
	cMixDebugTag = "FT.Part"
)

// sendThread waits on the sendQueue channel for parts to send. Once its
// receives a random number between 1 and 11 of file parts, they are encrypted,
// put into cMix messages, and sent to their recipients. Failed messages are
// added to the end of the queue.
func (m *Manager) sendThread(stop *stoppable.Single, healthChan chan bool,
	healthChanID uint64, getNumParts getRngNum) {
	jww.DEBUG.Print("[FT] Starting file part sending thread.")

	// Calculate the average amount of data sent via SendManyCMIX
	avgNumMessages := (minPartsSendPerRound + maxPartsSendPerRound) / 2
	avgSendSize := avgNumMessages * (8192 / 8)

	// Calculate the delay needed to reach max throughput
	delay := time.Duration((int(time.Second) * avgSendSize) / m.p.MaxThroughput)

	// Batch of parts read from the queue to be sent
	var partList []queuedPart

	// Create new sent round tracker that tracks which recent rounds file parts
	// were sent on so that they can be avoided on subsequent sends
	sentRounds := newSentRoundTracker(clearSentRoundsAge)

	// The size of each batch
	var numParts int

	// Timer triggers sending of unfilled batch to prevent hanging when the
	// file part queue has fewer items then the batch size
	timer := time.NewTimer(pollSleepDuration)

	// Tracks time that the last send completed
	var lastSend time.Time

	// Loop forever polling the sendQueue channel for new file parts to send. If
	// the channel is empty, then polling is suspended for pollSleepDuration. If
	// the network is not healthy, then polling is suspended until the network
	// becomes healthy.
	for {
		timer = time.NewTimer(pollSleepDuration)
		select {
		case <-stop.Quit():
			timer.Stop()

			// Close the thread when the stoppable is triggered
			m.closeSendThread(partList, stop, healthChanID)

			return
		case healthy := <-healthChan:
			var wasNotHealthy bool
			// If the network is unhealthy, wait until it becomes healthy
			if !healthy {
				jww.TRACE.Print("[FT] Suspending file part sending thread: " +
					"network is unhealthy.")
				wasNotHealthy = true
			}
			for !healthy {
				healthy = <-healthChan
			}
			if wasNotHealthy {
				jww.TRACE.Print("[FT] File part sending thread: " +
					"network is healthy.")
			}
		case part := <-m.sendQueue:
			// When a part is received from the queue, add it to the list of
			// parts to be sent

			// If the batch is empty (a send just occurred), start a new batch
			if partList == nil {
				rng := m.rng.GetStream()
				numParts = getNumParts(rng)
				rng.Close()
				partList = make([]queuedPart, 0, numParts)
			}

			partList = append(partList, part)

			// If the batch is full, then send the parts
			if len(partList) == numParts {
				quit := m.handleSend(
					&partList, &lastSend, delay, stop, healthChanID, sentRounds)
				if quit {
					timer.Stop()
					return
				}
			}
		case <-timer.C:
			// If the timeout is reached, send an incomplete batch

			// Skip if there are no parts to send
			if len(partList) == 0 {
				continue
			}

			quit := m.handleSend(
				&partList, &lastSend, delay, stop, healthChanID, sentRounds)
			if quit {
				return
			}
		}
	}
}

// closeSendThread safely stops the sending thread by saving unsent parts to the
// queue and setting the stoppable to stopped.
func (m *Manager) closeSendThread(partList []queuedPart, stop *stoppable.Single,
	healthChanID uint64) {
	// Exit the thread if the stoppable is triggered
	jww.DEBUG.Print("[FT] Stopping file part sending thread: stoppable " +
		"triggered.")

	// Add all the unsent parts back in the queue
	for _, part := range partList {
		m.sendQueue <- part
	}

	// Unregister network health channel
	m.net.GetHealthTracker().RemoveChannel(healthChanID)

	// Mark stoppable as stopped
	stop.ToStopped()
}

// handleSend handles the sending of parts with bandwidth limitations. On a
// successful send, the partList is cleared and lastSend is updated to the send
// timestamp. When the stoppable is triggered, closing is automatically handled.
// Returns true if the stoppable has been triggered and the sending thread
// should quit.
func (m *Manager) handleSend(partList *[]queuedPart, lastSend *time.Time,
	delay time.Duration, stop *stoppable.Single, healthChanID uint64,
	sentRounds *sentRoundTracker) bool {
	// Bandwidth limiter: wait to send until the delay has been reached so that
	// the bandwidth is limited to the maximum throughput
	if netTime.Since(*lastSend) < delay {
		waitingTime := delay - netTime.Since(*lastSend)
		jww.TRACE.Printf("[FT] Suspending file part sending (%d parts): "+
			"bandwidth limit reached; waiting %s to send.",
			len(*partList), waitingTime)

		waitingTimer := time.NewTimer(waitingTime)
		select {
		case <-stop.Quit():
			waitingTimer.Stop()

			// Close the thread when the stoppable is triggered
			m.closeSendThread(*partList, stop, healthChanID)

			return true
		case <-waitingTimer.C:
			jww.TRACE.Printf("[FT] Resuming file part sending (%d parts) "+
				"after waiting %s for bandwidth limiting.",
				len(*partList), waitingTime)
		}
	}

	// Send all the messages
	go func(partList []queuedPart, sentRounds *sentRoundTracker) {
		err := m.sendParts(partList, sentRounds)
		if err != nil {
			jww.ERROR.Print(err)
		}
	}(copyPartList(*partList), sentRounds)

	// Update the timestamp of the send
	*lastSend = netTime.Now()

	// Clear partList once done
	*partList = nil

	return false
}

// copyPartList makes a copy of the list of queuedPart.
func copyPartList(partList []queuedPart) []queuedPart {
	newPartList := make([]queuedPart, len(partList))
	copy(newPartList, partList)
	return newPartList
}

// sendParts handles the composing and sending of a cMix message for each part
// in the list. All errors returned are fatal errors.
func (m *Manager) sendParts(partList []queuedPart,
	sentRounds *sentRoundTracker) error {

	// Build cMix messages
	messages, transfers, groupedParts, partsToResend, err :=
		m.buildMessages(partList)
	if err != nil {
		return err
	}

	// Exit if there are no parts to send
	if len(messages) == 0 {
		return nil
	}

	// Clear all old rounds from the sent rounds list
	sentRounds.removeOldRounds()

	// Create cMix parameters with round exclusion list
	p := params.GetDefaultCMIX()
	p.SendTimeout = m.p.SendTimeout
	p.ExcludedRounds = sentRounds
	p.DebugTag = cMixDebugTag

	jww.TRACE.Printf("[FT] Sending %d file parts via SendManyCMIX with "+
		"parameters %+v", len(messages), p)

	// Send parts
	rid, _, err := m.net.SendManyCMIX(messages, p)
	if err != nil {
		// If an error occurs, then print a warning and add the file parts back
		// to the queue to try sending again
		jww.WARN.Printf(sendManyCmixWarn, len(messages), groupedParts, err)

		// Add parts back to queue
		for _, partIndex := range partsToResend {
			m.sendQueue <- partList[partIndex]
		}

		return nil
	}

	// Create list for transfer IDs to watch with the round results callback
	tIDs := make([]ftCrypto.TransferID, 0, len(transfers))

	// Set all parts to in-progress
	for tid, transfer := range transfers {
		exists, err := transfer.SetInProgress(rid, groupedParts[tid]...)
		if err != nil {
			return errors.Errorf(setInProgressErr, groupedParts[tid], tid, err)
		}

		transfer.CallProgressCB(nil)

		// Add transfer ID to list to be tracked; skip if the tracker has
		// already been launched for this transfer and round ID
		if !exists {
			tIDs = append(tIDs, tid)
		}
	}

	// Set up tracker waiting for the round to end to update state and update
	// progress
	roundResultCB := m.makeRoundEventCallback(
		map[id.Round][]ftCrypto.TransferID{rid: tIDs})
	err = m.getRoundResults(
		[]id.Round{rid}, roundResultsTimeout, roundResultCB)
	if err != nil {
		return errors.Errorf(getRoundResultsErr, rid, tIDs, err)
	}

	return nil
}

// buildMessages builds the list of cMix messages to send via SendManyCmix. Also
// returns three separate lists used for later progress tracking. The first, a
// map that contains each unique transfer for each part in the list. The second,
// a map of part numbers being sent grouped by their transfer. The last a list
// of partList index of parts that will be sent. Any part that encounters a
// non-fatal error will be skipped and will not be included in an of the lists.
// All errors returned are fatal errors.
func (m *Manager) buildMessages(partList []queuedPart) (
	[]message.TargetedCmixMessage, map[ftCrypto.TransferID]*ftStorage.SentTransfer,
	map[ftCrypto.TransferID][]uint16, []int, error) {
	messages := make([]message.TargetedCmixMessage, 0, len(partList))
	transfers := map[ftCrypto.TransferID]*ftStorage.SentTransfer{}
	groupedParts := map[ftCrypto.TransferID][]uint16{}
	partsToResend := make([]int, 0, len(partList))

	rng := m.rng.GetStream()
	defer rng.Close()

	for i, part := range partList {
		// Lookup the transfer by the ID; if the transfer does not exist, then
		// print a warning and skip this message
		st, err := m.sent.GetTransfer(part.tid)
		if err != nil {
			jww.WARN.Printf(noSentTransferWarn, part.tid, part.partNum, err)
			continue
		}

		// Generate new cMix message with encrypted file part
		cmixMsg, err := m.newCmixMessage(st, part.partNum)
		if err == ftStorage.MaxRetriesErr {
			jww.DEBUG.Printf("[FT] File transfer %s sent to %s ran out of "+
				"retries {parts: %d, numFps: %d/%d}",
				part.tid, st.GetRecipient(), st.GetNumParts(),
				st.GetNumFps()-st.GetNumAvailableFps(), st.GetNumFps())

			// If the max number of retries has been reached, then report the
			// error on the callback, delete the transfer, and skip to the next
			// message
			go st.CallProgressCB(errors.Errorf(maxRetriesErr, err))
			continue
		} else if err != nil {
			// For all other errors, return an error
			return nil, nil, nil, nil,
				errors.Errorf(newCmixMessageErr, part.partNum, part.tid, err)
		}

		// Construct TargetedCmixMessage
		msg := message.TargetedCmixMessage{
			Recipient: st.GetRecipient(),
			Message:   cmixMsg,
		}

		// Add to list of messages to send
		messages = append(messages, msg)
		transfers[part.tid] = st
		groupedParts[part.tid] = append(groupedParts[part.tid], part.partNum)
		partsToResend = append(partsToResend, i)
	}

	return messages, transfers, groupedParts, partsToResend, nil
}

// newCmixMessage creates a new cMix message with an encrypted file part, its
// MAC, and fingerprint.
func (m *Manager) newCmixMessage(transfer *ftStorage.SentTransfer,
	partNum uint16) (format.Message, error) {
	// Create new empty cMix message
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())

	// Get encrypted file part, file part MAC, nonce (nonce), and fingerprint
	encPart, mac, fp, err := transfer.GetEncryptedPart(partNum, cmixMsg.ContentsSize())
	if err != nil {
		return format.Message{}, err
	}

	// Construct cMix message
	cmixMsg.SetContents(encPart)
	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)

	return cmixMsg, nil
}

// makeRoundEventCallback returns an api.RoundEventCallback that is called once
// the round that file parts were sent on either succeeds or fails. If the round
// succeeds, then all file parts for each transfer are marked as finished and
// the progress callback is called with the current progress. If the round
// fails, then each part for each transfer is removed from the in-progress list,
// added to the end of the sending queue, and the callback called with an error.
func (m *Manager) makeRoundEventCallback(
	sentRounds map[id.Round][]ftCrypto.TransferID) cmix.RoundEventCallback {

	return func(allSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundLookupStatus) {
		for rid, roundResult := range rounds {
			if roundResult == cmix.Succeeded {
				// If the round succeeded, then set all parts for each transfer
				// for this round to finished and call the progress callback
				for _, tid := range sentRounds[rid] {
					st, err := m.sent.GetTransfer(tid)
					if err != nil {
						jww.ERROR.Printf(finishPassNoTransferErr, rid, tid, err)
						continue
					}

					// Mark as finished
					completed, err := st.FinishTransfer(rid)
					if err != nil {
						jww.ERROR.Printf(finishTransferErr, tid, err)
						continue
					}

					// Call progress callback after change in progress
					st.CallProgressCB(nil)

					// If the transfer is complete, send an E2E message to the
					// recipient informing them
					if completed {
						jww.DEBUG.Printf("[FT] Finished sending file "+
							"transfer %s to %s {parts: %d, numFps: %d/%d}",
							tid, st.GetRecipient(), st.GetNumParts(),
							st.GetNumFps()-st.GetNumAvailableFps(),
							st.GetNumFps())

						go func(tid ftCrypto.TransferID, recipient *id.ID) {
							err = m.sendEndE2eMessage(recipient)
							if err != nil {
								jww.ERROR.Printf(finishedEndE2eMsfErr,
									recipient, tid, err)
							}
						}(tid, st.GetRecipient())
					}
				}
			} else {

				jww.WARN.Printf(roundFailureWarn, sentRounds[rid], rid, roundResult)

				// If the round failed, then remove all parts for each transfer
				// for this round from the in-progress list, call the progress
				// callback with an error, and add the parts back into the queue
				for _, tid := range sentRounds[rid] {
					st, err := m.sent.GetTransfer(tid)
					if err != nil {
						jww.ERROR.Printf(finishFailNoTransferErr, rid, tid, err)
						continue
					}

					// Remove parts from in-progress list
					partsToResend, err := st.UnsetInProgress(rid)
					if err != nil {
						jww.ERROR.Printf(
							unsetInProgressErr, tid, roundResult, err)
					}

					// Call progress callback after change in progress
					st.CallProgressCB(nil)

					// Add all the unsent parts back in the queue
					m.queueParts(tid, partsToResend)
				}
			}
		}
	}
}

// sendEndE2eMessage sends an E2E message to the recipient once the transfer
// complete information them that all file parts have been sent.
func (m *Manager) sendEndE2eMessage(recipient *id.ID) error {
	// Get the partner
	partner, err := m.store.E2e().GetPartner(recipient)
	if err != nil {
		return errors.Errorf(endE2eGetPartnerErr, recipient, err)
	}

	// Build the message
	sendMsg := message.Send{
		Recipient:   recipient,
		MessageType: message.EndFileTransfer,
	}

	// Send the message under file transfer preimage
	e2eParams := params.GetDefaultE2E()
	e2eParams.IdentityPreimage = partner.GetFileTransferPreimage()
	e2eParams.DebugTag = "FT.End"

	// Store the message in the critical messages buffer first to ensure it is
	// present if the send fails
	m.store.GetCriticalMessages().AddProcessing(sendMsg, e2eParams)

	// Register health channel and wait for network to become healthy
	healthChan := make(chan bool, networkHealthBuffLen)
	healthChanID := m.net.GetHealthTracker().AddChannel(healthChan)
	defer m.net.GetHealthTracker().RemoveChannel(healthChanID)
	isHealthy := m.net.GetHealthTracker().IsHealthy()
	healthCheckTimer := time.NewTimer(sendEndE2eHealthTimeout)
	for !isHealthy {
		select {
		case isHealthy = <-healthChan:
		case <-healthCheckTimer.C:
			return errors.Errorf(endE2eHealthTimeout, sendEndE2eHealthTimeout)
		}
	}
	healthCheckTimer.Stop()

	// Send E2E message
	rounds, e2eMsgID, _, err := m.net.SendE2E(sendMsg, e2eParams, nil)
	if err != nil {
		return errors.Errorf(endE2eSendErr, recipient, err)
	}

	// Register the event for all rounds
	sendResults := make(chan ds.EventReturn, len(rounds))
	roundEvents := m.net.GetInstance().GetRoundEvents()
	for _, r := range rounds {
		roundEvents.AddRoundEventChan(r, sendResults, 10*time.Second,
			states.COMPLETED, states.FAILED)
	}

	// Wait until the result tracking responds
	success, numTimeOut, numRoundFail := cmix.TrackResults(
		sendResults, len(rounds))

	// If a single partition of the end file transfer message does not transmit,
	// then the partner will not be able to read the confirmation
	if !success {
		jww.ERROR.Printf("[FT] Sending E2E message %s to end file transfer "+
			"with %s failed to transmit %d/%d partitions: %d round failures, "+
			"%d timeouts", recipient, e2eMsgID, numRoundFail+numTimeOut,
			len(rounds), numRoundFail, numTimeOut)
		m.store.GetCriticalMessages().Failed(sendMsg, e2eParams)
		return nil
	}

	// Otherwise, the transmission is a success and this should be denoted in
	// the session and the log
	m.store.GetCriticalMessages().Succeeded(sendMsg, e2eParams)
	jww.INFO.Printf("[FT] Sending of message %s informing %s that a transfer "+
		"completed successfully.", e2eMsgID, recipient)

	return nil
}

// queueParts adds an entry for each file part in the list into the sendQueue
// channel in a random order.
func (m *Manager) queueParts(tid ftCrypto.TransferID, partNums []uint16) {
	// Shuffle the list
	shuffle.Shuffle16(&partNums)

	// Add each part to the queue
	for _, partNum := range partNums {
		m.sendQueue <- queuedPart{tid, partNum}
	}
}

// makeListOfPartNums returns a list of number of file part, from 0 to numParts.
func makeListOfPartNums(numParts uint16) []uint16 {
	partNumList := make([]uint16, numParts)
	for i := range partNumList {
		partNumList[i] = uint16(i)
	}

	return partNumList
}

// getPartSize determines the maximum size for each file part in bytes. The size
// is calculated based on the content size of a cMix message. Returns an error
// if a file part message cannot fit into a cMix message payload.
func (m *Manager) getPartSize() (int, error) {
	// Create new empty cMix message
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())

	// Create new empty file part message of size equal to the available payload
	// size in the cMix message
	partMsg, err := ftStorage.NewPartMessage(cmixMsg.ContentsSize())
	if err != nil {
		return 0, err
	}

	return partMsg.GetPartSize(), nil
}

// partitionFile splits the file into parts of the specified part size.
func partitionFile(file []byte, partSize int) [][]byte {
	// Initialize part list to the correct size
	numParts := (len(file) + partSize - 1) / partSize
	parts := make([][]byte, 0, numParts)
	buff := bytes.NewBuffer(file)

	for n := buff.Next(partSize); len(n) > 0; n = buff.Next(partSize) {
		newPart := make([]byte, partSize)
		copy(newPart, n)
		parts = append(parts, newPart)
	}

	return parts
}

// getRngNum takes in a PRNG source and returns a random number. This type makes
// it easier to test by allowing custom functions that return expected values.
type getRngNum func(rng csprng.Source) int

// getRandomNumParts returns a random number between minPartsSendPerRound and
// maxPartsSendPerRound, inclusive.
func getRandomNumParts(rng csprng.Source) int {
	// Generate random bytes
	b, err := csprng.Generate(8, rng)
	if err != nil {
		jww.FATAL.Panicf(getRandomNumPartsRandPanic, err)
	}

	// Convert bytes to integer
	num := binary.LittleEndian.Uint64(b)

	// Return random number that is minPartsSendPerRound <= num <= max
	return int((num % (maxPartsSendPerRound)) + minPartsSendPerRound)
}
