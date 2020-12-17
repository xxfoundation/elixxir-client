///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

// File for storing info about which rounds are processing

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type status struct {
	failCount  uint
	processing bool
}

// processing struct with a lock so it can be managed with concurrent threads.
type processing struct {
	rounds map[id.Round]*status
	sync.RWMutex
}

// newProcessingRounds returns a new processing rounds object.
func newProcessingRounds() *processing {
	return &processing{
		rounds: make(map[id.Round]*status),
	}
}

// Process adds a round to the list of processing rounds. The returned boolean
// is true when the round changes from "not processing" to "processing". The
// returned count is the number of times the round has been processed.
func (pr *processing) Process(id id.Round) (bool, uint) {
	pr.Lock()
	defer pr.Unlock()

	if rs, ok := pr.rounds[id]; ok {
		if rs.processing {
			return false, rs.failCount
		}
		rs.processing = true

		return true, rs.failCount
	}

	pr.rounds[id] = &status{
		failCount:  0,
		processing: true,
	}

	return true, 0
}

// IsProcessing determines if a round ID is marked as processing.
func (pr *processing) IsProcessing(id id.Round) bool {
	pr.RLock()
	defer pr.RUnlock()

	if rs, ok := pr.rounds[id]; ok {
		return rs.processing
	}

	return false
}

// Fail sets a round's processing status to failed and increments its fail
// counter so that it can be retried.
func (pr *processing) Fail(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		rs.processing = false
		rs.failCount++
	}
}

// Done deletes a round from the processing list.
func (pr *processing) Done(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	delete(pr.rounds, id)
}

// NotProcessing sets a round's processing status to failed so that it can be
// retried but does not increment its fail counter.
func (pr *processing) NotProcessing(id id.Round) {
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[id]; ok {
		rs.processing = false
	}
}
