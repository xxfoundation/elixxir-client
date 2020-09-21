package rounds

import (
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/primitives/id"
)

// getRoundChecker passes a context and the round infos received by the
// gateway to the funky round checker api to update round state.
// The returned function passes round event objects over the context
// to the rest of the message handlers for getting messages.
func (m *Manager) Checker(roundID id.Round, instance *network.Instance) bool {
	// Set round to processing, if we can
	processing, count := m.p.Process(roundID)
	if !processing {
		return false
	}
	if count == m.params.MaxAttemptsCheckingARound {
		m.p.Remove(roundID)
		return true
	}
	// FIXME: Spec has us SETTING processing, but not REMOVING it
	// until the get messages thread completes the lookup, this
	// is smell that needs refining. It seems as if there should be
	// a state that lives with the round info as soon as we know
	// about it that gets updated at different parts...not clear
	// needs to be thought through.
	//defer processing.Remove(roundID)

	// TODO: Bloom filter lookup -- return true when we don't have
	// Go get the round from the round infos, if it exists

	ri, err := instance.GetRound(roundID)
	if err != nil {
		// If we didn't find it, send to historical
		// rounds processor
		m.historicalRounds <- roundID
	} else {
		m.lookupRoundMessages <- ri
	}

	return false
}
