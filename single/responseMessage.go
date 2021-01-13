///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

const (
	partNumLen  = 1
	maxPartsLen = 1
)

type responseMessagePart struct {
	data     []byte // Serial of all contents
	partNum  []byte // Index of message in a series of messages
	maxParts []byte // The number of parts in this message.
	payload  []byte // The encrypted payload
}

// newResponseMessagePart generates a new response message part of the specified
// size.
func newResponseMessagePart(externalPayloadSize int) responseMessagePart {
	if externalPayloadSize < partNumLen+maxPartsLen {
		jww.FATAL.Panicf("Failed to create new single use response message "+
			"part: size of external payload (%d) is too small to contain the "+
			"message part number and max parts (%d)",
			externalPayloadSize, partNumLen+maxPartsLen)
	}

	return mapResponseMessagePart(make([]byte, externalPayloadSize))
}

// mapResponseMessagePart builds a message part mapped to the passed in data.
// It is mapped by reference; a copy is not made.
func mapResponseMessagePart(data []byte) responseMessagePart {
	return responseMessagePart{
		data:     data,
		partNum:  data[:partNumLen],
		maxParts: data[partNumLen : maxPartsLen+partNumLen],
		payload:  data[maxPartsLen+partNumLen:],
	}
}

// unmarshalResponseMessage converts a byte buffer into a response message part.
func unmarshalResponseMessage(b []byte) (responseMessagePart, error) {
	if len(b) < partNumLen+maxPartsLen {
		return responseMessagePart{}, errors.Errorf("Size of passed in bytes "+
			"(%d) is too small to contain the message part number and max "+
			"parts (%d).", len(b), partNumLen+maxPartsLen)
	}
	return mapResponseMessagePart(b), nil
}

// Marshal returns the bytes of the message part.
func (m responseMessagePart) Marshal() []byte {
	return m.data
}

// GetPartNum returns the index of this part in the message.
func (m responseMessagePart) GetPartNum() uint8 {
	return m.partNum[0]
}

// SetPartNum sets the part number of the message.
func (m responseMessagePart) SetPartNum(num uint8) {
	copy(m.partNum, []byte{num})
}

// GetMaxParts returns the number of parts in the message.
func (m responseMessagePart) GetMaxParts() uint8 {
	return m.maxParts[0]
}

// SetMaxParts sets the number of parts in the message.
func (m responseMessagePart) SetMaxParts(max uint8) {
	copy(m.maxParts, []byte{max})
}

// GetPayload returns the encrypted payload of the message part.
func (m responseMessagePart) GetPayload() []byte {
	return m.payload
}

// GetPayloadSize returns the length of the encrypted payload.
func (m responseMessagePart) GetPayloadSize() int {
	return len(m.payload)
}

// SetPayload sets the encrypted payload of the message part.
func (m responseMessagePart) SetPayload(payload []byte) {
	if len(payload) != m.GetPayloadSize() {
		jww.FATAL.Panicf("Failed to set payload of single use response "+
			"message part: size of supplied payload (%d) is different from "+
			"the size of the message payload (%d).",
			len(payload), m.GetPayloadSize()+maxPartsLen)
	}
	copy(m.payload, payload)
}
