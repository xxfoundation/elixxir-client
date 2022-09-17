////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"time"
)

// Sizes of marshaled data, in bytes.
const (
	timestampLen           = 8
	idLen                  = id.ArrIDLen
	internalPayloadSizeLen = 2
	internalMinLen         = timestampLen + idLen + internalPayloadSizeLen
)

// Error messages
const (
	newInternalSizeErr       = "max message size %d < %d minimum required"
	unmarshalInternalSizeErr = "size of data %d < %d minimum required"
)

// internalMsg is the internal, unencrypted data in a group message.
//
// +-------------------------------------------+
// |                    data                   |
// +-----------+----------+---------+----------+
// | timestamp | senderID |  size   | payload  |
// |  8 bytes  | 32 bytes | 2 bytes | variable |
// +-----------+----------+---------+----------+
type internalMsg struct {
	data      []byte // Serial of all the parts of the message
	timestamp []byte // 64-bit Unix time timestamp stored in nanoseconds
	senderID  []byte // 264-bit sender ID
	size      []byte // Size of the payload
	payload   []byte // Message contents
}

// newInternalMsg creates a new internalMsg of size maxDataSize. An error is
// returned if the maxDataSize is smaller than the minimum internalMsg size.
func newInternalMsg(maxDataSize int) (internalMsg, error) {
	if maxDataSize < internalMinLen {
		return internalMsg{},
			errors.Errorf(newInternalSizeErr, maxDataSize, internalMinLen)
	}

	return mapInternalMsg(make([]byte, maxDataSize)), nil
}

// mapInternalMsg maps all the parts of the internalMsg to the passed in data.
func mapInternalMsg(data []byte) internalMsg {
	return internalMsg{
		data:      data,
		timestamp: data[:timestampLen],
		senderID:  data[timestampLen : timestampLen+idLen],
		size:      data[timestampLen+idLen : timestampLen+idLen+internalPayloadSizeLen],
		payload:   data[timestampLen+idLen+internalPayloadSizeLen:],
	}
}

// unmarshalInternalMsg unmarshal the data into an internalMsg. An error is
// returned if the data length is smaller than the minimum allowed size.
func unmarshalInternalMsg(data []byte) (internalMsg, error) {
	if len(data) < internalMinLen {
		return internalMsg{},
			errors.Errorf(unmarshalInternalSizeErr, len(data), internalMinLen)
	}

	return mapInternalMsg(data), nil
}

// Marshal returns the serial of the internalMsg.
func (im internalMsg) Marshal() []byte {
	return im.data
}

// GetTimestamp returns the timestamp as a time.Time.
func (im internalMsg) GetTimestamp() time.Time {
	return time.Unix(0, int64(binary.LittleEndian.Uint64(im.timestamp)))
}

// SetTimestamp converts the time.Time to Unix nano and save as bytes.
func (im internalMsg) SetTimestamp(t time.Time) {
	binary.LittleEndian.PutUint64(im.timestamp, uint64(t.UnixNano()))
}

// GetSenderID returns the sender ID bytes as an id.ID.
func (im internalMsg) GetSenderID() (*id.ID, error) {
	return id.Unmarshal(im.senderID)
}

// SetSenderID sets the sender ID.
func (im internalMsg) SetSenderID(sid *id.ID) {
	copy(im.senderID, sid.Marshal())
}

// GetPayload returns the payload truncated to the correct size.
func (im internalMsg) GetPayload() []byte {
	return im.payload[:im.GetPayloadSize()]
}

// SetPayload sets the payload and saves it size.
func (im internalMsg) SetPayload(payload []byte) {
	// Save size of payload
	binary.LittleEndian.PutUint16(im.size, uint16(len(payload)))

	// Save payload
	copy(im.payload, payload)
}

// GetPayloadSize returns the length of the content in the payload.
func (im internalMsg) GetPayloadSize() int {
	return int(binary.LittleEndian.Uint16(im.size))
}

// GetPayloadMaxSize returns the maximum size of the payload.
func (im internalMsg) GetPayloadMaxSize() int {
	return len(im.payload)
}

// String prints a string representation of internalMsg. This functions
// satisfies the fmt.Stringer interface.
func (im internalMsg) String() string {
	timestamp := "<nil>"
	if len(im.timestamp) > 0 {
		timestamp = im.GetTimestamp().String()
	}

	senderID := "<nil>"
	if sid, _ := im.GetSenderID(); sid != nil {
		senderID = sid.String()
	}

	size := "<nil>"
	if len(im.size) > 0 {
		size = strconv.Itoa(im.GetPayloadSize())
	}

	payload := "<nil>"
	if len(im.size) > 0 {
		payload = fmt.Sprintf("%q", im.GetPayload())
	}

	return "{timestamp:" + timestamp + ", senderID:" + senderID +
		", size:" + size + ", payload:" + payload + "}"
}
