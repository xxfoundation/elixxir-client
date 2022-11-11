////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"encoding/binary"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
)

// Sizes of message parts, in bytes.
const (
	numPartsLen     = 1
	typeLen         = catalog.MessageTypeLen
	timestampLen    = 8
	firstPartVerLen = 1
	firstHeaderLen  = headerLen + numPartsLen + typeLen + timestampLen + firstPartVerLen
)

// The current version of the firstMessagePart message format.
const firstMessagePartCurrentVersion = 0

type firstMessagePart struct {
	messagePart
	NumParts  []byte
	Type      []byte
	Timestamp []byte
	Version   []byte // Version of the message format; always the last bit
}

// newFirstMessagePart creates a new firstMessagePart for the passed in
// contents. Does no length checks.
func newFirstMessagePart(mt catalog.MessageType, id uint32, numParts uint8,
	timestamp time.Time, contents []byte, size int) firstMessagePart {

	// Create the message structure
	m := firstMessagePartFromBytes(make([]byte, size))

	// Set the message type
	binary.BigEndian.PutUint32(m.Type, uint32(mt))

	// Set the message ID
	binary.BigEndian.PutUint32(m.Id, id)

	// Set the part number. It is always zero because this is the first part.
	// Because the default is zero this step could be skipped, but keep it in
	// the code for clarity.
	m.Part[0] = 0

	// Set the number of parts to the message
	m.NumParts[0] = numParts

	// Set the timestamp as unix nano
	binary.BigEndian.PutUint64(m.Timestamp, uint64(timestamp.UnixNano()))

	// Set the length of the contents
	binary.BigEndian.PutUint16(m.Len, uint16(len(contents)))

	// Set the contents
	copy(m.Contents[:len(contents)], contents)

	// Set the version number
	m.Version[0] = firstMessagePartCurrentVersion

	return m
}

// Map of firstMessagePart encoding version numbers to their map functions.
var firstMessagePartFromBytesVersions = map[uint8]func([]byte) firstMessagePart{
	firstMessagePartCurrentVersion: firstMessagePartFromBytesVer0,
}

// firstMessagePartFromBytes builds a firstMessagePart mapped to the passed in
// data slice. Mapped by reference; a copy is not made.
func firstMessagePartFromBytes(data []byte) firstMessagePart {

	// Map the data according to its version
	version := data[len(data)-1]
	jww.INFO.Printf("Unsafeversion: %d", version)
	mapFunc, exists := firstMessagePartFromBytesVersions[version]
	if exists {
		return mapFunc(data)
	}

	return firstMessagePart{}
}

func firstMessagePartFromBytesVer0(data []byte) firstMessagePart {
	return firstMessagePart{
		messagePart: messagePart{
			Data:     data,
			Id:       data[:idLen],
			Part:     data[idLen : idLen+partLen],
			Len:      data[idLen+partLen : idLen+partLen+lenLen],
			Contents: data[idLen+partLen+lenLen+numPartsLen+typeLen+timestampLen : len(data)-firstPartVerLen-1],
		},
		NumParts:  data[idLen+partLen+lenLen : idLen+partLen+lenLen+numPartsLen],
		Type:      data[idLen+partLen+lenLen+numPartsLen : idLen+partLen+lenLen+numPartsLen+typeLen],
		Timestamp: data[idLen+partLen+lenLen+numPartsLen+typeLen : idLen+partLen+lenLen+numPartsLen+typeLen+timestampLen],
		Version:   data[len(data)-firstPartVerLen:],
	}
}

// getType returns the message type.
func (m firstMessagePart) getType() catalog.MessageType {
	return catalog.MessageType(binary.BigEndian.Uint32(m.Type))
}

// getNumParts returns the number of message parts.
func (m firstMessagePart) getNumParts() uint8 {
	return m.NumParts[0]
}

// getTimestamp returns the timestamp as a time.Time.
func (m firstMessagePart) getTimestamp() time.Time {
	return time.Unix(0, int64(binary.BigEndian.Uint64(m.Timestamp)))
}

// getVersion returns the version number of the data encoding.
func (m firstMessagePart) getVersion() uint8 {
	return m.Version[0]
}

// bytes returns the serialised message data.
func (m firstMessagePart) bytes() []byte {
	return m.Data
}
