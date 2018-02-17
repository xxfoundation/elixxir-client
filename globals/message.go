package globals

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
)

// Defines message structure.  Based the "Basic Message Structure" doc
// Defining rangings in slices in go is inclusive for the beginning but
// exclusive for the end, so the END consts are one more then the final
// index.
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

	RID_LEN uint32 = 504
)

//TODO: generate ranges programmatic

// Structure which contains a message payload and the sender in an easily
// accessible format
type Message struct {
	senderID uint64
	payload  [PAYLOAD_LEN]byte
}

// Makes a new message for a sender and a
func NewMessage(sid uint64, messageString string) *Message {
	payload := []byte(messageString)

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
func ConstructMessageFromBytes(msg *[]byte) *Message {

	if uint32(len(*msg)) != TOTAL_LEN || (*msg)[0] != 0 {
		jww.ERROR.Printf("Invalid message bytes passed! Got %v expected %v",
			len(*msg), TOTAL_LEN)
		panic("Invalid message bytes passed")
		return nil
	}

	payload := (*msg)[PAYLOAD_START:PAYLOAD_END]
	// Endianness is defined by the hardware in go, if this inst handled
	// manually, there could be a mismatch in the resulting byte array
	sid := binary.BigEndian.Uint64((*msg)[SID_START:SID_END])

	message := Message{senderID: sid}

	copy(message.payload[:], payload)

	return &message
}

// Takes a message and builds a message byte array
func (message *Message) DeconstructMessageToBytes() *[]byte {

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

func GenerateRecipientIDBytes(rid uint64) *[]byte {
	ridarr := make([]byte, SID_LEN)

	binary.BigEndian.PutUint64(ridarr, rid)

	ridbytes := make([]byte, RID_LEN)

	ridbytes = append(ridbytes, ridarr...)

	return &ridbytes
}
