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


var backOffTable = [cappedTries]time.Duration{tryZero, tryOne, tryTwo, tryThree, tryFour, tryFive, trySix}

// processUncheckedRounds will (periodically) check every checkInterval
// for rounds that failed message retrieval in processMessageRetrieval.
// Rounds will have a backoff duration in which they will be tried again.
// If a round is found to be due on a periodical check, the round is sent
// back to processMessageRetrieval.
func (m *Manager) processUncheckedRounds(checkInterval time.Duration, backoffTable [cappedTries]time.Duration,
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
					select {
					case m.lookupRoundMessages <- rl:
					case <- time.After(500*time.Second):
					}

					// Update the state of the round for next look-up (if needed)
					err := uncheckedRoundStore.IncrementCheck(rid)
					if err != nil {
						jww.ERROR.Printf("processUncheckedRounds error: Could not "+
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
func isRoundCheckDue(tries uint64, ts time.Time, backoffTable [cappedTries]time.Duration) bool {
	now := netTime.Now()

	if tries > cappedTries {
		tries = cappedTries
	}
	roundCheckTime := ts.Add(backoffTable[tries])

	return now.After(roundCheckTime)
}
