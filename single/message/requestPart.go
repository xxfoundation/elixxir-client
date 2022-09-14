////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Error messages.
const (
	// NewRequestPart
	errReqPartPayloadSize = "[SU] Failed to create new single-use request " +
		"message part: external payload size (%d) is smaller than the " +
		"minimum message size for a request part (%d)."

	// UnmarshalRequestPart
	errReqPartDataSize = "size of data (%d) must be at least %d"

	// RequestPart.SetContents
	errReqPartContentsSize = "[SU] Failed to set contents of single-use " +
		"request message part: size of the supplied contents (%d) is larger " +
		"than the max message size (%d)."
)

// Sizes of fields.
const (
	reqPartPartNumLen = 1
	reqPartSizeLen    = 2
	reqPartMinSize    = reqPartPartNumLen + reqPartSizeLen
)

/*
+------------------------------+
|    cMix Message Contents     |
+---------+---------+----------+
| partNum |  size   | contents |
|  1 byte | 2 bytes | variable |
+---------+---------+----------+
*/

type RequestPart struct {
	data     []byte // Serial of all contents
	partNum  []byte // Index of message in a series of messages
	size     []byte // Size of the contents
	contents []byte // The encrypted contents
}

// NewRequestPart generates a new request message part of the specified size.
func NewRequestPart(externalPayloadSize int) RequestPart {
	if externalPayloadSize < reqPartMinSize {
		jww.FATAL.Panicf(
			errReqPartPayloadSize, externalPayloadSize, reqPartMinSize)
	}

	rmp := mapRequestPart(make([]byte, externalPayloadSize))
	return rmp
}

// GetRequestPartContentsSize returns the size of the contents for the given
// external payload size.
func GetRequestPartContentsSize(externalPayloadSize int) int {
	return externalPayloadSize - reqPartMinSize
}

// mapRequestPart builds a message part mapped to the passed in data.
// It is mapped by reference; a copy is not made.
func mapRequestPart(data []byte) RequestPart {
	return RequestPart{
		data:     data,
		partNum:  data[:reqPartPartNumLen],
		size:     data[reqPartPartNumLen:reqPartMinSize],
		contents: data[reqPartMinSize:],
	}
}

// UnmarshalRequestPart converts a byte buffer into a request message part.
func UnmarshalRequestPart(b []byte) (RequestPart, error) {
	if len(b) < reqPartMinSize {
		return RequestPart{}, errors.Errorf(
			errReqPartDataSize, len(b), reqPartMinSize)
	}
	return mapRequestPart(b), nil
}

// Marshal returns the bytes of the message part.
func (m RequestPart) Marshal() []byte {
	return m.data
}

// GetPartNum returns the index of this part in the message.
func (m RequestPart) GetPartNum() uint8 {
	return m.partNum[0]
}

// SetPartNum sets the part number of the message.
func (m RequestPart) SetPartNum(num uint8) {
	copy(m.partNum, []byte{num})
}

// GetNumParts always returns 0. It is here so that RequestPart adheres to th
// Part interface.
func (m RequestPart) GetNumParts() uint8 {
	return 0
}

// GetContents returns the contents of the message part.
func (m RequestPart) GetContents() []byte {
	return m.contents[:binary.BigEndian.Uint16(m.size)]
}

// GetContentsSize returns the length of the contents.
func (m RequestPart) GetContentsSize() int {
	return int(binary.BigEndian.Uint16(m.size))
}

// GetMaxContentsSize returns the max capacity of the contents.
func (m RequestPart) GetMaxContentsSize() int {
	return len(m.contents)
}

// SetContents sets the contents of the message part. Does not zero out previous
// contents.
func (m RequestPart) SetContents(contents []byte) {
	if len(contents) > len(m.contents) {
		jww.FATAL.Panicf(errReqPartContentsSize, len(contents), len(m.contents))
	}

	binary.BigEndian.PutUint16(m.size, uint16(len(contents)))

	copy(m.contents, contents)
}
