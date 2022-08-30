package channels

import (
	"math/rand"
	"testing"
	"time"

	"gitlab.com/elixxir/crypto/channel"
)

// withinMutationWindow is a utility test function to check if a mutated
// timestamp is within the allowable window
func withinMutationWindow(raw, mutated time.Time) bool {
	lowerBound := raw.Add(-time.Duration(halfTenMsInNs))
	upperBound := raw.Add(time.Duration(halfTenMsInNs))

	return mutated.After(lowerBound) && mutated.Before(upperBound)
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func TestMutateTimestampDeltaAverage(t *testing.T) {
	samples := 10000
	t1 := time.Now()
	sum := int64(0)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < samples; i++ {
		var msgID channel.MessageID
		rng.Read(msgID[:])
		t2 := mutateTimestamp(t1, msgID)
		delta := t2.Sub(t1)
		sum += abs(int64(delta))
	}

	avg := sum / int64(samples)
	diff := abs(avg - 2502865)
	if diff > 30000 {
		t.Fatal()
	}
}
