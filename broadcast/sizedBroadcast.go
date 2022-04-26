////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

// Message field sizes.
const (
	sizeSize              = 2
	sizedBroadcastMinSize = sizeSize
)

// Error messages.
const (
	// NewSizedBroadcast
	errNewSizedBroadcastMaxSize = "size of payload and its size %d too large to fit in max payload size %d"

	// DecodeSizedBroadcast
	errDecodeSizedBroadcastDataLen = "size of data %d must be greater than %d"
	errDecodeSizedBroadcastSize    = "stated payload size %d larger than provided data %d"
)

/*
+---------------------------+
|   cMix Message Contents   |
+---------+-----------------+
|  Size   |     Payload     |
| 2 bytes | remaining space |
+---------+-----------------+
*/

// NewSizedBroadcast creates a new broadcast with its size information embedded.
// The maxPayloadSize is the maximum size of the resulting payload. Returns an
// error when the sized broadcast cannot fit in the max payload size.
func NewSizedBroadcast(maxPayloadSize int, payload []byte) ([]byte, error) {
	if len(payload)+sizedBroadcastMinSize > maxPayloadSize {
		return nil, errors.Errorf(errNewSizedBroadcastMaxSize,
			len(payload)+sizedBroadcastMinSize, maxPayloadSize)
	}

	b := make([]byte, sizeSize)
	binary.LittleEndian.PutUint16(b, uint16(len(payload)))

	return append(b, payload...), nil
}

// DecodeSizedBroadcast  the data into its original payload of the correct size.
func DecodeSizedBroadcast(data []byte) ([]byte, error) {
	if len(data) < sizedBroadcastMinSize {
		return nil, errors.Errorf(
			errDecodeSizedBroadcastDataLen, len(data), sizedBroadcastMinSize)
	}

	size := binary.LittleEndian.Uint16(data[:sizeSize])
	if int(size) > len(data[sizeSize:]) {
		return nil, errors.Errorf(
			errDecodeSizedBroadcastSize, size, len(data[sizeSize:]))
	}

	return data[sizeSize : size+sizeSize], nil
}

// MaxSizedBroadcastPayloadSize returns the maximum payload size in a sized
// broadcast for the given out message max payload size.
func MaxSizedBroadcastPayloadSize(maxPayloadSize int) int {
	return maxPayloadSize - sizedBroadcastMinSize
}
