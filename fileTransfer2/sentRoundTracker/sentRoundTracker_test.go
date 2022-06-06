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

package sentRoundTracker

import (
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that NewManager returns the expected new Manager.
func Test_NewSentRoundTracker(t *testing.T) {
	interval := 10 * time.Millisecond
	expected := &Manager{
		rounds: make(map[id.Round]time.Time),
		age:    interval,
	}

	srt := NewManager(interval)

	if !reflect.DeepEqual(expected, srt) {
		t.Errorf("New Manager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, srt)
	}
}

// Tests that Manager.RemoveOldRounds removes only old rounds and not
// newer rounds.
func TestManager_RemoveOldRounds(t *testing.T) {
	srt := NewManager(50 * time.Millisecond)

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
	srt.RemoveOldRounds()

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

// Tests that Manager.Has returns true for all the even rounds and
// false for all odd rounds.
func TestManager_Has(t *testing.T) {
	srt := NewManager(0)

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

// Tests that Manager.Insert adds all the expected rounds to the map and that it
// returns true when the round does not already exist and false otherwise.
func TestManager_Insert(t *testing.T) {
	srt := NewManager(0)

	// Insert even rounds into the tracker
	for rid := id.Round(0); rid < 100; rid++ {
		if rid%2 == 0 {
			if !srt.Insert(rid) {
				t.Errorf("Did not insert round %d.", rid)
			}
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

	// Check that adding a round that already exists returns false
	if srt.Insert(0) {
		t.Errorf("Inserted round %d.", 0)
	}
}

// Tests that Manager.Remove removes all even rounds.
func TestManager_Remove(t *testing.T) {
	srt := NewManager(0)

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

// Tests that Manager.Len returns the expected length when the tracker
// is empty, filled, and then modified.
func TestManager_Len(t *testing.T) {
	srt := NewManager(0)

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
