////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"math/rand"
	"reflect"
	"testing"
)

// Tests that sentPartTracker satisfies the interfaces.FilePartTracker
// interface.
func Test_sentPartTracker_FilePartTrackerInterface(t *testing.T) {
	var _ interfaces.FilePartTracker = sentPartTracker{}
}

// Tests that newSentPartTracker returns a new sentPartTracker with the expected
// values.
func Test_newSentPartTracker(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	expected := sentPartTracker{
		numParts:  st.numParts,
		partStats: st.partStats.DeepCopy(),
	}

	newSPT := newSentPartTracker(st.partStats)

	if !reflect.DeepEqual(expected, newSPT) {
		t.Errorf("New sentPartTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newSPT)
	}
}

// Tests that sentPartTracker.GetPartStatus returns the expected status for each
// part loaded from a preconfigured SentTransfer.
func Test_sentPartTracker_GetPartStatus(t *testing.T) {
	// Create new SentTransfer
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Set statuses of parts in the SentTransfer and a map randomly
	prng := rand.New(rand.NewSource(42))
	partStatuses := make(map[uint16]interfaces.FpStatus, st.numParts)
	for partNum := uint16(0); partNum < st.numParts; partNum++ {
		partStatuses[partNum] = interfaces.FpStatus(prng.Intn(3))

		switch partStatuses[partNum] {
		case interfaces.FpSent:
			err := st.partStats.Set(partNum, inProgress)
			if err != nil {
				t.Errorf(
					"Failed to set part %d to in-progress: %+v", partNum, err)
			}
		case interfaces.FpArrived:
			err := st.partStats.Set(partNum, inProgress)
			if err != nil {
				t.Errorf(
					"Failed to set part %d to in-progress: %+v", partNum, err)
			}
			err = st.partStats.Set(partNum, finished)
			if err != nil {
				t.Errorf("Failed to set part %d to finished: %+v", partNum, err)
			}
		}
	}

	// Create a new sentPartTracker from the SentTransfer
	spt := newSentPartTracker(st.partStats)

	// Check that the statuses for each part matches the map
	for partNum := uint16(0); partNum < st.numParts; partNum++ {
		status := spt.GetPartStatus(partNum)
		if status != partStatuses[partNum] {
			t.Errorf("Part number %d does not have expected status."+
				"\nexpected: %d\nreceived: %d",
				partNum, partStatuses[partNum], status)
		}
	}
}

// Tests that sentPartTracker.GetNumParts returns the same number of parts as
// the SentTransfer it was created from.
func Test_sentPartTracker_GetNumParts(t *testing.T) {
	// Create new SentTransfer
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Create a new sentPartTracker from the SentTransfer
	spt := newSentPartTracker(st.partStats)

	if spt.GetNumParts() != st.GetNumParts() {
		t.Errorf("Number of parts incorrect.\nexpected: %d\nreceived: %d",
			st.GetNumParts(), spt.GetNumParts())
	}
}
