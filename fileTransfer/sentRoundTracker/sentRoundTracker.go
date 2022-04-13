////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sentRoundTracker

import (
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// Manager keeps track of rounds that file parts were sent on and when
// those rounds occurred. Rounds past the given age can be deleted manually.
// Adheres to excludedRounds.ExcludedRounds.
type Manager struct {
	rounds map[id.Round]time.Time
	age    time.Duration
	mux    sync.RWMutex
}

// NewManager creates a new sent round tracker Manager.
func NewManager(interval time.Duration) *Manager {
	return &Manager{
		rounds: make(map[id.Round]time.Time),
		age:    interval,
	}
}

// RemoveOldRounds removes any rounds that are older than the max round age.
func (srt *Manager) RemoveOldRounds() {
	srt.mux.Lock()
	defer srt.mux.Unlock()
	deleteBefore := netTime.Now().Add(-srt.age)

	for rid, timeStamp := range srt.rounds {
		if timeStamp.Before(deleteBefore) {
			delete(srt.rounds, rid)
		}
	}
}

// Has indicates if the round ID is in the tracker.
func (srt *Manager) Has(rid id.Round) bool {
	srt.mux.RLock()
	defer srt.mux.RUnlock()

	_, exists := srt.rounds[rid]
	return exists
}

// Insert adds the round to the tracker with the current time. Returns true if
// the round was added.
func (srt *Manager) Insert(rid id.Round) bool {
	timeNow := netTime.Now()
	srt.mux.Lock()
	defer srt.mux.Unlock()

	_, exists := srt.rounds[rid]
	if exists {
		return false
	}

	srt.rounds[rid] = timeNow
	return true
}

// Remove deletes a round ID from the tracker.
func (srt *Manager) Remove(rid id.Round) {
	srt.mux.Lock()
	defer srt.mux.Unlock()
	delete(srt.rounds, rid)
}

// Len returns the number of round IDs in the tracker.
func (srt *Manager) Len() int {
	srt.mux.RLock()
	defer srt.mux.RUnlock()

	return len(srt.rounds)
}
