package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Defines message structure.  Based the "Basic Message Structure" doc
// Defining rangings in slices in go is inclusive for the beginning but
// exclusive for the end, so the END consts are one more then the final
// index.
const (
	IV_LEN			uint64 = 72
	IV_START		uint64 = 0
	IV_END			uint64 = IV_LEN

	PAYLOAD_LEN   	uint64 = 495
	PAYLOAD_START	uint64 = IV_END
	PAYLOAD_END		uint64 = PAYLOAD_START+PAYLOAD_LEN

	SID_LEN   		uint64 = 8
	SID_START		uint64 = PAYLOAD_END
	SID_END			uint64 = SID_START+SID_LEN

	RID_LEN 		uint64 = 504
	RID_START		uint64 = IV_END
	RID_END			uint64 = RID_START+RID_LEN

	TOTAL_LEN 		uint64 = 512


)

//TODO: generate ranges programmatic


type MessageBytes struct{
	Payload 	 *cyclic.Int
	PayloadMIC	 *cyclic.Int
	Recipient 	 *cyclic.Int
	RecipientMIC *cyclic.Int
}

// Structure which contains a message payload and the sender in an easily
// accessible format
type Message struct {
	senderID 			*cyclic.Int
	payload  			*cyclic.Int
	recipientID 		*cyclic.Int
	payloadInitVect		*cyclic.Int
	recipientInitVect	*cyclic.Int
}

// Makes a new message for a sender and a
func NewMessage(sender, recipient uint64, text string) []*Message {

	if sender ==0 {
		panic("Invalid sender id")
		return nil
	}

	if recipient ==0 {
		panic("Invalid recipient id")
		return nil
	}

	// Split the payload into multiple sub-payloads if it is longer than the
	// maximum allowed
	payload := []byte(text)

	var payloadLst [][]byte

	for true{
		if uint64(len(payload))<PAYLOAD_LEN {
			payloadLst = append(payloadLst, payload[0:])
			break
		}else {
			payloadLst = append(payloadLst, payload[0:PAYLOAD_LEN])
			payload = payload[PAYLOAD_LEN:]
		}
	}

	// create a message for every sub-payload
	var messageList []*Message

	for i:=0;i<len(payloadLst);i++{
		msg := &Message{
			cyclic.NewInt(int64(sender)),
			cyclic.NewIntFromBytes(payloadLst[i]),
			cyclic.NewInt(int64(recipient)),
			cyclic.NewInt(0),
			cyclic.NewInt(0),
		}
		messageList = append(messageList,msg)
	}
	return messageList
}

// These functions return pointers to the internal data in MessageCyclic
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetSenderID() *cyclic.Int{
	return m.senderID
}

func (m *Message)GetPayload() *cyclic.Int{
	return m.payload
}

func (m *Message)GetRecipientID() *cyclic.Int{
	return m.recipientID
}

func (m *Message)GetPayloadInitVector() *cyclic.Int{
	return m.payloadInitVect
}

func (m *Message)GetRecipientInitVector() *cyclic.Int{
	return m.recipientInitVect
}

func (m *Message) getSenderIDInt() uint64{
	return m.senderID.Uint64()
}

func (m *Message) getRecipientIDInt() uint64{
	return m.recipientID.Uint64()
}

func (m *Message) getPayloadString() string{
	return string(m.payload.Bytes())
}

func (m *Message)ConstructMessageBytes() *MessageBytes{

	/*CONSTRUCT MESSAGE PAYLOAD*/
	var messagePayload []byte

	// append the initialization vector
	ivm := m.payloadInitVect.LeftpadBytes(IV_LEN)
	// Set the highest order bit to zero to make the 'blank'
	ivm[0] = ivm[0] & 0x7F

	messagePayload = append(messagePayload, ivm...)

	// append the payload
	messagePayload = append(messagePayload,
		m.payload.LeftpadBytes(PAYLOAD_LEN)...)

	// append the sender id
	messagePayload = append(messagePayload,
		m.senderID.LeftpadBytes(SID_LEN)...)

	/*CONSTRUCT RECIPIENT PAYLOAD*/
	var recipientPayload []byte

	// append the initialization vector
	ivr := m.recipientInitVect.LeftpadBytes(IV_LEN)
	// Set the highest order bit to zero to make the 'blank'
	ivr[0] = ivr[0] & 0x7F

	recipientPayload = append(recipientPayload, ivr...)

	//append the recipientid
	recipientPayload = append(recipientPayload,
		m.recipientID.LeftpadBytes(RID_LEN)...)

	//Create message

	mb := &MessageBytes{
		cyclic.NewIntFromBytes(messagePayload),
		cyclic.NewInt(0),
		cyclic.NewIntFromBytes(recipientPayload),
		cyclic.NewInt(0),
	}

	return mb
}

func (mb *MessageBytes)DeconstructMessageBytes() *Message{
	return &Message{
		cyclic.NewIntFromBytes(mb.Payload.LeftpadBytes(PAYLOAD_LEN)[SID_START:SID_END]),
		cyclic.NewIntFromBytes(mb.Payload.LeftpadBytes(PAYLOAD_LEN)[PAYLOAD_START:PAYLOAD_END]),
		cyclic.NewIntFromBytes(mb.Recipient.LeftpadBytes(PAYLOAD_LEN)[RID_START:RID_END]),
		cyclic.NewIntFromBytes(mb.Payload.LeftpadBytes(PAYLOAD_LEN)[IV_START:IV_END]),
		cyclic.NewIntFromBytes(mb.Recipient.LeftpadBytes(PAYLOAD_LEN)[IV_START:IV_END]),
	}
}

