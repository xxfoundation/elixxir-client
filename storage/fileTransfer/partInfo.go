////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"strconv"
)

// partInfo contains the transfer ID and fingerprint number for a file part.
type partInfo struct {
	id    ftCrypto.TransferID
	fpNum uint16
}

// newPartInfo generates a new partInfo with the specified transfer ID and
// fingerprint number for a file part.
func newPartInfo(tid ftCrypto.TransferID, fpNum uint16) *partInfo {
	pi := &partInfo{
		id:    tid,
		fpNum: fpNum,
	}

	return pi
}

// marshal serializes the partInfo into a byte slice.
func (pi *partInfo) marshal() []byte {
	// Construct the buffer
	buff := bytes.NewBuffer(nil)
	buff.Grow(ftCrypto.TransferIdLength + 2)

	// Write the transfer ID to the buffer
	buff.Write(pi.id.Bytes())

	// Write the fingerprint number to the buffer
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, pi.fpNum)
	buff.Write(b)

	// Return the serialized data
	return buff.Bytes()
}

// unmarshalPartInfo deserializes the byte slice into a partInfo.
func unmarshalPartInfo(b []byte) *partInfo {
	buff := bytes.NewBuffer(b)

	// Read transfer ID from the buffer
	transferIDBytes := buff.Next(ftCrypto.TransferIdLength)
	transferID := ftCrypto.UnmarshalTransferID(transferIDBytes)

	// Read the fingerprint number from the buffer
	fpNumBytes := buff.Next(2)
	fpNum := binary.LittleEndian.Uint16(fpNumBytes)

	// Return the reconstructed partInfo
	return &partInfo{
		id:    transferID,
		fpNum: fpNum,
	}
}

// String prints a string representation of partInfo. This functions satisfies
// the fmt.Stringer interface.
func (pi *partInfo) String() string {
	return "{id:" + pi.id.String() + " fpNum:" + strconv.Itoa(int(pi.fpNum)) + "}"
}
