///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/interfaces/message"
)

// Message is a message received from the cMix network in the clear
// or that has been decrypted using established E2E keys.
type Message struct {
	r message.Receive
}

// GetID returns the id of the message
func (m *Message) GetID() []byte {
	return m.r.ID[:]
}

// GetSender returns the message's sender ID, if available
func (m *Message) GetSender() []byte {
	return m.r.Sender.Bytes()
}

// GetPayload returns the message's payload/contents
func (m *Message) GetPayload() []byte {
	return m.r.Payload
}

// GetMessageType returns the message's type
func (m *Message) GetMessageType() int {
	return int(m.r.MessageType)
}

// GetTimestampMS returns the message's timestamp in milliseconds
func (m *Message) GetTimestampMS() int64 {
	ts := m.r.Timestamp.UnixNano()
	ts = (ts + 500000) / 1000000
	return ts
}

// GetTimestampNano returns the message's timestamp in nanoseconds
func (m *Message) GetTimestampNano() int64 {
	return m.r.Timestamp.UnixNano()
}

// GetRoundTimestampMS returns the message's round timestamp in milliseconds
func (m *Message) GetRoundTimestampMS() int64 {
	ts := m.r.RoundTimestamp.UnixNano()
	ts = (ts + 999999) / 1000000
	return ts
}

// GetRoundTimestampNano returns the message's round timestamp in nanoseconds
func (m *Message) GetRoundTimestampNano() int64 {
	return m.r.RoundTimestamp.UnixNano()
}

// GetRoundId returns the message's round ID
func (m *Message) GetRoundId() int64 {
	return int64(m.r.RoundId)
}

// GetRoundURL returns the message's round URL
func (m *Message) GetRoundURL() string {
	return getRoundURL(m.r.RoundId)
}
