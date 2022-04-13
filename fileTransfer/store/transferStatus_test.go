////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"strconv"
	"testing"
)

// Tests that TransferStatus.String returns the expected string for each value
// of TransferStatus.
func Test_TransferStatus_String(t *testing.T) {
	testValues := map[TransferStatus]string{
		Running:   "running",
		Completed: "completed",
		Failed:    "failed",
		100:       invalidTransferStatusStringErr + strconv.Itoa(100),
	}

	for status, expected := range testValues {
		if expected != status.String() {
			t.Errorf("TransferStatus string incorrect."+
				"\nexpected: %s\nreceived: %s", expected, status.String())
		}
	}
}

// Tests that a marshalled and unmarshalled TransferStatus matches the original.
func Test_TransferStatus_Marshal_UnmarshalTransferStatus(t *testing.T) {
	testValues := []TransferStatus{Running, Completed, Failed}

	for _, status := range testValues {
		marshalledStatus := status.Marshal()

		newStatus := UnmarshalTransferStatus(marshalledStatus)

		if status != newStatus {
			t.Errorf("Marshalled and unmarshalled TransferStatus does not "+
				"match original.\nexpected: %s\nreceived: %s", status, newStatus)
		}
	}
}
