////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Defines message structure.  Based the "Basic Message Structure" doc
// Defining rangings in slices in go is inclusive for the beginning but
// exclusive for the end, so the END consts are one more then the final
// index.
const (
	TOTAL_LEN 		uint64 = 512

	// Length and Position of the Initialization Vector for both the payload and
	// the recipient
	IV_LEN			uint64 = 9
	IV_START		uint64 = 0
	IV_END			uint64 = IV_LEN

	// Length and Position of message payload
	PAYLOAD_LEN   	uint64 = TOTAL_LEN-SID_LEN-IV_LEN-PMIC_LEN
	PAYLOAD_START	uint64 = IV_END
	PAYLOAD_END		uint64 = PAYLOAD_START+PAYLOAD_LEN

	// Length and Position of the Sender ID in the payload
	SID_LEN   		uint64 = 8
	SID_START		uint64 = PAYLOAD_END
	SID_END			uint64 = SID_START + SID_LEN

	// Length and Position of the Payload MIC
	PMIC_LEN	    uint64 = 8
	PMIC_START		uint64 = SID_END
	PMIC_END		uint64 = PMIC_START+PMIC_LEN

	// Length and Position of the Recipient ID
	RID_LEN 		uint64 = TOTAL_LEN-IV_LEN-RMIC_LEN
	RID_START		uint64 = IV_END
	RID_END			uint64 = RID_START+RID_LEN

	// Length and Position of the Recipient MIC
	RMIC_LEN	    uint64 = 8
	RMIC_START		uint64 = RID_END
	RMIC_END		uint64 = RMIC_START+RMIC_LEN
)

//TODO: generate ranges programmatic

//Holds the payloads once they have been serialized
//MIC stands for Message identification code
type MessageBytes struct{
	Payload 	 *cyclic.Int
	Recipient 	 *cyclic.Int
}

// Structure which contains a message payload and the sender in an easily
// accessible format
type Message struct {
	senderID 			*cyclic.Int
	payload  			*cyclic.Int
	recipientID 		*cyclic.Int
	payloadInitVect		*cyclic.Int
	recipientInitVect	*cyclic.Int
	payloadMIC			*cyclic.Int
	recipientMIC		*cyclic.Int
}

// Makes a new message for a certain sender and recipient
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

	for uint64(len(payload))>PAYLOAD_LEN{
		payloadLst = append(payloadLst, payload[0:PAYLOAD_LEN])
		payload = payload[PAYLOAD_LEN:]
	}
	payloadLst = append(payloadLst, payload)


	// create a message for every sub-payload
	var messageList []*Message

	for i:=0;i<len(payloadLst);i++{
		msg := &Message{
			cyclic.NewInt(int64(sender)),
			cyclic.NewIntFromBytes(payloadLst[i]),
			cyclic.NewInt(int64(recipient)),
			cyclic.NewInt(0),
			cyclic.NewInt(0),
			cyclic.NewInt(0),
			cyclic.NewInt(0),
		}
		messageList = append(messageList,msg)
	}
	return messageList
}

// This function returns a pointer to the sender ID in Message
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetSenderID() *cyclic.Int{
	return m.senderID
}

// This function returns a pointer to the payload in Message
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetPayload() *cyclic.Int{
	return m.payload
}

// This function returns a pointer to the Recipient ID in Message
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetRecipientID() *cyclic.Int{
	return m.recipientID
}

// This function returns a pointer to the Payload Initiliztion Vector in
// Message
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetPayloadInitVector() *cyclic.Int{
	return m.payloadInitVect
}

// This function returns a pointer to the Recipient ID Initilization Vector in
// Message
// This ensures that while the data can be edited, it cant be reallocated
func (m *Message)GetRecipientInitVector() *cyclic.Int{
	return m.recipientInitVect
}

// This function returns the Sender ID as a uint64 from Message
func (m *Message) getSenderIDInt() uint64{
	return m.senderID.Uint64()
}

// This function returns the Recipient ID as a uint64 from Message
func (m *Message) getRecipientIDInt() uint64{
	return m.recipientID.Uint64()
}

// This function returns a Payload String from Message
func (m *Message) GetPayloadString() string{
	return string(m.payload.Bytes())
}

func (m *Message) GetPayloadMIC() *cyclic.Int{
	return m.payloadMIC
}

func (m *Message) GetRecipientMIC() *cyclic.Int{
	return m.recipientMIC
}


//Builds the Serialized MessageBytes from Message
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

	// append the Payload MIC
	messagePayload = append(messagePayload,
		m.payloadMIC.LeftpadBytes(PMIC_LEN)...)

	/*CONSTRUCT RECIPIENT PAYLOAD*/
	var recipientPayload []byte

	// append the initialization vector
	ivr := m.recipientInitVect.LeftpadBytes(IV_LEN)
	// Set the highest order bit to zero to make the 'blank'
	ivr[0] = ivr[0] & 0x7F

	recipientPayload = append(recipientPayload, ivr...)

	//append the recipient id
	recipientPayload = append(recipientPayload,
		m.recipientID.LeftpadBytes(RID_LEN)...)

	//append the recipient MIC
	recipientPayload = append(recipientPayload,
		m.recipientMIC.LeftpadBytes(RMIC_LEN)...)

	//Create message

	mb := &MessageBytes{
		cyclic.NewIntFromBytes(messagePayload),
		cyclic.NewIntFromBytes(recipientPayload),
	}

	return mb
}

//Deserializes MessageBytes
func (mb *MessageBytes)DeconstructMessageBytes() *Message{
	payloadBytes := mb.Payload.LeftpadBytes(TOTAL_LEN)
	recipientBytes := mb.Recipient.LeftpadBytes(TOTAL_LEN)

	return &Message{
		cyclic.NewIntFromBytes(payloadBytes[SID_START:SID_END]),
		cyclic.NewIntFromBytes(payloadBytes[PAYLOAD_START:PAYLOAD_END]),
		cyclic.NewIntFromBytes(recipientBytes[RID_START:RID_END]),
		cyclic.NewIntFromBytes(payloadBytes[IV_START:IV_END]),
		cyclic.NewIntFromBytes(recipientBytes[IV_START:IV_END]),
		cyclic.NewIntFromBytes(payloadBytes[PMIC_START:PMIC_END]),
		cyclic.NewIntFromBytes(recipientBytes[RMIC_START:RMIC_END]),
	}
}

