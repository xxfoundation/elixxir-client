////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that newSentRoundTracker returns the expected new sentRoundTracker.
func Test_newSentRoundTracker(t *testing.T) {
	interval := 10 * time.Millisecond
	expected := &sentRoundTracker{
		rounds: make(map[id.Round]time.Time),
		age:    interval,
	}

	srt := newSentRoundTracker(interval)

	if !reflect.DeepEqual(expected, srt) {
		t.Errorf("New sentRoundTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, srt)
	}
}

// Tests that sentRoundTracker.removeOldRounds removes only old rounds and not
// newer rounds.
func Test_sentRoundTracker_removeOldRounds(t *testing.T) {
	srt := newSentRoundTracker(50 * time.Millisecond)

	// Add odd round to tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 != 0 {
			srt.Insert(rid)
		}
	}

	time.Sleep(50 * time.Millisecond)

	// Add even round to tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			srt.Insert(rid)
		}
	}

	// Remove all old rounds (should be all odd rounds)
	srt.removeOldRounds()

	// Check that only even rounds exist
	for rid := id.Round(0); rid < 100; rid++ {
		if srt.Has(rid) {
			if rid%2 != 0 {
				t.Errorf("Round %d exists.", rid)
			}
		} else if rid%2 == 0 {
			t.Errorf("Round %d does not exist.", rid)
		}
	}
}

// Tests that sentRoundTracker.Has returns true for all the even rounds and
// false for all odd rounds.
func Test_sentRoundTracker_Has(t *testing.T) {
	srt := newSentRoundTracker(0)

	// Insert even rounds into the tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			srt.Insert(rid)
		}
	}

	// Check that only even rounds exist
	for rid := id.Round(0); rid < 100; rid++ {
		if srt.Has(rid) {
			if rid%2 != 0 {
				t.Errorf("Round %d exists.", rid)
			}
		} else if rid%2 == 0 {
			t.Errorf("Round %d does not exist.", rid)
		}
	}
}

// Tests that sentRoundTracker.Insert adds all the expected rounds.
func Test_sentRoundTracker_Insert(t *testing.T) {
	srt := newSentRoundTracker(0)

	// Insert even rounds into the tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			srt.Insert(rid)
		}
	}

	// Check that only even rounds were added
	for rid := id.Round(0); rid < 100; rid++ {
		_, exists := srt.rounds[rid]
		if exists {
			if rid%2 != 0 {
				t.Errorf("Round %d exists.", rid)
			}
		} else if rid%2 == 0 {
			t.Errorf("Round %d does not exist.", rid)
		}
	}
}

// Tests that sentRoundTracker.Remove removes all even rounds.
func Test_sentRoundTracker_Remove(t *testing.T) {
	srt := newSentRoundTracker(0)

	// Add all round to tracker
	for rid := id.Round(0); rid < 100; rid++ {
		srt.Insert(rid)
	}

	// Remove even rounds from the tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			srt.Remove(rid)
		}
	}

	// Check that only even rounds were removed
	for rid := id.Round(0); rid < 100; rid++ {
		_, exists := srt.rounds[rid]
		if exists {
			if rid%2 == 0 {
				t.Errorf("Round %d does not exist.", rid)
			}
		} else if rid%2 != 0 {
			t.Errorf("Round %d exists.", rid)
		}
	}
}

// Tests that sentRoundTracker.Len returns the expected length when the tracker
// is empty, filled, and then modified.
func Test_sentRoundTracker_Len(t *testing.T) {
	srt := newSentRoundTracker(0)

	if srt.Len() != 0 {
		t.Errorf("Length of tracker incorrect.\nexpected: %d\nreceived: %d",
			0, srt.Len())
	}

	// Add all round to tracker
	for rid := id.Round(0); rid < 100; rid++ {
		srt.Insert(rid)
	}

	if srt.Len() != 100 {
		t.Errorf("Length of tracker incorrect.\nexpected: %d\nreceived: %d",
			100, srt.Len())
	}

	// Remove even rounds from the tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			srt.Remove(rid)
		}
	}

	if srt.Len() != 50 {
		t.Errorf("Length of tracker incorrect.\nexpected: %d\nreceived: %d",
			50, srt.Len())
	}
}
