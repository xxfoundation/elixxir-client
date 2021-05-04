///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// Constants for message retrieval backoff delays
const (
	tryZero  = 10 * time.Second
	tryOne   = 30 * time.Second
	tryTwo   = 5 * time.Minute
	tryThree = 30 * time.Minute
	tryFour  = 3 * time.Hour
	tryFive  = 12 * time.Hour
	trySix   = 24 * time.Hour
	// Amount of tries past which the
	// backoff will not increase
	cappedTries = 7
)

type backOffTable map[uint64]time.Duration

// uncheckedRoundScheduler will (periodically) check every checkInterval
// for rounds that failed message retrieval in processMessageRetrieval.
// Rounds will have a backoff duration in which they will be tried again.
// If a round is found to be due on a periodical check, the round is sent
// back to processMessageRetrieval.
func (m *Manager) uncheckedRoundScheduler(checkInterval time.Duration, backoffTable backOffTable,
	quitCh <-chan struct{}) {
	ticker := time.NewTicker(checkInterval)
	uncheckedRoundStore := m.Session.UncheckedRounds()
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true

		case <-ticker.C:
			// Pull and iterate through uncheckedRound list
			roundList := m.Session.UncheckedRounds().GetList()
			for rid, rnd := range roundList {
				// If this round is due for a round check, send the round over
				// to the retrieval thread. If not due, check next round.
				if isRoundCheckDue(rnd.NumChecks, rnd.LastCheck, backoffTable) {

					// Construct roundLookup object to send
					rl := roundLookup{
						roundInfo: rnd.Info,
						identity: reception.IdentityUse{
							Identity: reception.Identity{
								EphId:  rnd.EpdId,
								Source: rnd.Source,
							},
						},
					}

					// Send to processMessageRetrieval
					m.lookupRoundMessages <- rl

					// Update the state of the round for next look-up (if needed)
					err := uncheckedRoundStore.IncrementCheck(rid)
					if err != nil {
						jww.ERROR.Printf("uncheckedRoundScheduler error: Could not "+
							"increment check attempts for round %d: %v", rid, err)
					}

				}

			}
		}
	}
}

// isRoundCheckDue given the amount of tries and the timestamp the round
// was stored, determines whether this round is due for another check.
// Returns true if a new check is due
func isRoundCheckDue(tries uint64, ts time.Time, backoffTable backOffTable) bool {
	now := netTime.Now()

	if tries > cappedTries {
		tries = cappedTries
	}
	roundCheckTime := ts.Add(backoffTable[tries])

	return now.After(roundCheckTime)
}

// Constructs a backoff table mapping the amount of tries to
// backoff delay for trying to retrieve messages
func newBackoffTable() backOffTable {
	backoffTable := make(backOffTable)
	backoffTable[0] = tryZero
	backoffTable[1] = tryOne
	backoffTable[2] = tryTwo
	backoffTable[3] = tryThree
	backoffTable[4] = tryFour
	backoffTable[5] = tryFive
	backoffTable[6] = trySix

	return backoffTable
}
