///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/xx_network/primitives/id"
)

// the round checker is a single use function which is meant to be wrapped
// and adhere to the knownRounds checker interface. it receives a round ID and
// looks up the state of that round to determine if the client has a message
// waiting in it.
// It will return true if it can conclusively determine no message exists,
// returning false and set the round to processing if it needs further
// investigation.
// Once it determines messages might be waiting in a round, it determines
// if the information about that round is already present, if it is the data is
// sent to Message Retrieval Workers, otherwise it is sent to Historical Round
// Retrieval
func (m *Manager) Checker(roundID id.Round, filters []*RemoteFilter, identity reception.IdentityUse) bool {
	jww.DEBUG.Printf("Checker(roundID: %d)", roundID)
	// Set round to processing, if we can
	processing, count := m.p.Process(roundID)
	if !processing {
		// if is already processing, ignore
		return false
	}

	//if the number of times the round has been checked has hit the max, drop it
	if count == m.params.MaxAttemptsCheckingARound {
		jww.ERROR.Printf("Looking up Round %v failed the maximum number "+
			"of times (%v), stopping retrval attempt", roundID,
			m.params.MaxAttemptsCheckingARound)
		m.p.Done(roundID)
		return true
	}

	//find filters that could have the round
	var potentialFilters []*bloom.Ring

	for _, filter := range filters {
		if filter.FirstRound() <= roundID && filter.LastRound() >= roundID {
			potentialFilters = append(potentialFilters, filter.GetFilter())
		}
	}

	hasRound := false
	//check if the round is in any of the potential filters
	if len(potentialFilters) > 0 {
		serialRid := serializeRound(roundID)
		for _, f := range potentialFilters {
			if f.Test(serialRid) {
				hasRound = true
				break
			}
		}
	}

	//if it is not present, set the round as checked
	//that means no messages are available for the user in the round
	if !hasRound {
		m.p.Done(roundID)
		return true
	}

	// Go get the round from the round infos, if it exists
	ri, err := m.Instance.GetRound(roundID)
	if err != nil || m.params.ForceHistoricalRounds {
		if m.params.ForceHistoricalRounds {
			jww.WARN.Printf("Forcing use of historical rounds for round ID %d.",
				roundID)
		}
		jww.DEBUG.Printf("HistoricalRound <- %d", roundID)
		// If we didn't find it, send to Historical Rounds Retrieval
		m.historicalRounds <- historicalRoundRequest{
			rid:      roundID,
			identity: identity,
		}
	} else {
		jww.DEBUG.Printf("lookupRoundMessages <- %d", roundID)
		// If found, send to Message Retrieval Workers
		m.lookupRoundMessages <- roundLookup{
			roundInfo: ri,
			identity:  identity,
		}
	}

	return false
}

func serializeRound(roundId id.Round) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(roundId))
	return b
}
