package parse

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context/message"
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

func newFirstMessagePart(mt message.Type, id uint32, part uint8, numParts uint8,
	timestamp time.Time, contents []byte) firstMessagePart {
	data := make([]byte, len(contents)+firstHeaderLen)

	m := FirstMessagePartFromBytes(data)
	binary.BigEndian.PutUint32(m.Type, uint32(mt))
	binary.BigEndian.PutUint32(m.Id, id)
	m.Part[0] = part
	m.NumParts[0] = numParts

	timestampBytes, err := timestamp.MarshalBinary()
	if err != nil {
		jww.FATAL.Panicf("Failed to create firstMessagePart: %s", err.Error())
	}

	copy(m.Timestamp, timestampBytes)
	copy(m.Contents[:len(contents)], contents)
	//set the first bit to 1 to denote this is not a raw message
	data[0] |= 0b10000000
	return m
}

func FirstMessagePartFromBytes(data []byte) firstMessagePart {
	m := firstMessagePart{
		messagePart: messagePart{
			Data:     data,
			Id:       data[:idLen],
			Part:     data[idLen : idLen+partLen],
			Contents: data[idLen+partLen+numPartsLen+typeLen+timestampLen:],
		},
		NumParts:  data[idLen+partLen : idLen+partLen+numPartsLen],
		Type:      data[idLen+partLen+numPartsLen : idLen+partLen+numPartsLen+typeLen],
		Timestamp: data[idLen+partLen+numPartsLen+typeLen : idLen+partLen+numPartsLen+typeLen+timestampLen],
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
