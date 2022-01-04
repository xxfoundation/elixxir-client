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

// receivedCallbackTrackerStoppable is the name used for the tracker stoppable.
const receivedCallbackTrackerStoppable = "receivedCallbackTrackerStoppable"

// receivedCallbackTracker tracks the interfaces.ReceivedProgressCallback and
// information on when to call it. The callback will be called on each file part
// reception, unless the time since the lastCall is smaller than the period. In
// that case, a callback is marked as scheduled and waits to be called at the
// end of the period. A callback is called once every period, regardless of the
// number of receptions that occur.
type receivedCallbackTracker struct {
	period    time.Duration     // How often to call the callback
	lastCall  time.Time         // Timestamp of the last call
	scheduled bool              // Denotes if callback call is scheduled
	completed uint64            // Atomic that tells if transfer is completed
	stop      *stoppable.Single // Stops the scheduled callback from triggering
	cb        interfaces.ReceivedProgressCallback
	mux       sync.RWMutex
}

// newReceivedCallbackTracker creates a new and unused receivedCallbackTracker.
func newReceivedCallbackTracker(cb interfaces.ReceivedProgressCallback,
	period time.Duration) *receivedCallbackTracker {
	return &receivedCallbackTracker{
		period:    period,
		lastCall:  time.Time{},
		scheduled: false,
		completed: 0,
		stop:      stoppable.NewSingle(receivedCallbackTrackerStoppable),
		cb:        cb,
	}
}

// call triggers the progress callback with the most recent progress from the
// receivedProgressTracker. If a callback has been called within the last
// period, then a new call is scheduled to occur at the beginning of the next
// period. If a call is already scheduled, then nothing happens; when the
// callback is finally called, it will do so with the most recent changes.
func (rct *receivedCallbackTracker) call(tracker receivedProgressTracker, err error) {
	rct.mux.RLock()
	// Exit if a callback is already scheduled
	if rct.scheduled || atomic.LoadUint64(&rct.completed) == 1 {
		rct.mux.RUnlock()
		return
	}

	rct.mux.RUnlock()
	rct.mux.Lock()
	defer rct.mux.Unlock()

	if rct.scheduled {
		return
	}

	// Check if a callback has occurred within the last period
	timeSinceLastCall := netTime.Since(rct.lastCall)
	if timeSinceLastCall > rct.period {
		// If no callback occurred, then trigger the callback now
		rct.callNowUnsafe(false, tracker, err)
		rct.lastCall = netTime.Now()
	} else {
		// If a callback did occur, then schedule a new callback to occur at the
		// start of the next period
		rct.scheduled = true
		go func() {
			select {
			case <-rct.stop.Quit():
				rct.stop.ToStopped()
				return
			case <-time.NewTimer(rct.period - timeSinceLastCall).C:
				rct.mux.Lock()
				rct.callNow(false, tracker, err)
				rct.lastCall = netTime.Now()
				rct.scheduled = false
				rct.mux.Unlock()
			}
		}()
	}
}

// stopThread stops all scheduled callbacks.
func (rct *receivedCallbackTracker) stopThread() error {
	return rct.stop.Close()
}

// callNow calls the callback immediately regardless of the schedule or period.
func (rct *receivedCallbackTracker) callNow(skipCompletedCheck bool,
	tracker receivedProgressTracker, err error) {
	completed, received, total, t := tracker.GetProgress()
	if skipCompletedCheck || !completed ||
		atomic.CompareAndSwapUint64(&rct.completed, 0, 1) {
		go rct.cb(completed, received, total, t, err)
	}
}

// callNowUnsafe calls the callback immediately regardless of the schedule or
// period without taking a thread lock. This function should be used if a lock
// is already taken on the receivedProgressTracker.
func (rct *receivedCallbackTracker) callNowUnsafe(skipCompletedCheck bool,
	tracker receivedProgressTracker, err error) {
	completed, received, total, t := tracker.getProgress()
	if skipCompletedCheck || !completed ||
		atomic.CompareAndSwapUint64(&rct.completed, 0, 1) {
		go rct.cb(completed, received, total, t, err)
	}
}

// receivedProgressTracker interface tracks the progress of a transfer.
type receivedProgressTracker interface {
	// GetProgress returns the received transfer progress in a thread-safe
	// manner.
	GetProgress() (
		completed bool, received, total uint16, t interfaces.FilePartTracker)

	// getProgress returns the received transfer progress in a thread-unsafe
	// manner. This function should be used if a lock is already taken on the
	// sent transfer.
	getProgress() (
		completed bool, received, total uint16, t interfaces.FilePartTracker)
}
