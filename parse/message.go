package parse

import "gitlab.com/privategrity/client/globals"

type Message struct {
	TypedBody
	Sender globals.UserID
	Receiver globals.UserID
}

// Implement format.MessageInterface, just in case it's useful later
// TODO should return user id type?
func (m *Message) GetSender() []byte {
	return []byte(m.Sender)
}

// TODO this should return a []byte to avoid too much type casting
func (m *Message) GetPayload() string {
	return string(m.Body)
}

func (m *Message) GetRecipient() []byte {
	return []byte(m.Receiver)
}
