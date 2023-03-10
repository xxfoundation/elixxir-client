////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	xxid "gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
	"testing"
	"time"
)

func TestPollTracker(t *testing.T) {
	// Create poll tracker
	pt := newPollTracker()

	// Init ID and first EID
	id := xxid.NewIdFromString("zezima", xxid.User, t)
	eid, _, _, err := ephemeral.GetId(id, 16, netTime.Now().UnixNano())
	if err != nil {
		t.Errorf("Failed to create eid for ID %s", id.String())
	}
	eid2, _, _, err := ephemeral.GetId(id, 16, netTime.Now().Add(time.Hour*24).UnixNano())
	if err != nil {
		t.Errorf("Failed to create second eid for ID %s", id.String())
	}

	// Track untracked id & eid
	pt.Track(eid, id)
	if i, ok := (*pt)[*id]; ok {
		if j, ok2 := i[eid.Int64()]; ok2 {
			if j != 1 {
				t.Errorf("First EID entry value not 1")
			}
		} else {
			t.Errorf("No entry made for first EID")
		}
	} else {
		t.Errorf("No entry made for ID")
	}

	// track untracked eid on tracked id
	pt.Track(eid2, id)
	if i, ok := (*pt)[*id]; ok {
		if j, ok2 := i[eid2.Int64()]; ok2 {
			if j != 1 {
				t.Errorf("Second EID entry value not 1")
			}
		} else {
			t.Errorf("No entry made for second EID")
		}
	} else {
		t.Errorf("No entry made for ID (2)")
	}

	// re-add tracked eid & id
	pt.Track(eid2, id)
	if i, ok := (*pt)[*id]; ok {
		if j, ok2 := i[eid2.Int64()]; ok2 {
			if j != 2 {
				t.Errorf("EID entry value not 2")
			}
		} else {
			t.Errorf("No entry made for second EID (2)")
		}
	} else {
		t.Errorf("No entry made for ID (3)")
	}

	// Check report output
	s := strings.TrimSpace(pt.Report())

	expectedReport := "Polled the network 3 times"
	if s != expectedReport {
		t.Errorf("Did not receive expected report\n\tExpected: %s\n\tReceived: %s\n", expectedReport, s)
	}
}
