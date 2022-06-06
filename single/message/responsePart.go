///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Error messages.
const (
	// NewResponsePart
	errResPartPayloadSize = "[SU] Failed to create new single-use response " +
		"message part: external payload size (%d) is smaller than the " +
		"minimum message size for a response part (%d)."

	// UnmarshalResponsePart
	errResPartDataSize = "size of data (%d) must be at least %d"

	// ResponsePart.SetContents
	errResPartContentsSize = "[SU] Failed to set contents of single-use " +
		"response message part: size of the supplied contents (%d) is larger " +
		"than the max message size (%d)."
)

// Sizes of fields.
const (
	resPartVersionLen  = 1
	resPartPartNumLen  = 1
	resPartMaxPartsLen = 1
	resPartSizeLen     = 2
	resPartMinSize     = resPartVersionLen + resPartPartNumLen + resPartMaxPartsLen + resPartSizeLen
)

// The version number for the ResponsePart message.
const responsePartVersion = 0

/*
+---------------------------------------------------+
|               cMix Message Contents               |
+---------+---------+----------+---------+----------+
| version | partNum | maxParts |  size   | contents |
|  1 byte |  1 byte |  1 byte  | 2 bytes | variable |
+---------+---------+----------+---------+----------+
*/

type ResponsePart struct {
	data     []byte // Serial of all contents
	version  []byte // Version of the message
	partNum  []byte // Index of message in a series of messages
	maxParts []byte // The number of parts in this message.
	size     []byte // Size of the contents
	contents []byte // The encrypted contents
}

// NewResponsePart generates a new response message part of the specified size.
func NewResponsePart(externalPayloadSize int) ResponsePart {
	if externalPayloadSize < resPartMinSize {
		jww.FATAL.Panicf(
			errResPartPayloadSize, externalPayloadSize, resPartMinSize)
	}

	rmp := mapResponsePart(make([]byte, externalPayloadSize))
	rmp.version[0] = responsePartVersion
	return rmp
}

// mapResponsePart builds a message part mapped to the passed in data.
// It is mapped by reference; a copy is not made.
func mapResponsePart(data []byte) ResponsePart {
	return ResponsePart{
		data:     data,
		version:  data[:resPartVersionLen],
		partNum:  data[resPartVersionLen : resPartVersionLen+resPartPartNumLen],
		maxParts: data[resPartVersionLen+resPartPartNumLen : resPartVersionLen+resPartMaxPartsLen+resPartPartNumLen],
		size:     data[resPartVersionLen+resPartMaxPartsLen+resPartPartNumLen : resPartMinSize],
		contents: data[resPartMinSize:],
	}
}

// UnmarshalResponsePart converts a byte buffer into a response message part.
func UnmarshalResponsePart(data []byte) (ResponsePart, error) {
	if len(data) < resPartMinSize {
		return ResponsePart{}, errors.Errorf(
			errResPartDataSize, len(data), resPartMinSize)
	}
	return mapResponsePart(data), nil
}

// Marshal returns the bytes of the message part.
func (m ResponsePart) Marshal() []byte {
	return m.data
}

// GetPartNum returns the index of this part in the message.
func (m ResponsePart) GetPartNum() uint8 {
	return m.partNum[0]
}

// SetPartNum sets the part number of the message.
func (m ResponsePart) SetPartNum(num uint8) {
	copy(m.partNum, []byte{num})
}

// GetNumParts returns the number of parts in the message.
func (m ResponsePart) GetNumParts() uint8 {
	return m.maxParts[0]
}

// SetNumParts sets the number of parts in the message.
func (m ResponsePart) SetNumParts(max uint8) {
	copy(m.maxParts, []byte{max})
}

// GetContents returns the contents of the message part.
func (m ResponsePart) GetContents() []byte {
	return m.contents[:binary.BigEndian.Uint16(m.size)]
}

// GetContentsSize returns the length of the contents.
func (m ResponsePart) GetContentsSize() int {
	return int(binary.BigEndian.Uint16(m.size))
}

// GetMaxContentsSize returns the max capacity of the contents.
func (m ResponsePart) GetMaxContentsSize() int {
	return len(m.contents)
}

// SetContents sets the contents of the message part. Does not zero out previous
// contents.
func (m ResponsePart) SetContents(contents []byte) {
	if len(contents) > len(m.contents) {
		jww.FATAL.Panicf(errResPartContentsSize, len(contents), len(m.contents))
	}

	binary.BigEndian.PutUint16(m.size, uint16(len(contents)))

	copy(m.contents, contents)
}
