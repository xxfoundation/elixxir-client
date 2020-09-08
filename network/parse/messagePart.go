package parse

import (
	"encoding/binary"
)

const idLen = 4
const partLen = 1
const lenLen = 2
const headerLen = idLen + partLen + lenLen

type messagePart struct {
	Data     []byte
	Id       []byte
	Part     []byte
	Len      []byte
	Contents []byte
}

//creates a new message part for the passed in contents. Does no length checks
func newMessagePart(id uint32, part uint8, contents []byte) messagePart {
	//create the message structure
	data := make([]byte, len(contents)+headerLen)
	m := MessagePartFromBytes(data)

	//add the message ID to the message
	binary.BigEndian.PutUint32(m.Id, id)

	//set the message part number
	m.Part[0] = part

	//set the contents length
	binary.BigEndian.PutUint16(m.Len, uint16(len(contents)))

	//copy the contents into the message
	copy(m.Contents[:len(contents)], contents)
	return m
}

// Builds a Message part mapped to the passed in data slice. Mapped by
// reference, a copy is not made.
func MessagePartFromBytes(data []byte) messagePart {
	m := messagePart{
		Data:     data,
		Id:       data[:idLen],
		Part:     data[idLen : idLen+partLen],
		Len:      data[idLen+partLen : idLen+partLen+lenLen],
		Contents: data[idLen+partLen+lenLen:],
	}
	return m
}

func (m messagePart) GetID() uint32 {
	return binary.BigEndian.Uint32(m.Id)
}

func (m messagePart) GetPart() uint8 {
	return m.Part[0]
}

func (m messagePart) GetContents() []byte {
	return m.Contents
}

func (m messagePart) GetSizedContents() []byte {
	size := m.GetContentsLength()
	return m.Contents[:size]
}

func (m messagePart) GetContentsLength() int {
	return int(binary.BigEndian.Uint16(m.Len))
}

func (m messagePart) Bytes() []byte {
	return m.Data
}

