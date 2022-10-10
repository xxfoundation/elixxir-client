package timeTracker

import (
	"sync"
	"time"

	"gitlab.com/xx_network/primitives/id"
)

// TimeOffsetTracker tracks local clock skew relative to various
// gateways.
type TimeOffsetTracker interface {
	Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration)
	Aggregate() time.Duration
}

type timeOffsetTracker struct {
	numAvg int

	gatewayClockDelays map[id.ID][]time.Duration

	lock    sync.RWMutex
	offsets []time.Duration
}

// New returns an implementation of TimeOffsetTracker.
func New() TimeOffsetTracker {
	t := &timeOffsetTracker{
		numAvg: 50,
	}
	return t
}

// Add additional data to our aggregate clock skews.
func (t *timeOffsetTracker) Add(gwID *id.ID, startTime, rTs time.Time, rtt, gwD time.Duration) {
	delay := rtt/2 - gwD

	t.lock.Lock()
	defer t.lock.Unlock()

	_, ok := t.gatewayClockDelays[*gwID]
	if !ok {
		t.gatewayClockDelays[*gwID] = []time.Duration{}
	}

	pushDurationRing(delay, t.gatewayClockDelays[*gwID], t.numAvg)
	gwDelay := average(t.gatewayClockDelays[*gwID])

	delayTime1 := rTs.Add(-gwDelay)
	offsetTime := startTime.Add(-rTs.Sub(delayTime1))
	offset := offsetTime.Sub(startTime)

	pushDurationRing(offset, t.offsets, t.numAvg)
}

// Aggregate returns the average of the last n offsets.
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
