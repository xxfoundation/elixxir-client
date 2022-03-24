package network

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/network/identity/receptionID/store"
	"gitlab.com/xx_network/primitives/id"
)

// Checker is a single use function which is meant to be wrapped
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
func Checker(roundID id.Round, filters []*RemoteFilter, cr *store.CheckedRounds) bool {
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
