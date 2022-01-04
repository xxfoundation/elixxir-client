////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/utility"
)

// receivedPartTracker tracks the status of individual received file parts.
type receivedPartTracker struct {
	// The number of file parts in the file
	numParts uint16

	// Stores the received status for each file part in a bitstream format
	receivedStatus *utility.StateVector
}

// newReceivedPartTracker creates a new receivedPartTracker with copies of the
// received status state vectors.
func newReceivedPartTracker(received *utility.StateVector) receivedPartTracker {
	return receivedPartTracker{
		numParts:       uint16(received.GetNumKeys()),
		receivedStatus: received.DeepCopy(),
	}
}

// GetPartStatus returns the status of the received file part with the given
// part number. The possible values for the status are:
// 0 = unreceived
// 3 = received (receiver has received a part)
func (rpt receivedPartTracker) GetPartStatus(partNum uint16) interfaces.FpStatus {
	if rpt.receivedStatus.Used(uint32(partNum)) {
		return interfaces.FpReceived
	} else {
		return interfaces.FpUnsent
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (rpt receivedPartTracker) GetNumParts() uint16 {
	return rpt.numParts
}
