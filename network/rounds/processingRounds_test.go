package rounds

// Testing functions for Processing Round structure

import (
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Tests happy path of newProcessingRounds.
func Test_newProcessingRounds(t *testing.T) {
	expectedPr := &processing{
		rounds: make(map[id.Round]*status),
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
	testData := []struct {
		rid        id.Round
		processing bool
		change     bool
		count      uint
	}{
		{10, true, true, 0},
		{10, true, false, 0},
		{10, false, true, 1},
		{100, true, true, 0},
		{100, true, false, 0},
		{100, false, true, 1},
	}

	for i, d := range testData {
		if _, exists := pr.rounds[d.rid]; exists {
			pr.rounds[d.rid].processing = d.processing
		}
		change, count := pr.Process(d.rid)
		if change != d.change {
			t.Errorf("Process() did not return the correct boolean for round "+
				"ID %d (%d).\n\texpected: %v\n\trecieved: %v",
				d.rid, i, d.change, change)
		}
		if count != d.count {
			t.Errorf("Process did not return the expected count for round ID "+
				"%d (%d).\n\texpected: %d\n\trecieved: %d",
				d.rid, i, d.count, count)
		}

		if _, ok := pr.rounds[d.rid]; !ok {
			t.Errorf("Process() did not add round ID %d to the map (%d).",
				d.rid, i)
		}
	}

}

// Tests happy path of IsProcessing.
func TestProcessing_IsProcessing(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	pr.rounds[rid] = &status{0, true}
	if !pr.IsProcessing(rid) {
		t.Errorf("IsProcessing() should have returned true for round ID %d.", rid)
	}
	pr.rounds[rid].processing = false
	if pr.IsProcessing(rid) {
		t.Errorf("IsProcessing() should have returned false for round ID %d.", rid)
	}
}

// Tests happy path of Fail.
func TestProcessing_Fail(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	pr.rounds[rid] = &status{0, true}
	pr.Fail(rid)
	if pr.rounds[rid].processing {
		t.Errorf("Fail() did not mark processing as false for round id %d.", rid)
	}
}

// Tests happy path of Done.
func TestProcessing_Done(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	pr.rounds[rid] = &status{0, true}
	pr.Done(rid)
	if _, ok := pr.rounds[id.Round(10)]; ok {
		t.Errorf("Done() failed to delete round ID %d.", rid)
	}
}
