///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"reflect"
	"testing"
)

// Testing functions for Processing Round structure

// Tests happy path of newProcessingRounds.
func Test_newProcessingRounds(t *testing.T) {
	expectedPr := &processing{
		rounds: make(map[hashID]*status),
	}

	pr := newProcessingRounds()

	if !reflect.DeepEqual(expectedPr, pr) {
		t.Errorf("Did not get expected processing."+
			"\n\texpected: %v\n\trecieved: %v", expectedPr, pr)
	}
}

// Tests happy path of Process.
func TestProcessing_Process(t *testing.T) {
	pr := newProcessingRounds()

	ephID := ephemeral.Id{}
	source := &id.ID{}

	testData := []struct {
		rid    id.Round
		status Status
		count  uint
	}{
		{10, NotProcessing, 0},
		{10, NotProcessing, 0},
		{10, Processing, 0},
		{100, NotProcessing, 0},
		{100, NotProcessing, 0},
		{100, Processing, 0},
	}

	for i, d := range testData {
		hid := makeHashID(d.rid, ephID, source)
		if _, exists := pr.rounds[hid]; exists {
			pr.rounds[hid].Status = d.status
		}
		status, count := pr.Process(d.rid, ephID, source)
		if status != d.status {
			t.Errorf("Process() did not return the correct boolean for round "+
				"ID %d (%d).\nexpected: %s\nrecieved: %s",
				d.rid, i, d.status, status)
		}
		if count != d.count {
			t.Errorf("Process did not return the expected count for round ID "+
				"%d (%d).\n\texpected: %d\n\trecieved: %d",
				d.rid, i, d.count, count)
		}

		if _, ok := pr.rounds[hid]; !ok {
			t.Errorf("Process() did not add round ID %d to the map (%d).",
				d.rid, i)
		}
	}

}

// Tests happy path of IsProcessing.
func TestProcessing_IsProcessing(t *testing.T) {
	pr := newProcessingRounds()
	ephID := ephemeral.Id{}
	source := &id.ID{}
	rid := id.Round(10)
	hid := makeHashID(rid, ephID, source)
	pr.rounds[hid] = &status{0, Processing}
	if !pr.IsProcessing(rid, ephID, source) {
		t.Errorf("IsProcessing() should have returned %s for round ID %d.", Processing, rid)
	}
	pr.rounds[hid].Status = NotProcessing
	if pr.IsProcessing(rid, ephID, source) {
		t.Errorf("IsProcessing() should have returned %s for round ID %d.", NotProcessing, rid)
	}
}

// Tests happy path of Fail.
func TestProcessing_Fail(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	ephID := ephemeral.Id{}
	source := &id.ID{}
	hid := makeHashID(rid, ephID, source)
	pr.rounds[hid] = &status{0, Processing}
	pr.Fail(rid, ephID, source)
	if pr.rounds[hid].Status == Processing {
		t.Errorf("Fail() did not mark processing as false for round id %d.", rid)
	}
	if pr.rounds[hid].failCount != 1 {
		t.Errorf("Fail() did not increment the fail count of round id %d.", rid)
	}
}

// Tests happy path of Done.
func TestProcessing_Done(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	ephID := ephemeral.Id{}
	source := &id.ID{}
	hid := makeHashID(rid, ephID, source)
	pr.rounds[hid] = &status{0, Processing}
	pr.Done(rid, ephID, source)
	if s, _ := pr.rounds[hid]; s.Status != Done {
		t.Errorf("Done() failed to flag round ID %d.", rid)
	}
}
