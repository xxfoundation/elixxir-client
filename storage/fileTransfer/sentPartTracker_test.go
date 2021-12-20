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

// Tests that SentPartTracker satisfies the interfaces.FilePartTracker
// interface.
func TestSentPartTracker_FilePartTrackerInterface(t *testing.T) {
	var _ interfaces.FilePartTracker = SentPartTracker{}
}

// Tests that NewSentPartTracker returns a new SentPartTracker with the expected
// values.
func TestNewSentPartTracker(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	expected := SentPartTracker{
		numParts:         st.numParts,
		inProgressStatus: st.inProgressStatus.DeepCopy(),
		finishedStatus:   st.finishedStatus.DeepCopy(),
	}

	newSPT := NewSentPartTracker(st.inProgressStatus, st.finishedStatus)

	if !reflect.DeepEqual(expected, newSPT) {
		t.Errorf("New SentPartTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newSPT)
	}
}

// Tests that SentPartTracker.GetPartStatus returns the expected status for each
// part loaded from a preconfigured SentTransfer.
func TestSentPartTracker_GetPartStatus(t *testing.T) {
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
			st.inProgressStatus.Use(uint32(partNum))
		case interfaces.FpArrived:
			st.finishedStatus.Use(uint32(partNum))
		}
	}

	// Create a new SentPartTracker from the SentTransfer
	spt := NewSentPartTracker(st.inProgressStatus, st.finishedStatus)

	// Check that the statuses for each part matches the map
	for partNum := uint16(0); partNum < st.numParts; partNum++ {
		if spt.GetPartStatus(partNum) != partStatuses[partNum] {
			t.Errorf("Part number %d does not have expected status."+
				"\nexpected: %d\nreceived: %d",
				partNum, partStatuses[partNum], spt.GetPartStatus(partNum))
		}
	}
}

// Tests that SentPartTracker.GetNumParts returns the same number of parts as
// the SentTransfer it was created from.
func TestSentPartTracker_GetNumParts(t *testing.T) {
	// Create new SentTransfer
	kv := versioned.NewKV(make(ekv.Memstore))
	_, st := newRandomSentTransfer(16, 24, kv, t)

	// Create a new SentPartTracker from the SentTransfer
	spt := NewSentPartTracker(st.inProgressStatus, st.finishedStatus)

	if spt.GetNumParts() != st.GetNumParts() {
		t.Errorf("Number of parts incorrect.\nexpected: %d\nreceived: %d",
			st.GetNumParts(), spt.GetNumParts())
	}
}
