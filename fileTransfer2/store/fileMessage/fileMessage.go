////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileMessage

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Size constants.
const (
	partNumLen = 2          // The length of the part number in bytes
	fmMinSize  = partNumLen // Minimum size for the PartMessage
)

// Error messages.
const (
	errNewFmSize       = "[FT] Could not create file part message: size of payload (%d) must be greater than %d"
	unmarshalFmSizeErr = "size of passed in bytes (%d) must be greater than %d"
	errSetFileFm       = "[FT] Could not set file part message payload: length of part bytes (%d) must be smaller than maximum payload size %d"
)

/*
+-------------------------------+
|     CMIX Message Contents     |
+-------------+-----------------+
| Part Number |    File Data    |
|   2 bytes   | remaining space |
+-------------+-----------------+
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
func NewPartMessage(externalPayloadSize int) PartMessage {
	if externalPayloadSize < fmMinSize {
		jww.FATAL.Panicf(errNewFmSize, externalPayloadSize, fmMinSize)
	}

	return mapPartMessage(make([]byte, externalPayloadSize))
}

// mapPartMessage maps the data to the components of a PartMessage. It is mapped
// by reference; a copy is not made.
func mapPartMessage(data []byte) PartMessage {
	return PartMessage{
		data:    data,
		partNum: data[:partNumLen],
		part:    data[partNumLen:],
	}
}

// UnmarshalPartMessage converts the bytes into a PartMessage. An error is
// returned if the size of the data is too small for a PartMessage.
func UnmarshalPartMessage(b []byte) (PartMessage, error) {
	if len(b) < fmMinSize {
		return PartMessage{},
			errors.Errorf(unmarshalFmSizeErr, len(b), fmMinSize)
	}

	return mapPartMessage(b), nil
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
func (m PartMessage) SetPart(b []byte) {
	if len(b) > len(m.part) {
		jww.FATAL.Panicf(errSetFileFm, len(b), len(m.part))
	}

	copy(m.part, b)
}

// GetPartSize returns the number of bytes available to store part data.
func (m PartMessage) GetPartSize() int {
	return len(m.part)
}
