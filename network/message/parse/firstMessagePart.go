package parse

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/interfaces/message"
	"time"
)

const numPartsLen = 1
const typeLen = message.TypeLen
const timestampLen = 15
const firstHeaderLen = headerLen + numPartsLen + typeLen + timestampLen

type firstMessagePart struct {
	messagePart
	NumParts  []byte
	Type      []byte
	Timestamp []byte
}

//creates a new first message part for the passed in contents. Does no length checks
func newFirstMessagePart(mt message.Type, id uint32, numParts uint8,
	timestamp time.Time, contents []byte) firstMessagePart {
	//create the message structure
	data := make([]byte, len(contents)+firstHeaderLen)
	m := FirstMessagePartFromBytes(data)

	//Put the message type in the message
	binary.BigEndian.PutUint32(m.Type, uint32(mt))

	//Add the message ID
	binary.BigEndian.PutUint32(m.Id, id)

	// Add the part number to the message, its always zero because this is the
	// first part. Because the default is zero this step could be skipped, but\
	// keep it in the code for clarity
	m.Part[0] = 0

	// Add the number of parts to the message
	m.NumParts[0] = numParts

	//Serialize and add the timestamp to the payload
	timestampBytes, err := timestamp.MarshalBinary()
	if err != nil {
		jww.FATAL.Panicf("Failed to create firstMessagePart: %s", err.Error())
	}
	copy(m.Timestamp, timestampBytes)

	//set the contents length
	binary.BigEndian.PutUint16(m.Len, uint16(len(contents)))

	//add the contents to the payload
	copy(m.Contents[:len(contents)], contents)

	return m
}

// Builds a first message part mapped to the passed in data slice. Mapped by
// reference, a copy is not made.
func FirstMessagePartFromBytes(data []byte) firstMessagePart {
	m := firstMessagePart{
		messagePart: messagePart{
			Data:     data,
			Id:       data[:idLen],
			Part:     data[idLen : idLen+partLen],
			Len:      data[idLen+partLen : idLen+partLen+lenLen],
			Contents: data[idLen+partLen+numPartsLen+typeLen+timestampLen+lenLen:],
		},
		NumParts:  data[idLen+partLen+lenLen : idLen+partLen+numPartsLen+lenLen],
		Type:      data[idLen+partLen+numPartsLen+lenLen : idLen+partLen+numPartsLen+typeLen+lenLen],
		Timestamp: data[idLen+partLen+numPartsLen+typeLen+lenLen : idLen+partLen+numPartsLen+typeLen+timestampLen+lenLen],
	}
	return m
}

func (m firstMessagePart) GetType() message.Type {
	return message.Type(binary.BigEndian.Uint32(m.Type))
}

func (m firstMessagePart) GetNumParts() uint8 {
	return m.NumParts[0]
}

func (m firstMessagePart) GetTimestamp() (time.Time, error) {
	var t time.Time
	err := t.UnmarshalBinary(m.Timestamp)
	return t, err
}

func (m firstMessagePart) Bytes() []byte {
	return m.Data
}
