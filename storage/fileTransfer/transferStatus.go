////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"encoding/binary"
	"strconv"
)

// TransferStatus indicates the state of the transfer.
type TransferStatus int

const (
	Running  TransferStatus = iota // Sending parts
	Stopping                       // Sent last part but not callback
	Stopped                        // Sent last part and callback
)

const invalidTransferStatusStringErr = "INVALID TransferStatus: "

// String prints the string representation of the TransferStatus. This function
// satisfies the fmt.Stringer interface.
func (ts TransferStatus) String() string {
	switch ts {
	case Running:
		return "running"
	case Stopping:
		return "stopping"
	case Stopped:
		return "stopped"
	default:
		return invalidTransferStatusStringErr + strconv.Itoa(int(ts))
	}
}

// Marshal returns the byte representation of the TransferStatus.
func (ts TransferStatus) Marshal() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(ts))
	return b
}

// UnmarshalTransferStatus unmarshalls the 8-byte byte slice into a
// TransferStatus.
func UnmarshalTransferStatus(b []byte) TransferStatus {
	return TransferStatus(binary.LittleEndian.Uint64(b))
}
