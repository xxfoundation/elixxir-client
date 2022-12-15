////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import "testing"

func TestMessageType_String(t *testing.T) {
	expected := []string{"Text", "AdminText", "Reaction",
		"Unknown messageType 4", "Unknown messageType 5",
		"Unknown messageType 6", "Unknown messageType 7",
		"Unknown messageType 8", "Unknown messageType 9",
		"Unknown messageType 10"}

	for i := 1; i <= 10; i++ {
		mt := MessageType(i)
		if mt.String() != expected[i-1] {
			t.Errorf("Stringer failed on test %d, %s vs %s", i,
				mt.String(), expected[i-1])
		}
	}
}
