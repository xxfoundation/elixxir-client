package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
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
func (m *Manager) Checker(roundID id.Round) bool {
	jww.INFO.Printf("Checking round ID: %d", roundID)
	// Set round to processing, if we can
	processing, count := m.p.Process(roundID)
	if !processing {
		// if is already processing, ignore
		return false
	}

	//if the number of times the round has been checked has hit the max, drop it
	if count == m.params.MaxAttemptsCheckingARound {
		jww.ERROR.Printf("Round %v failed the maximum number of times "+
			"(%v), stopping retrval attempt", roundID,
			m.params.MaxAttemptsCheckingARound)
		m.p.Done(roundID)
		return true
	}

	// TODO: Bloom filter lookup -- return true when we don't have

	// Go get the round from the round infos, if it exists
	ri, err := m.Instance.GetRound(roundID)
	if err != nil {
		jww.INFO.Printf("Historical Round: %d", roundID)
		// If we didn't find it, send to Historical Rounds Retrieval
		m.historicalRounds <- roundID
	} else {
		jww.INFO.Printf("Looking up Round: %d", roundID)
		// IF found, send to Message Retrieval Workers
		m.lookupRoundMessages <- ri
	}

	return false
}
