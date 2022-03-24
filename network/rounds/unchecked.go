///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/network/rounds/store"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// Constants for message retrieval backoff delays
// todo - make this a real backoff
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
// todo - make this system know which rounds are still in progress instead of just assume by time
func (m *manager) processUncheckedRounds(checkInterval time.Duration, backoffTable [cappedTries]time.Duration,
	stop *stoppable.Single) {
	ticker := time.NewTicker(checkInterval)
	uncheckedRoundStore := m.unchecked
	for {
		select {
		case <-stop.Quit():
			ticker.Stop()
			stop.ToStopped()
			return

		case <-ticker.C:
			iterator := func(rid id.Round, rnd store.UncheckedRound) {
				jww.DEBUG.Printf("checking if %d due for a message lookup", rid)
				// If this round is due for a round check, send the round over
				// to the retrieval thread. If not due, check next round.
				if !isRoundCheckDue(rnd.NumChecks, rnd.LastCheck, backoffTable) {
					return
				}
				jww.INFO.Printf("Round %d due for a message lookup, retrying...", rid)
				//check if it needs to be processed by historical Rounds
				m.GetMessagesFromRound(rid, interfaces.EphemeralIdentity{
					EphId:  rnd.EpdId,
					Source: rnd.Source,
				})
				// Update the state of the round for next look-up (if needed)
				err := uncheckedRoundStore.IncrementCheck(rid, rnd.Source, rnd.EpdId)
				if err != nil {
					jww.ERROR.Printf("processUncheckedRounds error: Could not "+
						"increment check attempts for round %d: %v", rid, err)
				}
			}
			// Pull and iterate through uncheckedRound list
			m.unchecked.IterateOverList(iterator)
		}
	}
}

// isRoundCheckDue given the amount of tries and the timestamp the round
// was stored, determines whether this round is due for another check.
// Returns true if a new check is due
func isRoundCheckDue(tries uint64, ts time.Time, backoffTable [cappedTries]time.Duration) bool {
	now := netTime.Now()

	if tries >= uint64(len(backoffTable)) {
		tries = uint64(len(backoffTable)) - 1
	}
	roundCheckTime := ts.Add(backoffTable[tries])

	return now.After(roundCheckTime)
}
