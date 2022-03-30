///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package parse

import (
	"encoding/binary"
)

// Sizes of message parts, in bytes.
const (
	idLen      = 4
	partLen    = 1
	lenLen     = 2
	partVerLen = 1
	headerLen  = idLen + partLen + lenLen + partVerLen
)

// The current version of the messagePart message format.
const messagePartCurrentVersion = 0

type messagePart struct {
	Data     []byte
	Id       []byte
	Part     []byte
	Len      []byte
	Contents []byte
	Version  []byte // Version of the message format; always the last bit
}

// newMessagePart creates a new messagePart for the passed in contents. Does no
// length checks.
func newMessagePart(id uint32, part uint8, contents []byte) messagePart {
	// Create the message structure
	data := make([]byte, len(contents)+headerLen)
	m := messagePartFromBytes(data)

	// Set the message ID
	binary.BigEndian.PutUint32(m.Id, id)

	// Set the message part number
	m.Part[0] = part

	// Set the contents length
	binary.BigEndian.PutUint16(m.Len, uint16(len(contents)))

	// Copy the contents into the message
	copy(m.Contents[:len(contents)], contents)

	// Set the version number
	m.Version[0] = messagePartCurrentVersion

	return m
}

// Map of messagePart encoding version numbers to their map functions.
var messagePartFromBytesVersions = map[uint8]func([]byte) messagePart{
	messagePartCurrentVersion: messagePartFromBytesVer0,
}

// messagePartFromBytes builds a messagePart mapped to the passed in data slice.
// Mapped by reference; a copy is not made.
func messagePartFromBytes(data []byte) messagePart {

	// Map the data according to its version
	version := data[len(data)-1]
	mapFunc, exists := messagePartFromBytesVersions[version]
	if exists {
		return mapFunc(data)
	}

	return messagePart{}
}

func messagePartFromBytesVer0(data []byte) messagePart {
	return messagePart{
		Data:     data,
		Id:       data[:idLen],
		Part:     data[idLen : idLen+partLen],
		Len:      data[idLen+partLen : idLen+partLen+lenLen],
		Contents: data[idLen+partLen+lenLen : len(data)-partVerLen],
		Version:  data[len(data)-partVerLen:],
	}
}

// getID returns the message ID.
func (m messagePart) getID() uint32 {
	return binary.BigEndian.Uint32(m.Id)
}

// getPart returns the message part number.
func (m messagePart) getPart() uint8 {
	return m.Part[0]
}

// getContents returns the entire contents slice.
func (m messagePart) getContents() []byte {
	return m.Contents
}

// getSizedContents returns the contents truncated to include only stored data.
func (m messagePart) getSizedContents() []byte {
	return m.Contents[:m.getContentsLength()]
}

// getContentsLength returns the length of the data in the contents.
func (m messagePart) getContentsLength() int {
	return int(binary.BigEndian.Uint16(m.Len))
}

// bytes returns the serialised message data.
func (m messagePart) bytes() []byte {
	return m.Data
}
