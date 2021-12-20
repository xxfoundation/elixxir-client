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

// Tests that ReceivedPartTracker satisfies the interfaces.FilePartTracker
// interface.
func TestReceivedPartTracker_FilePartTrackerInterface(t *testing.T) {
	var _ interfaces.FilePartTracker = ReceivedPartTracker{}
}

// Tests that NewReceivedPartTracker returns a new ReceivedPartTracker with the
// expected values.
func TestNewReceivedPartTracker(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newRandomReceivedTransfer(16, 24, kv, t)

	expected := ReceivedPartTracker{
		numParts:       rt.numParts,
		receivedStatus: rt.receivedStatus.DeepCopy(),
	}

	newRPT := NewReceivedPartTracker(rt.receivedStatus)

	if !reflect.DeepEqual(expected, newRPT) {
		t.Errorf("New ReceivedPartTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newRPT)
	}
}

// Tests that ReceivedPartTracker.GetPartStatus returns the expected status for
// each part loaded from a preconfigured ReceivedTransfer.
func TestReceivedPartTracker_GetPartStatus(t *testing.T) {
	// Create new ReceivedTransfer
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 24, kv, t)

	// Set statuses of parts in the ReceivedTransfer and a map randomly
	prng := rand.New(rand.NewSource(42))
	partStatuses := make(map[uint16]interfaces.FpStatus, rt.numParts)
	for partNum := uint16(0); partNum < rt.numParts; partNum++ {
		partStatuses[partNum] = interfaces.FpStatus(prng.Intn(2)) * interfaces.FpReceived

		if partStatuses[partNum] == interfaces.FpReceived {
			rt.receivedStatus.Use(uint32(partNum))
		}
	}

	// Create a new ReceivedPartTracker from the ReceivedTransfer
	rpt := NewReceivedPartTracker(rt.receivedStatus)

	// Check that the statuses for each part matches the map
	for partNum := uint16(0); partNum < rt.numParts; partNum++ {
		if rpt.GetPartStatus(partNum) != partStatuses[partNum] {
			t.Errorf("Part number %d does not have expected status."+
				"\nexpected: %d\nreceived: %d",
				partNum, partStatuses[partNum], rpt.GetPartStatus(partNum))
		}
	}
}

// Tests that ReceivedPartTracker.GetNumParts returns the same number of parts
// as the ReceivedPartTracker it was created from.
func TestReceivedPartTracker_GetNumParts(t *testing.T) {
	// Create new ReceivedTransfer
	kv := versioned.NewKV(make(ekv.Memstore))
	_, rt, _ := newEmptyReceivedTransfer(16, 24, kv, t)

	// Create a new ReceivedPartTracker from the ReceivedTransfer
	rpt := NewReceivedPartTracker(rt.receivedStatus)

	if rpt.GetNumParts() != rt.GetNumParts() {
		t.Errorf("Number of parts incorrect.\nexpected: %d\nreceived: %d",
			rt.GetNumParts(), rpt.GetNumParts())
	}
}
