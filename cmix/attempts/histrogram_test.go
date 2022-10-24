////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package attempts

import (
	"math/rand"
	"reflect"
	"testing"
)

// Tests that NewSendAttempts returns a new sendAttempts with the expected
// fields.
func TestNewSendAttempts(t *testing.T) {
	optimalAttempts := int32(optimalAttemptsInitValue)
	expected := &sendAttempts{
		optimalAttempts: &optimalAttempts,
		isFull:          false,
		currentIndex:    0,
		numAttempts:     make([]int, maxHistogramSize),
	}

	sa := NewSendAttempts()

	if !reflect.DeepEqual(expected, sa) {
		t.Errorf("New SendAttemptTracker does not match expected."+
			"\nexpected: %+v\nreceivedL %+v", expected, sa)
	}
}

// Tests that sendAttempts.SubmitProbeAttempt properly increments and stores the
// attempts.
func Test_sendAttempts_SubmitProbeAttempt(t *testing.T) {
	sa := NewSendAttempts().(*sendAttempts)

	for i := 0; i < maxHistogramSize+20; i++ {
		sa.SubmitProbeAttempt(i)

		if sa.currentIndex != (i+1)%maxHistogramSize {
			t.Errorf("Incorrect currentIndex (%d).\nexpected: %d\nreceived: %d",
				i, (i+1)%maxHistogramSize, sa.currentIndex)
		} else if sa.numAttempts[i%maxHistogramSize] != i {
			t.Errorf("Incorrect numAttempts at %d.\nexpected: %d\nreceived: %d",
				i, i, sa.numAttempts[i%maxHistogramSize])
		} else if i > maxHistogramSize && !sa.isFull {
			t.Errorf("Should be marked full when numAttempts > %d.",
				maxHistogramSize)
		}
	}
}

// Tests sendAttempts.GetOptimalNumAttempts returns numbers close to 70% of the
// average of attempts feeding in.
func Test_sendAttempts_GetOptimalNumAttempts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	sa := NewSendAttempts().(*sendAttempts)

	attempts, ready := sa.GetOptimalNumAttempts()
	if ready {
		t.Errorf("Marked ready when no attempts have been made.")
	} else if attempts != 0 {
		t.Errorf("Incorrect number of attempt.\nexpected: %d\nreceived: %d",
			0, attempts)
	}

	const n = 100
	factor := (n * 7) / 10
	for i := 0; i < 500; i++ {
		sa.SubmitProbeAttempt(prng.Intn(n))
		attempts, ready = sa.GetOptimalNumAttempts()

		if (sa.currentIndex < minElements && !sa.isFull) && ready {
			t.Errorf("Ready when less than %d attempts made (%d).",
				minElements, i)
		} else if sa.currentIndex >= minElements {
			if !ready {
				t.Errorf("Not ready when more than %d attempts made (%d).",
					minElements, i)
			} else if attempts < factor-25 || attempts > factor+25 {
				t.Errorf("Attempts is not close to average (%d)."+
					"\naverage:  %d\nattempts: %d", i, factor, attempts)
			}
		}
	}
}
