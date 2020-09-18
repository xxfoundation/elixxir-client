package network

// File for storing info about which rounds are processing

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type roundStatus struct {
	count      uint
	processing bool
}

// Struct with a lock so we can manage it with concurrent threads
type ProcessingRounds struct {
	rounds map[id.Round]*roundStatus
	sync.RWMutex
}

// NewProcessingRounds returns a processing rounds object
func NewProcessingRounds() *ProcessingRounds {
	return &ProcessingRounds{
		rounds: make(map[id.Round]*roundStatus),
	}
}

// Add a round to the list of processing rounds
// the boolean is true if the round was changes from not processing to processing
// the count is the number of times the round has been processed
func (pr *ProcessingRounds) Process(id id.Round) (bool, uint) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		rs.count++
		process := !rs.processing
		rs.processing = true
		return process, rs.count
	}
	pr.rounds[id] = &roundStatus{
		count:      0,
		processing: true,
	}
	return true, 0
}

// Check if a round ID is marked as processing
func (pr *ProcessingRounds) IsProcessing(id id.Round) bool {
	pr.RLock()
	defer pr.RUnlock()
	if rs, ok := pr.rounds[id]; ok {
		return rs.processing
	}
	return false
}

// set a rounds processing status to failure so it can be retried
func (pr *ProcessingRounds) Fail(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		rs.processing = false
	}
}

// Remove a round from the processing list
func (pr *ProcessingRounds) Remove(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	delete(pr.rounds, id)
}
