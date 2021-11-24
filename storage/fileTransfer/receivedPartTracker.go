////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import "gitlab.com/elixxir/client/storage/utility"

// ReceivedPartTracker tracks the status of individual received file parts.
type ReceivedPartTracker struct {
	// The number of file parts in the file
	numParts uint16

	// Stores the received status for each file part in a bitstream format
	receivedStatus *utility.StateVector
}

// NewReceivedPartTracker creates a new ReceivedPartTracker with copies of the
// received status state vectors.
func NewReceivedPartTracker(received *utility.StateVector) ReceivedPartTracker {
	return ReceivedPartTracker{
		numParts:       uint16(received.GetNumKeys()),
		receivedStatus: received.DeepCopy(),
	}
}

// GetPartStatus returns the status of the received file part with the given part
// number. The possible values for the status are:
// 0 = unreceived
// 3 = received (receiver has received a part)
func (rpt ReceivedPartTracker) GetPartStatus(partNum uint16) int {
	if rpt.receivedStatus.Used(uint32(partNum)) {
		return receivedStatus
	} else {
		return unsentStatus
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (rpt ReceivedPartTracker) GetNumParts() uint16 {
	return rpt.numParts
}
