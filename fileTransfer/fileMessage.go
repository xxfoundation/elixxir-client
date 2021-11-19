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
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// Size constants.
const (
	paddingLen = ftCrypto.NonceSize      // The length of the padding in bytes
	partNumLen = 2                       // The length of the part number in bytes
	fmMinSize  = partNumLen + paddingLen // Minimum size for the partMessage
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

// partMessage contains part of the data being transferred and 256-bit padding
// that is used as a nonce.
type partMessage struct {
	data    []byte // Serial of all contents
	padding []byte // Random padding bytes
	partNum []byte // The part number of the file
	part    []byte // File part data
}

// newPartMessage generates a new part message that fits into the specified
// external payload size. An error is returned if the external payload size is
// too small to fit the part message.
func newPartMessage(externalPayloadSize int) (partMessage, error) {
	if externalPayloadSize < fmMinSize {
		return partMessage{},
			errors.Errorf(newFmSizeErr, externalPayloadSize, fmMinSize)
	}

	return mapPartMessage(make([]byte, externalPayloadSize)), nil
}

// mapPartMessage maps the data to the components of a partMessage. It is mapped
// by reference; a copy is not made.
func mapPartMessage(data []byte) partMessage {
	return partMessage{
		data:    data,
		padding: data[:paddingLen],
		partNum: data[paddingLen : paddingLen+partNumLen],
		part:    data[paddingLen+partNumLen:],
	}
}

// unmarshalPartMessage converts the bytes into a partMessage. An error is
// returned if the size of the data is too small for a partMessage.
func unmarshalPartMessage(b []byte) (partMessage, error) {
	if len(b) < fmMinSize {
		return partMessage{},
			errors.Errorf(unmarshalFmSizeErr, len(b), fmMinSize)
	}

	return mapPartMessage(b), nil
}

// marshal returns the byte representation of the partMessage.
func (m partMessage) marshal() []byte {
	return m.data
}

// getPadding returns the padding in the message.
func (m partMessage) getPadding() []byte {
	return m.padding
}

// setPadding sets the partMessage padding to the given bytes. Note that this
// padding should be random bytes generated via the appropriate crypto function.
func (m partMessage) setPadding(b []byte) {
	copy(m.padding, b)
}

// getPartNum returns the file part number.
func (m partMessage) getPartNum() uint16 {
	return binary.LittleEndian.Uint16(m.partNum)
}

// setPartNum sets the file part number.
func (m partMessage) setPartNum(num uint16) {
	b := make([]byte, partNumLen)
	binary.LittleEndian.PutUint16(b, num)
	copy(m.partNum, b)
}

// getPart returns the file part data from the message.
func (m partMessage) getPart() []byte {
	return m.part
}

// setPart sets the partMessage part to the given bytes. An error is returned if
// the size of the provided part data is too large to store.
func (m partMessage) setPart(b []byte) error {
	if len(b) > len(m.part) {
		return errors.Errorf(setFileFmErr, len(b), len(m.part))
	}

	copy(m.part, b)

	return nil
}

// getPartSize returns the number of bytes available to store part data.
func (m partMessage) getPartSize() int {
	return len(m.part)
}
