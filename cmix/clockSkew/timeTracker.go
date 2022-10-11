////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// package clockSkew tracks local clock skew relative to gateways.
package clockSkew

import (
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"time"

	"gitlab.com/xx_network/primitives/id"
)

const maxHistogramSize = 50

// Tracker tracks local clock skew relative to various
// gateways.
type Tracker interface {
	// Add additional data to our aggregate clock skews.
	Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration)

	// Aggregate returns the average of the last n offsets.
	Aggregate() time.Duration
}

// gatewayDelays is a helper type used by the timeOffsetTracker below
// to keep track of the last maxHistogramSize number of durations.
type gatewayDelays struct {
	lock         sync.RWMutex
	delays       []*time.Duration
	currentIndex int
}

func newGatewayDelays() *gatewayDelays {
	return &gatewayDelays{
		delays:       make([]*time.Duration, maxHistogramSize),
		currentIndex: 0,
	}
}

func (g *gatewayDelays) Add(d time.Duration) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.delays[g.currentIndex] = &d
	g.currentIndex += 1
	if g.currentIndex == len(g.delays) {
		g.currentIndex = 0
	}
}

func (g *gatewayDelays) Average() time.Duration {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return average(g.delays)
}

// timeOffsetTracker implements the Tracker
type timeOffsetTracker struct {
	gatewayClockDelays *sync.Map // id.ID -> *gatewayDelays

	lock         sync.RWMutex
	offsets      []*time.Duration
	currentIndex int
	clamp        time.Duration
}

// New returns an implementation of Tracker.
func New(clamp time.Duration) Tracker {
	t := &timeOffsetTracker{
		gatewayClockDelays: new(sync.Map),
		offsets:            make([]*time.Duration, maxHistogramSize),
		currentIndex:       0,
		clamp:              clamp,
	}
	return t
}

// Add implements the Add method of the Tracker interface.
func (t *timeOffsetTracker) Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration) {
	if abs(startTime.Sub(rTs)) > time.Hour {
		jww.WARN.Printf("Time data from %s dropped, more than an hour off from"+
			" local time; local: %s, remote: %s", startTime, rTs)
		return
	}

	delay := (rtt - gwD) / 2

	delays, _ := t.gatewayClockDelays.LoadOrStore(*gwID, newGatewayDelays())

	gwdelays := delays.(*gatewayDelays)
	gwdelays.Add(delay)
	gwDelay := gwdelays.Average()

	offset := startTime.Sub(rTs.Add(-gwDelay))
	t.addOffset(offset)
}

func abs(duration time.Duration) time.Duration {
	if duration < 0 {
		return -duration
	}
	return duration
}

func (t *timeOffsetTracker) addOffset(offset time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.offsets[t.currentIndex] = &offset
	t.currentIndex += 1
	if t.currentIndex == len(t.offsets) {
		t.currentIndex = 0
	}
}

// Aggregate implements the Aggregate method fo the Tracker interface.
func (t *timeOffsetTracker) Aggregate() time.Duration {
	t.lock.RLock()
	defer t.lock.RUnlock()

	avg := average(t.offsets)
	if avg < (-t.clamp) || avg > t.clamp {
		return avg
	} else {
		return 0
	}

}

func average(durations []*time.Duration) time.Duration {
	sum := int64(0)
	count := int64(0)
	for i := 0; i < len(durations); i++ {
		if durations[i] == nil {
			break
		}
		sum += int64(*durations[i])
		count += 1
	}

	return time.Duration(sum / count)
}
