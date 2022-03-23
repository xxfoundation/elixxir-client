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
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/storage/rounds"
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
// false: no message
// true: message
func Checker(roundID id.Round, filters []*RemoteFilter, cr *rounds.CheckedRounds) bool {
	// Skip checking if the round is already checked
	if cr.IsChecked(roundID) {
		return true
	}

	//find filters that could have the round and check them
	serialRid := serializeRound(roundID)
	for _, filter := range filters {
		if filter != nil && filter.FirstRound() <= roundID &&
			filter.LastRound() >= roundID {
			if filter.GetFilter().Test(serialRid) {
				return true
			}
		}
	}
	return false
}

func serializeRound(roundId id.Round) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(roundId))
	return b
}

func (m *Manager) GetMessagesFromRound(roundID id.Round, identity receptionID.IdentityUse) {
	ri, err := m.Instance.GetRound(roundID)
	if err != nil || m.params.ForceHistoricalRounds {
		if m.params.ForceHistoricalRounds {
			jww.WARN.Printf("Forcing use of historical rounds for round ID %d.",
				roundID)
		}
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via historical lookup", roundID, identity.EphId.Int64(),
			identity.Source)
		//store the round as an unretreived round
		err = m.Session.UncheckedRounds().AddRound(roundID, nil,
			identity.Source, identity.EphId)
		if err != nil {
			jww.FATAL.Panicf("Failed to denote Unchecked Round for round %d", roundID)
		}
		// If we didn't find it, send to Historical Rounds Retrieval
		m.historicalRounds <- historicalRoundRequest{
			rid:         roundID,
			identity:    identity,
			numAttempts: 0,
		}
	} else {
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via in ram lookup", roundID, identity.EphId.Int64(),
			identity.Source)
		//store the round as an unretreived round
		if !m.params.RealtimeOnly {
			err = m.Session.UncheckedRounds().AddRound(roundID, ri,
				identity.Source, identity.EphId)
			if err != nil {
				jww.FATAL.Panicf("Failed to denote Unchecked Round for round %d", roundID)
			}
		}

		// If found, send to Message Retrieval Workers
		m.lookupRoundMessages <- roundLookup{
			roundInfo: ri,
			identity:  identity,
		}
	}

}
