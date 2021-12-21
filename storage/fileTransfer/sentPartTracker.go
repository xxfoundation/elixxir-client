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

// sentPartTracker tracks the status of individual sent file parts.
type sentPartTracker struct {
	// The number of file parts in the file
	numParts uint16

	// Stores the in-progress status for each file part in a bitstream format
	inProgressStatus *utility.StateVector

	// Stores the finished status for each file part in a bitstream format
	finishedStatus *utility.StateVector
}

// newSentPartTracker creates a new sentPartTracker with copies of the
// in-progress and finished status state vectors.
func newSentPartTracker(inProgress, finished *utility.StateVector) sentPartTracker {
	return sentPartTracker{
		numParts:         uint16(inProgress.GetNumKeys()),
		inProgressStatus: inProgress.DeepCopy(),
		finishedStatus:   finished.DeepCopy(),
	}
}

// GetPartStatus returns the status of the sent file part with the given part
// number. The possible values for the status are:
// 0 = unsent
// 1 = sent (sender has sent a part, but it has not arrived)
// 2 = arrived (sender has sent a part, and it has arrived)
func (spt sentPartTracker) GetPartStatus(partNum uint16) interfaces.FpStatus {
	if spt.inProgressStatus.Used(uint32(partNum)) {
		return interfaces.FpSent
	} else if spt.finishedStatus.Used(uint32(partNum)) {
		return interfaces.FpArrived
	} else {
		return interfaces.FpUnsent
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (spt sentPartTracker) GetNumParts() uint16 {
	return spt.numParts
}
