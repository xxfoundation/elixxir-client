package parse

import (
	"encoding/binary"
)

const idLen = 4
const partLen = 1
const headerLen = idLen + partLen

type messagePart struct {
	Data     []byte
	Id       []byte
	Part     []byte
	Contents []byte
}

func newMessagePart(id uint32, part uint8, contents []byte) messagePart {
	data := make([]byte, len(contents)+headerLen)
	m := MessagePartFromBytes(data)
	binary.BigEndian.PutUint32(m.Id, id)
	m.Part[0] = part
	copy(m.Contents[:len(contents)], contents)
	//set the first bit to 1 to denote this is not a raw message
	data[0] |= 0b10000000
	return m
}

func MessagePartFromBytes(data []byte) messagePart {
	m := messagePart{
		Data:     data,
		Id:       data[:idLen],
		Part:     data[idLen : idLen+partLen],
		Contents: data[idLen+partLen+numPartsLen+typeLen:],
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

func (m messagePart) Bytes() []byte {
	return m.Data
}

func (m messagePart) IsRaw() bool {
	return isRaw(m.Data[0])
}
