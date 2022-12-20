////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"gitlab.com/xx_network/primitives/netTime"
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
	samples := 10_000
	t1 := netTime.Now()
	sum := int64(0)

	rng := rand.New(rand.NewSource(netTime.Now().UnixNano()))

	for i := 0; i < samples; i++ {
		var msgID channel.MessageID
		rng.Read(msgID[:])
		t2 := mutateTimestamp(t1, msgID)
		delta := t2.Sub(t1)
		sum += abs(int64(delta))
	}

	avg := sum / int64(samples)
	diff := abs(avg - 2_502_865)
	if diff > 38_000 {
		t.Fatalf("Difference %d is greater than %d", diff, 38_000)
	}
}

const generationRange = beforeGrace + afterGrace

// TestVetTimestamp_Happy tests that when the localTS is within the allowed
// range, it is unmodified.
func TestVetTimestamp_Happy(t *testing.T) {
	samples := 10_000

	rng := rand.New(rand.NewSource(netTime.Now().UnixNano()))

	for i := 0; i < samples; i++ {

		now := netTime.Now()

		tested := now.Add(-beforeGrace).Add(
			time.Duration(rng.Int63()) % generationRange)

		var msgID channel.MessageID
		rng.Read(msgID[:])

		result := vetTimestamp(tested, now, msgID)

		if !tested.Equal(result) {
			t.Errorf("Timestamp was molested unexpectedly")
		}
	}
}

// TestVetTimestamp_Happy tests that when the localTS is less than the allowed
// time period it is replaced.
func TestVetTimestamp_BeforePeriod(t *testing.T) {
	samples := 10_000

	rng := rand.New(rand.NewSource(netTime.Now().UnixNano()))

	for i := 0; i < samples; i++ {

		now := netTime.Now()

		tested := now.Add(-beforeGrace).Add(
			-time.Duration(rng.Int63()) % (100_000 * time.Hour))

		var msgID channel.MessageID
		rng.Read(msgID[:])

		result := vetTimestamp(tested, now, msgID)

		if tested.Equal(result) {
			t.Errorf("Timestamp was unmolested unexpectedly")
		}
	}
}

// TestVetTimestamp_Happy tests that when the localTS is greater than the
// allowed time period it is replaced
func TestVetTimestamp_AfterPeriod(t *testing.T) {
	samples := 10_000

	rng := rand.New(rand.NewSource(netTime.Now().UnixNano()))

	for i := 0; i < samples; i++ {

		now := netTime.Now()

		tested := now.Add(afterGrace).Add(
			-time.Duration(rng.Int63()) % (100_000 * time.Hour))

		var msgID channel.MessageID
		rng.Read(msgID[:])

		result := vetTimestamp(tested, now, msgID)

		if tested.Equal(result) {
			t.Errorf("Timestamp was unmolested unexpectedly")
		}
	}
}
