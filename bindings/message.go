///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/interfaces/message"
)

// Message is a message received from the cMix network in the clear
// or that has been decrypted using established E2E keys.
type Message struct {
	r message.Receive
}

//Returns the id of the message
func (m *Message) GetID() []byte {
	return m.r.ID[:]
}

// Returns the message's sender ID, if available
func (m *Message) GetSender() []byte {
	return m.r.Sender.Bytes()
}

// Returns the message's payload/contents
func (m *Message) GetPayload() []byte {
	return m.r.Payload
}

// Returns the message's type
func (m *Message) GetMessageType() int {
	return int(m.r.MessageType)
}

// Returns the message's timestamp in ms
func (m *Message) GetTimestampMS() int64 {
	ts := m.r.Timestamp.UnixNano()
	ts = (ts + 999999) / 1000000
	return ts
}

func (m *Message) GetTimestampNano() int64 {
	return m.r.Timestamp.UnixNano()
}
