////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Contains functionality related to the event model driven network health
// tracker.

package health

import (
	"sync/atomic"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/comms/network"
)

type Monitor interface {
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
	IsHealthy() bool
	WasHealthy() bool
	StartProcesses() (stoppable.Stoppable, error)
}

type tracker struct {
	// timeout parameter describes how long
	// without good news until the network is considered unhealthy
	timeout time.Duration

	// channel on which new status updates are received from the network handler
	heartbeat chan network.Heartbeat

	// denotes the last time news was heard. Both hold ns since unix epoc
	// in an atomic
	lastCompletedRound *int64
	lastWaitingRound   *int64

	// Denotes that the past health status wasHealthy is true if isHealthy has
	// ever been true in an atomic.
	wasHealthy *uint32

	// stores registered callbacks to receive event updates
	*trackerCallback
}

// Init creates a single HealthTracker thread, starts it, and returns a tracker
// and a stoppable.
func Init(instance *network.Instance, timeout time.Duration) Monitor {

	trkr := newTracker(timeout)
	instance.SetNetworkHealthChan(trkr.heartbeat)

	return trkr
}

// newTracker builds and returns a new tracker object given a Context.
func newTracker(timeout time.Duration) *tracker {

	lastCompletedRound := int64(0)
	lastWaitingRound := int64(0)

	wasHealthy := uint32(0)

	t := &tracker{
		timeout:            timeout,
		heartbeat:          make(chan network.Heartbeat, 100),
		lastCompletedRound: &lastCompletedRound,
		lastWaitingRound:   &lastWaitingRound,
		wasHealthy:         &wasHealthy,
	}
	t.trackerCallback = initTrackerCallback()
	return t
}

// getLastCompletedRoundTimestamp atomically loads the completed round timestamp
// and converts it to a time object, then returns it
func (t *tracker) getLastCompletedRoundTimestamp() time.Time {
	return time.Unix(0, atomic.LoadInt64(t.lastCompletedRound))
}

// getLastWaitingRoundTimestamp atomically loads the waiting round timestamp
// and converts it to a time object, then returns it
func (t *tracker) getLastWaitingRoundTimestamp() time.Time {
	return time.Unix(0, atomic.LoadInt64(t.lastWaitingRound))
}

// IsHealthy returns true if the network is healthy, which is
// defined as the client having knowledge of both valid queued rounds
// and completed rounds within the last tracker.timeout seconds
func (t *tracker) IsHealthy() bool {
	// use the system time instead of netTime.Now() which can
	// include an offset because local monotonicity is what
	// matters here, not correctness relative to absolute time
	now := time.Now()

	completedRecently := false
	if now.Sub(t.getLastCompletedRoundTimestamp()) < t.timeout {
		completedRecently = true
	}

	waitingRecently := false
	if now.Sub(t.getLastWaitingRoundTimestamp()) < t.timeout {
		waitingRecently = true
	}

	return completedRecently && waitingRecently
}

// updateHealth atomically updates the internal
// timestamps to now if there are new waiting / completed
// rounds
func (t *tracker) updateHealth(hasWaiting, hasCompleted bool) {
	// use the system time instead of netTime.Now() which can
	// include an offset because local monotonicity is what
	// matters here, not correctness relative to absolute time
	now := time.Now().UnixNano()

	if hasWaiting {
		atomic.StoreInt64(t.lastWaitingRound, now)
	}

	if hasCompleted {
		atomic.StoreInt64(t.lastCompletedRound, now)
	}
}

// forceUnhealthy cleats the internal timestamps, forcing the
// tracker into unhealthy
func (t *tracker) forceUnhealthy() {
	atomic.StoreInt64(t.lastWaitingRound, 0)
	atomic.StoreInt64(t.lastCompletedRound, 0)
}

// WasHealthy returns true if isHealthy has ever been true.
func (t *tracker) WasHealthy() bool {
	return atomic.LoadUint32(t.wasHealthy) == 1
}

// AddHealthCallback adds a function to the list of tracker functions such that
// each function can be run after network changes. Returns a unique ID for the
// function.
func (t *tracker) AddHealthCallback(f func(isHealthy bool)) uint64 {
	return t.addHealthCallback(f, t.IsHealthy())
}

// StartProcesses starts running the
func (t *tracker) StartProcesses() (stoppable.Stoppable, error) {

	atomic.StoreUint32(t.wasHealthy, 0)

	stop := stoppable.NewSingle("health tracker")

	go t.start(stop)

	return stop, nil
}

// start begins a long-running thread used to monitor and report on network
// health.
func (t *tracker) start(stop *stoppable.Single) {

	// ensures wasHealthy is only set once
	hasSetWasHealthy := false

	// denotation of the previous state in order to catch state changes
	lastState := false

	// flag denoting required exit, allows final signaling
	quit := false

	//ensured the timeout error is only printed once per timeout period
	timedOut := true

	for {

		/* wait for an event */
		select {
		case <-stop.Quit():
			t.forceUnhealthy()

			// flag the quit instead of quitting here so the
			// joint signaling handler code can be triggered
			quit = true

		case heartbeat := <-t.heartbeat:
			t.updateHealth(heartbeat.HasWaitingRound, heartbeat.IsRoundComplete)
			timedOut = false
		case <-time.After(t.timeout):
			if !timedOut {
				jww.ERROR.Printf("Network health tracker timed out, network " +
					"is no longer healthy, follower likely has stopped...")
			}
			timedOut = true

			// note: no need to force to unhealthy because by definition the
			// timestamps will be stale
		}

		/* handle the state change resulting from an event */

		// send signals if the state has changed
		newHealthState := t.IsHealthy()
		if newHealthState != lastState {
			// set was healthy if we are healthy and it was never set before
			if newHealthState && !hasSetWasHealthy {
				atomic.StoreUint32(t.wasHealthy, 1)
				hasSetWasHealthy = true
			}

			//trigger downstream events
			t.callback(newHealthState)

			lastState = newHealthState
		}

		// quit if required to quit
		if quit {
			stop.ToStopped()
			return
		}
	}
}
