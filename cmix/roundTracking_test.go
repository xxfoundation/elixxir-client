////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestRoundTracker(t *testing.T) {
	rt := NewRoundTracker()
	rid := id.Round(2)
	rt.denote(rid, MessageAvailable)
	if rt.state[rid] != MessageAvailable {
		t.Errorf("Round %d is not in expected state\n\tExpected: %s\n\tReceived: %s\n", rid, MessageAvailable, rt.state[rid])
	}
	rt.denote(rid, Unchecked)
	if rt.state[rid] != MessageAvailable {
		t.Errorf("Round %d is not in expected state\n\tExpected: %s\n\tReceived: %s\n", rid, MessageAvailable, rt.state[rid])
	}
	rt.denote(rid, Abandoned)
	if rt.state[rid] != Abandoned {
		t.Errorf("Round %d is not in expected state\n\tExpected: %s\n\tReceived: %s\n", rid, Abandoned, rt.state[rid])
	}
}
