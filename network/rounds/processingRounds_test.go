///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

// Testing functions for Processing Round structure

/*
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
		rid        id.Round
		processing bool
		change     bool
		count      uint
	}{
		{10, true, true, 0},
		{10, true, false, 0},
		{10, false, true, 0},
		{100, true, true, 0},
		{100, true, false, 0},
		{100, false, true, 0},
	}

	for i, d := range testData {
		hid := makeHashID(d.rid, ephID, source)
		if _, exists := pr.rounds[hid]; exists {
			pr.rounds[hid].processing = d.processing
		}
		change, _, count := pr.Process(d.rid, ephID, source)
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
	pr.rounds[hid] = &status{0, true, false}
	if !pr.IsProcessing(rid, ephID, source) {
		t.Errorf("IsProcessing() should have returned true for round ID %d.", rid)
	}
	pr.rounds[hid].processing = false
	if pr.IsProcessing(rid, ephID, source) {
		t.Errorf("IsProcessing() should have returned false for round ID %d.", rid)
	}
}

// Tests happy path of Fail.
func TestProcessing_Fail(t *testing.T) {
	pr := newProcessingRounds()
	rid := id.Round(10)
	ephID := ephemeral.Id{}
	source := &id.ID{}
	hid := makeHashID(rid, ephID, source)
	pr.rounds[hid] = &status{0, true, false}
	pr.Fail(rid, ephID, source)
	if pr.rounds[hid].processing {
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
	pr.rounds[hid] = &status{0, true, false}
	pr.Done(rid, ephID, source)
	if s, _ := pr.rounds[hid]; !s.done {
		t.Errorf("Done() failed to flag round ID %d.", rid)
	}
}*/
