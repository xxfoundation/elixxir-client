////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

type callback func(err error)

// callbackTracker tracks the fileTransfer.SentProgressCallback and
// information on when to call it. The callback will be called on each send,
// unless the time since the lastCall is smaller than the period. In that case,
// a callback is marked as scheduled and waits to be called at the end of the
// period. A callback is called once every period, regardless of the number of
// sends that occur.
type callbackTracker struct {
	period    time.Duration     // How often to call the callback
	lastCall  time.Time         // Timestamp of the last call
	scheduled bool              // Denotes if callback call is scheduled
	complete  bool              // Denotes if the callback should not be called
	stop      *stoppable.Single // Stops the scheduled callback from triggering
	cb        callback
	mux       sync.RWMutex
}

// newCallbackTracker creates a new and unused sentCallbackTracker.
func newCallbackTracker(
	cb callback, period time.Duration, stop *stoppable.Single) *callbackTracker {
	return &callbackTracker{
		period:    period,
		lastCall:  time.Time{},
		scheduled: false,
		complete:  false,
		stop:      stop,
		cb:        cb,
	}
}

// call triggers the progress callback with the most recent progress from the
// sentProgressTracker. If a callback has been called within the last period,
// then a new call is scheduled to occur at the beginning of the next period. If
// a call is already scheduled, then nothing happens; when the callback is
// finally called, it will do so with the most recent changes.
func (ct *callbackTracker) call(err error) {
	ct.mux.RLock()
	// Exit if a callback is already scheduled
	if (ct.scheduled || ct.complete) && err == nil {
		ct.mux.RUnlock()
		return
	}

	ct.mux.RUnlock()
	ct.mux.Lock()
	defer ct.mux.Unlock()

	if (ct.scheduled || ct.complete) && err == nil {
		return
	}

	// Mark callback complete if an error is passed
	ct.complete = err != nil

	// Check if a callback has occurred within the last period
	timeSinceLastCall := netTime.Since(ct.lastCall)
	if timeSinceLastCall > ct.period {

		// If no callback occurred, then trigger the callback now
		ct.cb(err)
		ct.lastCall = netTime.Now()
	} else {
		// If a callback did occur, then schedule a new callback to occur at the
		// start of the next period
		ct.scheduled = true
		go func() {
			timer := time.NewTimer(ct.period - timeSinceLastCall)
			select {
			case <-ct.stop.Quit():
				timer.Stop()
				ct.stop.ToStopped()
				return
			case <-timer.C:
				ct.mux.Lock()
				ct.cb(err)
				ct.lastCall = netTime.Now()
				ct.scheduled = false
				ct.mux.Unlock()
			}
		}()
	}
}
