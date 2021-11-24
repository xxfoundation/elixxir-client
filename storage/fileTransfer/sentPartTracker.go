////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/storage/utility"
)

// Part statuses.
const (
	unsentStatus   = 0
	sentStatus     = 1
	arrivedStatus  = 2
	receivedStatus = 3
)

// SentPartTracker tracks the status of individual sent file parts.
type SentPartTracker struct {
	// The number of file parts in the file
	numParts uint16

	// Stores the in-progress status for each file part in a bitstream format
	inProgressStatus *utility.StateVector

	// Stores the finished status for each file part in a bitstream format
	finishedStatus *utility.StateVector
}

// NewSentPartTracker creates a new SentPartTracker with copies of the
// in-progress and finished status state vectors.
func NewSentPartTracker(inProgress, finished *utility.StateVector) SentPartTracker {
	return SentPartTracker{
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
func (spt SentPartTracker) GetPartStatus(partNum uint16) int {
	if spt.inProgressStatus.Used(uint32(partNum)) {
		return sentStatus
	} else if spt.finishedStatus.Used(uint32(partNum)) {
		return arrivedStatus
	} else {
		return unsentStatus
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (spt SentPartTracker) GetNumParts() uint16 {
	return spt.numParts
}
