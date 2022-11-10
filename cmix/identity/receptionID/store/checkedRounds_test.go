////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"container/list"
	"encoding/binary"
	"gitlab.com/elixxir/client/v5/storage/utility"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Happy path of NewCheckedRounds.
func TestNewCheckedRounds(t *testing.T) {
	maxRounds := 230
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Create a new BlockStore for storing the round IDs to storage
	store, err := utility.NewBlockStore(
		itemsPerBlock, maxRounds/itemsPerBlock+1, kv)
	if err != nil {
		t.Errorf("Failed to create new BlockStore: %+v", err)
	}

	expected := &CheckedRounds{
		m:         make(map[id.Round]interface{}),
		l:         list.New(),
		recent:    []id.Round{},
		store:     store,
		maxRounds: maxRounds,
	}

	received, err := NewCheckedRounds(maxRounds, kv)
	if err != nil {
		t.Errorf("NewCheckedRounds returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("NewCheckedRounds did not return the exepcted CheckedRounds."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that a CheckedRounds that has been saved and loaded from storage
// matches the original.
func TestCheckedRounds_SaveCheckedRounds_TestLoadCheckedRounds(t *testing.T) {
	// Create new CheckedRounds and add rounds to it
	kv := versioned.NewKV(ekv.MakeMemstore())
	cr, err := NewCheckedRounds(50, kv)
	if err != nil {
		t.Errorf("Failed to make new CheckedRounds: %+v", err)
	}
	for i := id.Round(0); i < 100; i++ {
		cr.Check(i)
	}

	err = cr.SaveCheckedRounds()
	if err != nil {
		t.Errorf("SaveCheckedRounds returned an error: %+v", err)
	}

	cr.Prune()

	newCR, err := LoadCheckedRounds(50, kv)
	if err != nil {
		t.Errorf("LoadCheckedRounds returned an error: %+v", err)
	}

	if !reflect.DeepEqual(cr, newCR) {
		t.Errorf("Failed to store and load CheckedRounds."+
			"\nexpected: %+v\nreceived: %+v", cr, newCR)
	}
}

// Happy path.
func TestCheckedRounds_Next(t *testing.T) {
	cr := newCheckedRounds(100, nil)
	rounds := make([][]byte, 10)
	for i := id.Round(0); i < 10; i++ {
		cr.Check(i)
	}

	for i := id.Round(0); i < 10; i++ {
		round, exists := cr.Next()
		if !exists {
			t.Error("Next returned false when there should be more IDs.")
		}

		rounds[i] = round
	}
	round, exists := cr.Next()
	if exists {
		t.Errorf("Next returned true when the list should be empty: %d", round)
	}

	testCR := newCheckedRounds(100, nil)
	testCR.unmarshal(rounds)

	if !reflect.DeepEqual(cr, testCR) {
		t.Errorf("unmarshal did not return the expected CheckedRounds."+
			"\nexpected: %+v\nreceived: %+v", cr, testCR)
	}
}

// Happy path.
func Test_CheckedRounds_Check(t *testing.T) {
	cr := newCheckedRounds(100, nil)
	var expected []id.Round
	for i := id.Round(1); i < 11; i++ {
		if i%2 == 0 {
			if !cr.Check(i) {
				t.Errorf("Check returned false when the round ID should have "+
					"been added (%d).", i)
			}

			val := cr.l.Back().Value.(id.Round)
			if val != i {
				t.Errorf("Check did not add the round ID to the back of "+
					"the list.\nexpected: %d\nreceived: %d", i, val)
			}
			expected = append(expected, i)
		}
	}

	if !reflect.DeepEqual(cr.recent, expected) {
		t.Errorf("Unexpected list of recent rounds."+
			"\nexpected: %+v\nreceived: %+v", expected, cr.recent)
	}

	for i := id.Round(1); i < 11; i++ {
		result := cr.Check(i)
		if i%2 == 0 {
			if result {
				t.Errorf("Check returned true when the round ID should not "+
					"have been added (%d).", i)
			}
		} else if !result {
			t.Errorf("Check returned false when the round ID should have "+
				"been added (%d).", i)
		} else {
			expected = append(expected, i)
		}
	}

	if !reflect.DeepEqual(cr.recent, expected) {
		t.Errorf("Unexpected list of recent rounds."+
			"\nexpected: %+v\nreceived: %+v", expected, cr.recent)
	}
}

// Happy path.
func TestCheckedRounds_IsChecked(t *testing.T) {
	cr := newCheckedRounds(100, nil)

	for i := id.Round(0); i < 100; i += 2 {
		cr.Check(i)
	}

	for i := id.Round(0); i < 100; i++ {
		if i%2 == 0 {
			if !cr.IsChecked(i) {
				t.Errorf("IsChecked falsly reported round ID %d as not checked.", i)
			}
		} else if cr.IsChecked(i) {
			t.Errorf("IsChecked falsly reported round ID %d as checked.", i)
		}
	}
}

// Happy path.
func TestCheckedRounds_Prune(t *testing.T) {
	cr := newCheckedRounds(5, nil)
	for i := id.Round(0); i < 10; i++ {
		cr.Check(i)
	}

	cr.Prune()

	if len(cr.m) != 5 || cr.l.Len() != 5 {
		t.Errorf("Prune did not remove the correct number of round IDs."+
			"\nexpected: %d\nmap:      %d\nlist:     %d", 5,
			len(cr.m), cr.l.Len())
	}
}

// Happy path: length of the list is not too long and does not need to be
// pruned.
func TestCheckedRounds_Prune_NoChange(t *testing.T) {
	cr := newCheckedRounds(100, nil)
	for i := id.Round(0); i < 10; i++ {
		cr.Check(i)
	}

	cr.Prune()

	if len(cr.m) != 10 || cr.l.Len() != 10 {
		t.Errorf("Prune did not remove the correct number of round IDs."+
			"\nexpected: %d\nmap:      %d\nlist:     %d", 5,
			len(cr.m), cr.l.Len())
	}
}

// Happy path.
func TestCheckedRounds_unmarshal(t *testing.T) {
	expected := newCheckedRounds(100, nil)
	rounds := make([][]byte, 10)
	for i := id.Round(0); i < 10; i++ {
		expected.Check(i)
		rounds[i] = make([]byte, 8)
		binary.LittleEndian.PutUint64(rounds[i], uint64(i))
	}
	expected.recent = []id.Round{}

	cr := newCheckedRounds(100, nil)
	cr.unmarshal(rounds)

	if !reflect.DeepEqual(expected, cr) {
		t.Errorf("unmarshal did not return the expected CheckedRounds."+
			"\nexpected: %+v\nreceived: %+v", expected, cr)
	}
}
