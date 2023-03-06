////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"testing"
)

// Consistency test of SentPartStatus.String.
func TestSentPartStatus_String_Consistency(t *testing.T) {
	tests := map[SentPartStatus]string{
		UnsentPart:   "unsent",
		SentPart:     "sent",
		ReceivedPart: "received",
		153:          "INVALID STATUS: 153",
	}
	
	for sps, expected := range tests {
		if sps.String() != expected {
			t.Errorf(
				"Incorrect string.\nexpected: %s\nreceived: %s", expected, sps)
		}
	}
}
