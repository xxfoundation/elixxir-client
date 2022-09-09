////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"testing"
)

// Unit test of Status.String.
func TestStatus_String(t *testing.T) {
	testValues := []struct {
		status   Status
		expected string
	}{
		{Running, "running"},
		{Stopping, "stopping"},
		{Stopped, "stopped"},
		{100, "INVALID STATUS: 100"},
	}

	for i, val := range testValues {
		if val.status.String() != val.expected {
			t.Errorf("String did not return the expected value (%d)."+
				"\nexpected: %s\nreceived: %s", i, val.status.String(), val.expected)
		}
	}
}
