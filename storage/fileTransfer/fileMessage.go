///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

// Size constants.
const (
	partNumLen = 2          // The length of the part number in bytes
	FmMinSize  = partNumLen // Minimum size for the PartMessage
)

// Error messages.
const (
	newFmSizeErr       = "size of external payload (%d) must be greater than %d"
	unmarshalFmSizeErr = "size of passed in bytes (%d) must be greater than %d"
	setFileFmErr       = "length of part bytes (%d) must be smaller than maximum payload size %d"
)

/*
+-----------------------------------------+
|          CMIX Message Contents          |
+---------+-------------+-----------------+
| Padding | Part Number |    File Data    |
| 8 bytes |   2 bytes   | remaining space |
+---------+-------------+-----------------+
*/

// PartMessage contains part of the data being transferred and 256-bit nonce
// that is used as a nonce.
type PartMessage struct {
	data    []byte // Serial of all contents
	partNum []byte // The part number of the file
	part    []byte // File part data
}

// NewPartMessage generates a new part message that fits into the specified
// external payload size. An error is returned if the external payload size is
// too small to fit the part message.
func NewPartMessage(externalPayloadSize int) (PartMessage, error) {
	if externalPayloadSize < FmMinSize {
		return PartMessage{},
			errors.Errorf(newFmSizeErr, externalPayloadSize, FmMinSize)
	}

	return MapPartMessage(make([]byte, externalPayloadSize)), nil
}

// MapPartMessage maps the data to the components of a PartMessage. It is mapped
// by reference; a copy is not made.
func MapPartMessage(data []byte) PartMessage {
	return PartMessage{
		data:    data,
		partNum: data[:partNumLen],
		part:    data[partNumLen:],
	}
}

// UnmarshalPartMessage converts the bytes into a PartMessage. An error is
// returned if the size of the data is too small for a PartMessage.
func UnmarshalPartMessage(b []byte) (PartMessage, error) {
	if len(b) < FmMinSize {
		return PartMessage{},
			errors.Errorf(unmarshalFmSizeErr, len(b), FmMinSize)
	}

	return MapPartMessage(b), nil
}

// Marshal returns the byte representation of the PartMessage.
func (m PartMessage) Marshal() []byte {
	b := make([]byte, len(m.data))
	copy(b, m.data)
	return b
}

// GetPartNum returns the file part number.
func (m PartMessage) GetPartNum() uint16 {
	return binary.LittleEndian.Uint16(m.partNum)
}

// SetPartNum sets the file part number.
func (m PartMessage) SetPartNum(num uint16) {
	b := make([]byte, partNumLen)
	binary.LittleEndian.PutUint16(b, num)
	copy(m.partNum, b)
}

// GetPart returns the file part data from the message.
func (m PartMessage) GetPart() []byte {
	b := make([]byte, len(m.part))
	copy(b, m.part)
	return b
}

// SetPart sets the PartMessage part to the given bytes. An error is returned if
// the size of the provided part data is too large to store.
func (m PartMessage) SetPart(b []byte) error {
	if len(b) > len(m.part) {
		return errors.Errorf(setFileFmErr, len(b), len(m.part))
	}

	copy(m.part, b)

	return nil
}

// GetPartSize returns the number of bytes available to store part data.
func (m PartMessage) GetPartSize() int {
	return len(m.part)
}
