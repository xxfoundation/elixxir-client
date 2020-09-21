package rounds

// File for storing info about which rounds are processing

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type status struct {
	count      uint
	processing bool
}

// Struct with a lock so we can manage it with concurrent threads
type processing struct {
	rounds map[id.Round]*status
	sync.RWMutex
}

// NewProcessingRounds returns a processing rounds object
func newProcessingRounds() *processing {
	return &processing{
		rounds: make(map[id.Round]*status),
	}
}

// Add a round to the list of processing rounds
// the boolean is true if the round was changes from not processing to processing
// the count is the number of times the round has been processed
func (pr *processing) Process(id id.Round) (bool, uint) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		if rs.processing {
			return false, rs.count
		}
		rs.count++
		rs.processing = true
		return true, rs.count
	}
	pr.rounds[id] = &status{
		count:      0,
		processing: true,
	}
	return true, 0
}

// Check if a round ID is marked as processing
func (pr *processing) IsProcessing(id id.Round) bool {
	pr.RLock()
	defer pr.RUnlock()
	if rs, ok := pr.rounds[id]; ok {
		return rs.processing
	}
	return false
}

// set a rounds processing status to failure so it can be retried
func (pr *processing) Fail(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		rs.processing = false
	}
}

// Done a round from the processing list
func (pr *processing) Done(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	delete(pr.rounds, id)
}
