package globals

import (
	"encoding/binary"
)

// Defines message structure.  Based the "Basic Message Structure" doc
const (
	BLANK_INDEX uint32 = 0
	BLANK_LEN   uint32 = 1

	PAYLOAD_START uint32 = 1
	PAYLOAD_END   uint32 = 504
	PAYLOAD_LEN   uint32 = 503

	SID_START uint32 = 504
	SID_END   uint32 = 512
	SID_LEN   uint32 = 8

	TOTAL_LEN uint32 = BLANK_LEN + PAYLOAD_LEN + SID_LEN
)

// Structure which contains a message payload and the sender in an easily
// accessible format
type Message struct {
	senderID uint64
	payload  [PAYLOAD_LEN]byte
}

// Makes a new message for a sender and a
func NewMessage(sid uint64, pl string) *Message {
	payload := []byte(pl)

	if uint32(len(payload)) > PAYLOAD_LEN {
		payload = payload[0:PAYLOAD_LEN]
	} else if uint32(len(payload)) < PAYLOAD_LEN {
		tmp := make([]byte, PAYLOAD_LEN-uint32(len(payload)))
		payload = append(tmp, payload...)
	}

	message := Message{senderID: sid}

	copy(message.payload[:], payload)

	return &message
}

// Takes a message byte array and splits it into its component parts
func ConstructMessage(msg *[]byte) *Message {

	if uint32(len(*msg)) != TOTAL_LEN || (*msg)[0] != 0 {
		return nil
	}

	payload := (*msg)[PAYLOAD_START:PAYLOAD_END]
	sid := binary.BigEndian.Uint64((*msg)[SID_START:SID_END])

	message := Message{senderID: sid}

	copy(message.payload[:], payload)

	return &message
}

// Takes a message and builds a message byte array
func (message *Message) DeconstructMessage() *[]byte {

	sidarr := make([]byte, SID_LEN)

	binary.BigEndian.PutUint64(sidarr, message.senderID)

	rtnslc := make([]byte, BLANK_LEN)
	rtnslc[BLANK_INDEX] = 0x00
	rtnslc = append(rtnslc, message.payload[:]...)
	rtnslc = append(rtnslc, sidarr...)

	return &rtnslc
}

// Returns a copy of the payload
func (message *Message) GetPayload() *[]byte {
	var rntpayload []byte

	copy(rntpayload, message.payload[:])

	return &rntpayload
}

// Returns a string of the payload
func (message *Message) GetStringPayload() string {
	return string(message.payload[:])
}

// Return the sender of the payload
func (message *Message) GetSenderID() uint64 {
	return message.senderID
}
