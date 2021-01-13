///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type transmitMessage struct {
	data    []byte // Serial of all contents
	pubKey  []byte
	payload []byte // The encrypted payload containing reception ID and contents
}

// newTransmitMessage generates a new empty message for transmission that is the
// size of the specified external payload.
func newTransmitMessage(externalPayloadSize, pubKeySize int) transmitMessage {
	if externalPayloadSize < pubKeySize {
		jww.FATAL.Panicf("Payload size of single use transmission message "+
			"(%d) too small to contain the public key (%d).",
			externalPayloadSize, pubKeySize)
	}

	return mapTransmitMessage(make([]byte, externalPayloadSize), pubKeySize)
}

// mapTransmitMessage builds a message mapped to the passed in data. It is
// mapped by reference; a copy is not made.
func mapTransmitMessage(data []byte, pubKeySize int) transmitMessage {
	return transmitMessage{
		data:    data,
		pubKey:  data[:pubKeySize],
		payload: data[pubKeySize:],
	}
}

// unmarshalTransmitMessage unmarshalls a byte slice into a transmitMessage. An
// error is returned if the slice is not large enough for the public key size.
func unmarshalTransmitMessage(b []byte, pubKeySize int) (transmitMessage, error) {
	if len(b) < pubKeySize {
		return transmitMessage{}, errors.Errorf("Length of marshaled bytes "+
			"(%d) too small to contain public key (%d).", len(b), pubKeySize)
	}

	return mapTransmitMessage(b, pubKeySize), nil
}

// Marshal returns the serialised data of a transmitMessage.
func (m transmitMessage) Marshal() []byte {
	return m.data
}

// GetPubKey returns the public key that is part of the given group.
func (m transmitMessage) GetPubKey(grp *cyclic.Group) *cyclic.Int {
	return grp.NewIntFromBytes(m.pubKey)
}

// GetPubKeySize returns the length of the public key.
func (m transmitMessage) GetPubKeySize() int {
	return len(m.pubKey)
}

// SetPubKey saves the public key to the message as bytes.
func (m transmitMessage) SetPubKey(pubKey *cyclic.Int) {
	copy(m.pubKey, pubKey.LeftpadBytes(uint64(len(m.pubKey))))
}

// GetPayload returns the encrypted payload of the message.
func (m transmitMessage) GetPayload() []byte {
	return m.payload
}

// GetPayloadSize returns the length of the encrypted payload.
func (m transmitMessage) GetPayloadSize() int {
	return len(m.payload)
}

// SetPayload saves the supplied bytes as the payload of the message, if the
// size is correct.
func (m transmitMessage) SetPayload(b []byte) {
	if len(b) != len(m.payload) {
		jww.FATAL.Panicf("Size of payload of single use transmission message "+
			"(%d) is not the same as the size of the supplied payload (%d).",
			len(m.payload), len(b))
	}

	copy(m.payload, b)
}

const numSize = 1

// transmitMessagePayload is the structure of transmitMessage's payload.
type transmitMessagePayload struct {
	data     []byte // Serial of all contents
	rid      []byte // Response reception ID
	num      []byte // Number of messages expected in response
	contents []byte
}

// newTransmitMessage generates a new empty message for transmission that is the
// size of the specified payload, which should match the size of the payload in
// the corresponding transmitMessage.
func newTransmitMessagePayload(payloadSize int) transmitMessagePayload {
	if payloadSize < id.ArrIDLen+numSize {
		jww.FATAL.Panicf("Size of single use transmission message payload "+
			"(%d) too small to contain the reception ID (%d) + the message "+
			"count (%d).",
			payloadSize, id.ArrIDLen, numSize)
	}

	return mapTransmitMessagePayload(make([]byte, payloadSize))
}

// mapTransmitMessagePayload builds a message payload mapped to the passed in
// data. It is mapped by reference; a copy is not made.
func mapTransmitMessagePayload(data []byte) transmitMessagePayload {
	return transmitMessagePayload{
		data:     data,
		rid:      data[:id.ArrIDLen],
		num:      data[id.ArrIDLen : id.ArrIDLen+numSize],
		contents: data[id.ArrIDLen+numSize:],
	}
}

// unmarshalTransmitMessagePayload unmarshalls a byte slice into a
// transmitMessagePayload. An error is returned if the slice is not large enough
// for the reception ID and message count.
func unmarshalTransmitMessagePayload(b []byte) (transmitMessagePayload, error) {
	if len(b) < id.ArrIDLen+numSize {
		return transmitMessagePayload{}, errors.Errorf("Length of marshaled "+
			"bytes(%d) too small to contain the reception ID (%d) + the "+
			"message count (%d).", len(b), id.ArrIDLen, numSize)
	}

	return mapTransmitMessagePayload(b), nil
}

// Marshal returns the serialised data of a transmitMessagePayload.
func (mp transmitMessagePayload) Marshal() []byte {
	return mp.data
}

// GetRID returns the reception ID.
func (mp transmitMessagePayload) GetRID() *id.ID {
	rid, err := id.Unmarshal(mp.rid)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal transmission ID of single use "+
			"transmission message payload: %+v", err)
	}

	return rid
}

// SetRID sets the reception ID of the payload.
func (mp transmitMessagePayload) SetRID(rid *id.ID) {
	copy(mp.rid, rid.Marshal())
}

// GetCount returns the number of messages expected in response.
func (mp transmitMessagePayload) GetCount() uint8 {
	return mp.num[0]
}

// SetCount sets the number of expected messages.
func (mp transmitMessagePayload) SetCount(num uint8) {
	copy(mp.num, []byte{num})
}

// GetContents returns the payload's contents.
func (mp transmitMessagePayload) GetContents() []byte {
	return mp.contents
}

// GetContentsSize returns the length of payload's contents.
func (mp transmitMessagePayload) GetContentsSize() int {
	return len(mp.contents)
}

// SetContents saves the contents to the payload, if the size is correct.
func (mp transmitMessagePayload) SetContents(b []byte) {
	if len(b) != len(mp.contents) {
		jww.FATAL.Panicf("Size of content of single use transmission message "+
			"payload (%d) is not the same as the size of the supplied "+
			"contents (%d).", len(mp.contents), len(b))
	}

	copy(mp.contents, b)
}
