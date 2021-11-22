////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// sentCallbackTracker tracks the interfaces.SentProgressCallback and
// information on when to call it. The callback will be called on each send,
// unless the time since the lastCall is smaller than the period. In that case,
// a callback is marked as scheduled and waits to be called at the end of the
// period. A callback is called once every period, regardless of the number of
// sends that occur.
type sentCallbackTracker struct {
	period    time.Duration // How often to call the callback
	lastCall  time.Time     // Timestamp of the last call
	scheduled bool          // Denotes if callback call is scheduled
	cb        interfaces.SentProgressCallback
	mux       sync.RWMutex
}

// newSentCallbackTracker creates a new and unused sentCallbackTracker.
func newSentCallbackTracker(cb interfaces.SentProgressCallback,
	period time.Duration) *sentCallbackTracker {
	return &sentCallbackTracker{
		period:    period,
		lastCall:  time.Time{},
		scheduled: false,
		cb:        cb,
	}
}

// call triggers the progress callback with the most recent progress from the
// sentProgressTracker. If a callback has been called within the last period,
// then a new call is scheduled to occur at the beginning of the next period. If
// a call is already scheduled, then nothing happens; when the callback is
// finally called, it will do so with the most recent changes.
func (sct *sentCallbackTracker) call(tracker sentProgressTracker, err error) {
	sct.mux.RLock()
	// Exit if a callback is already scheduled
	if sct.scheduled {
		sct.mux.RUnlock()
		return
	}

	sct.mux.RUnlock()
	sct.mux.Lock()
	defer sct.mux.Unlock()

	if sct.scheduled {
		return
	}

	// Check if a callback has occurred within the last period
	timeSinceLastCall := netTime.Since(sct.lastCall)
	if timeSinceLastCall > sct.period {
		// If no callback occurred, then trigger the callback now
		sct.callNow(tracker, err)
		sct.lastCall = netTime.Now()
	} else {
		// If a callback did occur, then schedule a new callback to occur at the
		// start of the next period
		sct.scheduled = true
		go func() {
			select {
			case <-time.NewTimer(sct.period - timeSinceLastCall).C:
				sct.mux.Lock()
				sct.callNow(tracker, err)
				sct.lastCall = netTime.Now()
				sct.scheduled = false
				sct.mux.Unlock()
			}
		}()
	}
}

// callNow calls the callback immediately regardless of the schedule or period.
func (sct *sentCallbackTracker) callNow(tracker sentProgressTracker, err error) {
	completed, sent, arrived, total := tracker.GetProgress()
	go sct.cb(completed, sent, arrived, total, err)
}

// callNowUnsafe calls the callback immediately regardless of the schedule or
// period without taking a thread lock. This function should be used if a lock
// is already taken on the sentProgressTracker.
func (sct *sentCallbackTracker) callNowUnsafe(tracker sentProgressTracker, err error) {
	completed, sent, arrived, total := tracker.getProgress()
	go sct.cb(completed, sent, arrived, total, err)
}

// sentProgressTracker interface tracks the progress of a transfer.
type sentProgressTracker interface {
	// GetProgress returns the sent transfer progress in a thread-safe manner.
	GetProgress() (completed bool, sent, arrived, total uint16)

	// getProgress returns the sent transfer progress in a thread-unsafe manner.
	// This function should be used if a lock is already taken on the sent
	// transfer.
	getProgress() (completed bool, sent, arrived, total uint16)
}
