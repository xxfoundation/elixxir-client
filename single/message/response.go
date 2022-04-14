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

const (
	partNumLen                 = 1
	maxPartsLen                = 1
	responseMinSize            = receptionMessageVersionLen + partNumLen + maxPartsLen + sizeSize
	receptionMessageVersion    = 0
	receptionMessageVersionLen = 1
)

/*
+-----------------------------------------------------------+
|                   CMIX Message Contents                   |
+---------+------------------+---------+---------+----------+
| version | maxResponseParts |  size   | partNum | contents |
| 1 bytes |      1 byte      | 2 bytes | 1 bytes | variable |
+---------+------------------+---------+---------+----------+
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
	if externalPayloadSize < responseMinSize {
		jww.FATAL.Panicf("Failed to create new single-use response message "+
			"part: size of external payload (%d) is too small to contain the "+
			"message part number and max parts (%d)",
			externalPayloadSize, responseMinSize)
	}

	rmp := mapResponsePart(make([]byte, externalPayloadSize))
	rmp.version[0] = receptionMessageVersion
	return rmp
}

// mapResponsePart builds a message part mapped to the passed in data.
// It is mapped by reference; a copy is not made.
func mapResponsePart(data []byte) ResponsePart {
	return ResponsePart{
		data:     data,
		version:  data[:receptionMessageVersionLen],
		partNum:  data[receptionMessageVersionLen : receptionMessageVersionLen+partNumLen],
		maxParts: data[receptionMessageVersionLen+partNumLen : receptionMessageVersionLen+maxPartsLen+partNumLen],
		size:     data[receptionMessageVersionLen+maxPartsLen+partNumLen : responseMinSize],
		contents: data[responseMinSize:],
	}
}

// UnmarshalResponse converts a byte buffer into a response message part.
func UnmarshalResponse(b []byte) (ResponsePart, error) {
	if len(b) < responseMinSize {
		return ResponsePart{}, errors.Errorf("Size of passed in bytes "+
			"(%d) is too small to contain the message part number and max "+
			"parts (%d).", len(b), responseMinSize)
	}
	return mapResponsePart(b), nil
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
		jww.FATAL.Panicf("Failed to set contents of single-use response "+
			"message part: max size of message contents (%d) is smaller than "+
			"the size of the supplied contents (%d).",
			len(m.contents), len(contents))
	}

	binary.BigEndian.PutUint16(m.size, uint16(len(contents)))

	copy(m.contents, contents)
}
