package storage

// File for storing info about which rounds are processing

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Struct with a lock so we can manage it with concurrent threads
type ProcessingRounds struct {
	rounds map[id.Round]struct{}
	sync.RWMutex
}

// Insert a round to the list of processing rounds
func (pr *ProcessingRounds) Processing(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	pr.rounds[id] = struct{}{}
}

// Check if a round ID is marked as processing
func (pr *ProcessingRounds) IsProcessing(id id.Round) bool {
	pr.RLock()
	defer pr.RUnlock()
	_, ok := pr.rounds[id]
	return ok
}

// Remove a round from the processing list
func (pr *ProcessingRounds) Done(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	delete(pr.rounds, id)
}
