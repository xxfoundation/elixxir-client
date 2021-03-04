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
	// Set round to processing, if we can
	status, count := m.p.Process(roundID, identity.EphId, identity.Source)
	jww.INFO.Printf("checking: %d, status: %s", roundID, status)

	switch status{
	case Processing:
		return false
	case Done:
		return true
	}

	//if the number of times the round has been checked has hit the max, drop it
	if count == m.params.MaxAttemptsCheckingARound {
		jww.ERROR.Printf("Looking up Round %v for %d (%s) failed "+
			"the maximum number of times (%v), stopping retrval attempt",
			roundID, identity.EphId, identity.Source,
			m.params.MaxAttemptsCheckingARound)
		m.p.Done(roundID, identity.EphId, identity.Source)
		return true
	}

	hasRound := false
	//find filters that could have the round and check them
	serialRid := serializeRound(roundID)
	for _, filter := range filters {
		if filter != nil && filter.FirstRound() <= roundID &&
			filter.LastRound() >= roundID {
			if filter.GetFilter().Test(serialRid) {
				hasRound = true
				break
			}
		}
	}

	//if it is not present, set the round as checked
	//that means no messages are available for the user in the round
	if !hasRound {
		jww.DEBUG.Printf("No messages found for %d (%s) in round %d, "+
			"will not check again", identity.EphId.Int64(), identity.Source, roundID)
		m.p.Done(roundID, identity.EphId, identity.Source)
		return true
	}

	// Go get the round from the round infos, if it exists
	ri, err := m.Instance.GetRound(roundID)
	if err != nil || m.params.ForceHistoricalRounds {
		if m.params.ForceHistoricalRounds {
			jww.WARN.Printf("Forcing use of historical rounds for round ID %d.",
				roundID)
		}
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via historical lookup", roundID, identity.EphId.Int64(),
			identity.Source)
		// If we didn't find it, send to Historical Rounds Retrieval
		m.historicalRounds <- historicalRoundRequest{
			rid:      roundID,
			identity: identity,
		}
	} else {
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking " +
			"up messages via in ram lookup", roundID, identity.EphId.Int64(),
			identity.Source)
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
