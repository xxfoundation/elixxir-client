////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"sync/atomic"
	"time"
)

// sentCallbackTrackerStoppable is the name used for the tracker stoppable.
const sentCallbackTrackerStoppable = "sentCallbackTrackerStoppable"

// sentCallbackTracker tracks the interfaces.SentProgressCallback and
// information on when to call it. The callback will be called on each send,
// unless the time since the lastCall is smaller than the period. In that case,
// a callback is marked as scheduled and waits to be called at the end of the
// period. A callback is called once every period, regardless of the number of
// sends that occur.
type sentCallbackTracker struct {
	period    time.Duration     // How often to call the callback
	lastCall  time.Time         // Timestamp of the last call
	scheduled bool              // Denotes if callback call is scheduled
	completed uint64            // Atomic that tells if transfer is completed
	stop      *stoppable.Single // Stops the scheduled callback from triggering
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
		completed: 0,
		stop:      stoppable.NewSingle(sentCallbackTrackerStoppable),
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
	if sct.scheduled || atomic.LoadUint64(&sct.completed) == 1 {
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
		sct.callNowUnsafe(false, tracker, err)
		sct.lastCall = netTime.Now()
	} else {
		// If a callback did occur, then schedule a new callback to occur at the
		// start of the next period
		sct.scheduled = true
		go func() {
			select {
			case <-sct.stop.Quit():
				sct.stop.ToStopped()
				return
			case <-time.NewTimer(sct.period - timeSinceLastCall).C:
				sct.mux.Lock()
				sct.callNow(false, tracker, err)
				sct.lastCall = netTime.Now()
				sct.scheduled = false
				sct.mux.Unlock()
			}
		}()
	}
}

// stopThread stops all scheduled callbacks.
func (sct *sentCallbackTracker) stopThread() error {
	return sct.stop.Close()
}

// callNow calls the callback immediately regardless of the schedule or period.
func (sct *sentCallbackTracker) callNow(skipCompletedCheck bool,
	tracker sentProgressTracker, err error) {
	completed, sent, arrived, total, t := tracker.GetProgress()
	if skipCompletedCheck || !completed ||
		atomic.CompareAndSwapUint64(&sct.completed, 0, 1) {
		go sct.cb(completed, sent, arrived, total, t, err)
	}
}

// callNowUnsafe calls the callback immediately regardless of the schedule or
// period without taking a thread lock. This function should be used if a lock
// is already taken on the sentProgressTracker.
func (sct *sentCallbackTracker) callNowUnsafe(skipCompletedCheck bool,
	tracker sentProgressTracker, err error) {
	completed, sent, arrived, total, t := tracker.getProgress()
	if skipCompletedCheck || !completed ||
		atomic.CompareAndSwapUint64(&sct.completed, 0, 1) {
		go sct.cb(completed, sent, arrived, total, t, err)
	}
}

// sentProgressTracker interface tracks the progress of a transfer.
type sentProgressTracker interface {
	// GetProgress returns the sent transfer progress in a thread-safe manner.
	GetProgress() (
		completed bool, sent, arrived, total uint16, t interfaces.FilePartTracker)

	// getProgress returns the sent transfer progress in a thread-unsafe manner.
	// This function should be used if a lock is already taken on the sent
	// transfer.
	getProgress() (
		completed bool, sent, arrived, total uint16, t interfaces.FilePartTracker)
}
