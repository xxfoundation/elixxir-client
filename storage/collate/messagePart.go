package collate

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/context/message"
)

const typeLen = message.TypeLen
const idLen = 4
const partLen = 1
const numPartsLen = 1
const headerLen = typeLen + idLen + partLen + numPartsLen

type messagePart struct {
	Data     []byte
	Type     []byte
	Id       []byte
	Part     []byte
	NumParts []byte
	Contents []byte
}

func newMessage(mt message.Type, id uint32, part uint8, numParts uint8, contents []byte) messagePart {
	data := make([]byte, len(contents)+headerLen)

	m := Unmarshal(data)

	binary.BigEndian.PutUint32(m.Type, uint32(mt))
	binary.BigEndian.PutUint32(m.Id, id)
	m.Part[0] = part
	m.NumParts[0] = numParts
	copy(m.Contents, contents)
	return m
}

func Unmarshal(data []byte) messagePart {
	m := messagePart{
		Data:     data,
		Type:     data[:typeLen],
		Id:       data[typeLen : typeLen+idLen],
		Part:     data[typeLen+idLen : typeLen+idLen+partLen],
		NumParts: data[typeLen+idLen+partLen : typeLen+idLen+partLen+numPartsLen],
		Contents: data[typeLen+idLen+partLen+numPartsLen:],
	}
	return m
}

func (m messagePart) GetType() message.Type {
	return message.Type(binary.BigEndian.Uint32(m.Type))
}

func (m messagePart) GetID() uint32 {
	return binary.BigEndian.Uint32(m.Id)
}

func (m messagePart) GetPart() uint8 {
	return m.Part[0]
}

func (m messagePart) GetNumParts() uint8 {
	return m.NumParts[0]
}

func (m messagePart) GetContents() []byte {
	return m.Contents
}

func (m messagePart) Marshal() []byte {
	return m.Data
}
