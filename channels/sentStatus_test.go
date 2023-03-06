////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import "testing"

// Consistency test of SentStatus.String.
func TestSentStatus_String_Consistency(t *testing.T) {
	tests := map[SentStatus]string{
		Unsent:    "unsent",
		Sent:      "sent",
		Delivered: "delivered",
		Failed:    "failed",
		232:       "Invalid SentStatus: 232",
	}

	for ss, expected := range tests {
		if ss.String() != expected {
			t.Errorf("Incorrect string for SentStatus %d."+
				"\nexpected: %s\nreceived: %s", ss, ss, expected)
		}
	}
}
