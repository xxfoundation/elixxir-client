////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/utility"
)

// sentPartTracker tracks the status of individual sent file parts.
type sentPartTracker struct {
	// The number of file parts in the file
	numParts uint16

	// Stores the status for each file part in a bitstream format
	partStats *utility.MultiStateVector
}

// newSentPartTracker creates a new sentPartTracker with copies of the
// in-progress and finished status state vectors.
func newSentPartTracker(partStats *utility.MultiStateVector) sentPartTracker {
	return sentPartTracker{
		numParts:  partStats.GetNumKeys(),
		partStats: partStats.DeepCopy(),
	}
}

// GetPartStatus returns the status of the sent file part with the given part
// number. The possible values for the status are:
// 0 = unsent
// 1 = sent (sender has sent a part, but it has not arrived)
// 2 = arrived (sender has sent a part, and it has arrived)
func (spt sentPartTracker) GetPartStatus(partNum uint16) interfaces.FpStatus {
	status, err := spt.partStats.Get(partNum)
	if err != nil {
		jww.FATAL.Fatalf("failed to get status for part %d: %+v", partNum, err)
	}
	return interfaces.FpStatus(status)
}

// GetNumParts returns the total number of file parts in the transfer.
func (spt sentPartTracker) GetNumParts() uint16 {
	return spt.numParts
}
