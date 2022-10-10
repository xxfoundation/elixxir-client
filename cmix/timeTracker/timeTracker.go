////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// package timeTracker tracks local clock skew relative to gateways.
package timeTracker

import (
	"sync"
	"time"

	"gitlab.com/xx_network/primitives/id"
)

const maxHistogramSize = 50

// TimeOffsetTracker tracks local clock skew relative to various
// gateways.
type TimeOffsetTracker interface {
	// Add additional data to our aggregate clock skews.
	Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration)

	// Aggregate returns the average of the last n offsets.
	Aggregate() time.Duration
}

type gatewayDelays struct {
	lock   sync.RWMutex
	delays []time.Duration
}

type timeOffsetTracker struct {
	numAvg int

	gatewayClockDelays *sync.Map // id.ID -> []time.Duration

	lock    sync.RWMutex
	offsets []time.Duration
}

// New returns an implementation of TimeOffsetTracker.
func New() TimeOffsetTracker {
	t := &timeOffsetTracker{
		numAvg:             maxHistogramSize,
		gatewayClockDelays: new(sync.Map),
	}
	return t
}

func (t *timeOffsetTracker) Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration) {
	delay := rtt/2 - gwD

	delays, _ := t.gatewayClockDelays.LoadOrStore(*gwID, []time.Duration{})

	gwdelays := delays.(*gatewayDelays)

	gwdelays.lock.Lock()
	pushDurationRing(delay, gwdelays.delays, t.numAvg)
	gwDelay := average(gwdelays.delays)
	gwdelays.lock.Unlock()

	offset := startTime.Sub(rTs.Add(-gwDelay))

	t.lock.Lock()
	defer t.lock.Unlock()
	pushDurationRing(offset, t.offsets, t.numAvg)
}

func (t *timeOffsetTracker) Aggregate() time.Duration {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return average(t.offsets)
}

func pushDurationRing(duration time.Duration, durations []time.Duration, size int) {
	durations = append(durations, duration)
	if len(durations) > size {
		durations = durations[1:]
	}
}

func average(durations []time.Duration) time.Duration {
	sum := int64(0)
	for i := 0; i < len(durations); i++ {
		sum += int64(durations[i])
	}
	return time.Duration(sum / int64(len(durations)))
}
