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

// Tests that TransferStatus.String returns the expected string for each value
// of TransferStatus.
func Test_TransferStatus_String(t *testing.T) {
	testValues := map[TransferStatus]string{
		Running:   "running",
		Completed: "completed",
		Failed:    "failed",
		100:       "INVALID STATUS: 100",
	}

	for status, expected := range testValues {
		if expected != status.String() {
			t.Errorf("TransferStatus string incorrect."+
				"\nexpected: %s\nreceived: %s", expected, status.String())
		}
	}
}
