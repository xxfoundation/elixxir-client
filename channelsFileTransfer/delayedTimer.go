////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import "time"

// The DelayedTimer type represents a single event manually started.
// When the DelayedTimer expires, the current time will be sent on C.
// A DelayedTimer must be created with NewDelayedTimer.
type DelayedTimer struct {
	d time.Duration
	t *time.Timer
	C *<-chan time.Time
}

// NewDelayedTimer creates a new DelayedTimer that will send the current time on
// its channel after at least duration d once it is started.
func NewDelayedTimer(d time.Duration) *DelayedTimer {
	c := make(<-chan time.Time)
	return &DelayedTimer{
		d: d,
		C: &c,
	}
}

// Start starts the timer that will send the current time on its channel after
// at least duration d. If it is already running or stopped, it does nothing.
func (dt *DelayedTimer) Start() {
	if dt.t == nil {
		dt.t = time.NewTimer(dt.d)
		dt.C = &dt.t.C
	}
}

// Stop prevents the Timer from firing.
// It returns true if the call stops the timer, false if the timer has already
// expired, been stopped, or was never started.
func (dt *DelayedTimer) Stop() bool {
	if dt.t == nil {
		return false
	}

	return dt.t.Stop()
}
