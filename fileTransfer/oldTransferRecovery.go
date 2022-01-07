////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	oldTransfersRoundResultsErr = "[FT] failed to recover round information " +
		"for %d rounds for old file transfers after %d attempts"
)

// roundResultsMaxAttempts is the maximum number of attempts to get round
// results via api.RoundEventCallback before stopping to try
const roundResultsMaxAttempts = 5

// oldTransferRecovery adds all unsent file parts back into the queue and
// updates the in-progress file parts by getting round updates.
func (m Manager) oldTransferRecovery(healthyChan chan bool, chanID uint64) {
	// Return if there are no parts to recover
	if len(m.recoveredSentRounds) == 0 {
		jww.DEBUG.Print(
			"[FT] No in-progress rounds from old transfers to recover.")
		return
	}

	// Update parts that were sent by looking up the status of the rounds they
	// were sent on
	err := m.updateSentRounds(healthyChan)
	if err != nil {
		jww.ERROR.Print(err)
	}

	// Remove channel from tacker once done with it
	m.net.GetHealthTracker().RemoveChannel(chanID)
}

// updateSentRounds looks up the status of each round that parts were sent on
// but never arrived. It updates the status of each part depending on if the
// round failed or succeeded.
func (m Manager) updateSentRounds(healthyChan chan bool) error {
	// Tracks the number of attempts to get round results
	var getRoundResultsAttempts int

	sentRounds := copySentRounds(m.recoveredSentRounds)
	roundList := roundIdMapToList(sentRounds)
	callback := m.makeRoundEventCallback(m.recoveredSentRounds)

	jww.DEBUG.Print("[FT] Starting old file transfer recovery thread.")

	// Wait for network to be healthy to attempt to get round states
	for getRoundResultsAttempts < roundResultsMaxAttempts {
		select {
		case healthy := <-healthyChan:
			// If the network is unhealthy, wait until it becomes healthy
			if !healthy {
				jww.DEBUG.Print("[FT] Suspending old file transfer recovery " +
					"thread: network is unhealthy.")
			}
			for !healthy {
				healthy = <-healthyChan
			}
			jww.DEBUG.Print("[FT] Old file transfer recovery thread: " +
				"network is healthy.")

			// Register callback to get Round results and retry on error
			err := m.getRoundResults(roundList, roundResultsTimeout, callback)
			if err != nil {
				jww.WARN.Printf("[FT] Failed to get round results for old "+
					"transfers for rounds %d (attempt %d/%d): %+v",
					getRoundResultsAttempts, roundResultsMaxAttempts,
					roundList, err)
			} else {
				jww.INFO.Printf(
					"[FT] Successfully recovered old file transfers: %v",
					sentRounds)

				return nil
			}
			getRoundResultsAttempts++
		}
	}

	return errors.Errorf(
		oldTransfersRoundResultsErr, len(sentRounds), getRoundResultsAttempts)
}

// roundIdMapToList returns a list of all round IDs in the map.
func roundIdMapToList(roundMap map[id.Round][]ftCrypto.TransferID) []id.Round {
	roundSlice := make([]id.Round, 0, len(roundMap))
	for rid := range roundMap {
		roundSlice = append(roundSlice, rid)
	}
	return roundSlice
}

// copySentRounds makes a copy of the sent rounds map.
func copySentRounds(
	sentRounds map[id.Round][]ftCrypto.TransferID) map[id.Round][]ftCrypto.TransferID {
	sentRoundsCopy := make(
		map[id.Round][]ftCrypto.TransferID, len(sentRounds))

	for rid, transfers := range sentRounds {
		sentRoundsCopy[rid] = make([]ftCrypto.TransferID, len(transfers))
		copy(sentRoundsCopy[rid], transfers)
	}

	return sentRoundsCopy
}
