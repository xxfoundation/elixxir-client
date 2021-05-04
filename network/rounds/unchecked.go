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

func (m *Manager) UncheckedRoundScheduler(checkInterval time.Duration,
	quitCh <-chan struct{}) {
	ticker := time.NewTicker(checkInterval)
	uncheckedRoundStore := m.Session.UncheckedRounds()
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true

		case <-ticker.C:
			roundList := m.Session.UncheckedRounds().GetList()
			for rid, rnd := range roundList {
				// If this round is due for a round check, send the round over
				// to the retrieval thread
				if isRoundCheckDue(rnd.NumTries, rnd.StoredTimestamp) {
					ri, err := m.Instance.GetRound(rid)
					if err != nil {
						jww.WARN.Printf("Could not get round %s from instance", err)
						continue
					}

					// Construct roundLookup object to send
					rl := roundLookup{
						roundInfo: ri,
						identity: reception.IdentityUse{
							Identity: reception.Identity{
								EphId:  rnd.EpdId,
								Source: rnd.Source,
							},
						},
					}

					m.lookupRoundMessages <- rl
					err = uncheckedRoundStore.IncrementCheck(rid)
					if err != nil {
						jww.ERROR.Printf("UncheckedRoundScheduler error: Could not " +
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
func isRoundCheckDue(tries uint, ts time.Time) bool {
	now := netTime.Now()

	roundCheckTime := ts.Add(calculateBackoff(tries))


	return now.After(roundCheckTime)
}

// calculateBackoff is a helper function which returns
// the total time interval
func calculateBackoff(tries uint) time.Duration {
	// todo: implement me
	return 0
}

