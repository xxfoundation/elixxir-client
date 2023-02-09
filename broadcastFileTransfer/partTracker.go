////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store"
	"gitlab.com/elixxir/client/v4/storage/utility"
)

// sentFilePartTracker contains utility.StateVector that tracks which parts have
// arrived. It adheres to the FilePartTracker interface.
type sentFilePartTracker struct {
	*utility.MultiStateVector
}

// GetPartStatus returns the status of the sent file part with the given part
// number.
func (s *sentFilePartTracker) GetPartStatus(partNum uint16) FpStatus {
	if partNum >= s.GetNumKeys() {
		return -1
	}

	switch s.Get(partNum) {
	case uint8(store.UnsentPart):
		return FpUnsent
	case uint8(store.SentPart):
		return FpSent
	case uint8(store.ReceivedPart):
		return FpReceived
	default:
		return -1
	}
}

// GetNumParts returns the total number of file parts in the transfer.
func (s *sentFilePartTracker) GetNumParts() uint16 {
	return s.GetNumKeys()
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
