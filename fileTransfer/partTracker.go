////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/storage/utility"
)

// sentFilePartTracker contains utility.StateVector that tracks which parts have
// arrived. It adheres to the FilePartTracker interface.
type sentFilePartTracker struct {
	*utility.StateVector
}

// GetPartStatus returns the status of the sent file part with the given part
// number.
func (s *sentFilePartTracker) GetPartStatus(partNum uint16) FpStatus {
	if uint32(partNum) >= s.GetNumKeys() {
		return -1
	}

	switch s.Used(uint32(partNum)) {
	case true:
		return FpArrived
	case false:
		return FpUnsent
	default:
		return -1
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (s *sentFilePartTracker) GetNumParts() uint16 {
	return uint16(s.GetNumKeys())
}

// receivedFilePartTracker contains utility.StateVector that tracks which parts
// have been received. It adheres to the FilePartTracker interface.
type receivedFilePartTracker struct {
	*utility.StateVector
}

// GetPartStatus returns the status of the received file part with the given
// part number.
func (r *receivedFilePartTracker) GetPartStatus(partNum uint16) FpStatus {
	if uint32(partNum) >= r.GetNumKeys() {
		return -1
	}

	switch r.Used(uint32(partNum)) {
	case true:
		return FpReceived
	case false:
		return FpUnsent
	default:
		return -1
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (r *receivedFilePartTracker) GetNumParts() uint16 {
	return uint16(r.GetNumKeys())
}
