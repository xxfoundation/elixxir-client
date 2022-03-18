///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/xx_network/primitives/id"
	"io"
)

/*
+------------------------------------------------------------------------------+
|                              CMIX Message Contents                           |
+------------+--------------------------------------------             --------+
|   Version  |   pubKey   |          payload (transmitMessagePayload)          |
|    1 byte  | pubKeySize |          externalPayloadSize - pubKeySize          |
+------------+------------+----------+---------+----------+---------+----------+
                          |  Tag FP  |  nonce  | maxParts |  size   | contents |
                          | 16 bytes | 8 bytes |  1 byte  | 2 bytes | variable |
                          +----------+---------+----------+---------+----------+
*/

const transmitMessageVersion = 0
const transmitMessageVersionSize = 1

type transmitMessage struct {
	data    []byte // Serial of all contents
	version []byte
	pubKey  []byte
	payload []byte // The encrypted payload containing reception ID and contents
}

// newTransmitMessage generates a new empty message for transmission that is the
// size of the specified external payload.
func newTransmitMessage(externalPayloadSize, pubKeySize int) transmitMessage {
	if externalPayloadSize < pubKeySize {
		jww.FATAL.Panicf("Payload size of single-use transmission message "+
			"(%d) too small to contain the public key (%d).",
			externalPayloadSize, pubKeySize)
	}

	tm := mapTransmitMessage(make([]byte, externalPayloadSize), pubKeySize)
	tm.version[0] = transmitMessageVersion

	return tm
}

// mapTransmitMessage builds a message mapped to the passed in data. It is
// mapped by reference; a copy is not made.
func mapTransmitMessage(data []byte, pubKeySize int) transmitMessage {
	return transmitMessage{
		data:    data,
		version: data[:transmitMessageVersionSize],
		pubKey:  data[transmitMessageVersionSize : transmitMessageVersionSize+pubKeySize],
		payload: data[transmitMessageVersionSize+pubKeySize:],
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

// Version returns the version of the message.
func (m transmitMessage) Version() uint8 {
	return m.version[0]
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
		jww.FATAL.Panicf("Size of payload of single-use transmission message "+
			"(%d) is not the same as the size of the supplied payload (%d).",
			len(m.payload), len(b))
	}

	copy(m.payload, b)
}

const (
	tagFPSize         = singleUse.TagFpSize
	nonceSize         = 8
	maxPartsSize      = 1
	sizeSize          = 2
	transmitPlMinSize = tagFPSize + nonceSize + maxPartsSize + sizeSize
)

// transmitMessagePayload is the structure of transmitMessage's payload.
type transmitMessagePayload struct {
	data     []byte // Serial of all contents
	tagFP    []byte // Tag fingerprint identifies the type of message
	nonce    []byte
	maxParts []byte // Max number of messages expected in response
	size     []byte // Size of the contents
	contents []byte
}

// newTransmitMessage generates a new empty message for transmission that is the
// size of the specified payload, which should match the size of the payload in
// the corresponding transmitMessage.
func newTransmitMessagePayload(payloadSize int) transmitMessagePayload {
	if payloadSize < transmitPlMinSize {
		jww.FATAL.Panicf("Size of single-use transmission message payload "+
			"(%d) too small to contain the necessary data (%d).",
			payloadSize, transmitPlMinSize)
	}

	// Map fields to data
	mp := mapTransmitMessagePayload(make([]byte, payloadSize))

	return mp
}

// mapTransmitMessagePayload builds a message payload mapped to the passed in
// data. It is mapped by reference; a copy is not made.
func mapTransmitMessagePayload(data []byte) transmitMessagePayload {
	mp := transmitMessagePayload{
		data:     data,
		tagFP:    data[:tagFPSize],
		nonce:    data[tagFPSize : tagFPSize+nonceSize],
		maxParts: data[tagFPSize+nonceSize : tagFPSize+nonceSize+maxPartsSize],
		size:     data[tagFPSize+nonceSize+maxPartsSize : transmitPlMinSize],
		contents: data[transmitPlMinSize:],
	}

	return mp
}

// unmarshalTransmitMessagePayload unmarshalls a byte slice into a
// transmitMessagePayload. An error is returned if the slice is not large enough
// for the reception ID and message count.
func unmarshalTransmitMessagePayload(b []byte) (transmitMessagePayload, error) {
	if len(b) < transmitPlMinSize {
		return transmitMessagePayload{}, errors.Errorf("Length of marshaled "+
			"bytes(%d) too small to contain the necessary data (%d).",
			len(b), transmitPlMinSize)
	}

	return mapTransmitMessagePayload(b), nil
}

// Marshal returns the serialised data of a transmitMessagePayload.
func (mp transmitMessagePayload) Marshal() []byte {
	return mp.data
}

// GetRID generates the reception ID from the bytes of the payload.
func (mp transmitMessagePayload) GetRID(pubKey *cyclic.Int) *id.ID {
	return singleUse.NewRecipientID(pubKey, mp.Marshal())
}

// GetTagFP returns the tag fingerprint.
func (mp transmitMessagePayload) GetTagFP() singleUse.TagFP {
	return singleUse.UnmarshalTagFP(mp.tagFP)
}

// SetTagFP sets the tag fingerprint.
func (mp transmitMessagePayload) SetTagFP(tagFP singleUse.TagFP) {
	copy(mp.tagFP, tagFP.Bytes())
}

// GetNonce returns the nonce as a uint64.
func (mp transmitMessagePayload) GetNonce() uint64 {
	return binary.BigEndian.Uint64(mp.nonce)
}

// SetNonce generates a random nonce from the RNG. An error is returned if the
// reader fails.
func (mp transmitMessagePayload) SetNonce(rng io.Reader) error {
	if _, err := rng.Read(mp.nonce); err != nil {
		return errors.Errorf("failed to generate nonce: %+v", err)
	}

	return nil
}

// GetMaxParts returns the number of messages expected in response.
func (mp transmitMessagePayload) GetMaxParts() uint8 {
	return mp.maxParts[0]
}

// SetMaxParts sets the number of expected messages.
func (mp transmitMessagePayload) SetMaxParts(num uint8) {
	copy(mp.maxParts, []byte{num})
}

// GetContents returns the payload's contents.
func (mp transmitMessagePayload) GetContents() []byte {
	return mp.contents[:binary.BigEndian.Uint16(mp.size)]
}

// GetContentsSize returns the length of payload's contents.
func (mp transmitMessagePayload) GetContentsSize() int {
	return int(binary.BigEndian.Uint16(mp.size))
}

// GetMaxContentsSize returns the max capacity of the contents.
func (mp transmitMessagePayload) GetMaxContentsSize() int {
	return len(mp.contents)
}

// SetContents saves the contents to the payload, if the size is correct. Does
// not zero out previous content.
func (mp transmitMessagePayload) SetContents(contents []byte) {
	if len(contents) > len(mp.contents) {
		jww.FATAL.Panicf("Failed to set contents of single-use transmission "+
			"message: max size of message content (%d) is smaller than the "+
			"size of the supplied contents (%d).",
			len(mp.contents), len(contents))
	}

	binary.BigEndian.PutUint16(mp.size, uint16(len(contents)))

	copy(mp.contents, contents)
}

// String returns the contents for printing adhering to the stringer interface.
func (mp transmitMessagePayload) String() string {
	return fmt.Sprintf("Data: %x [tagFP: %x, nonce: %x, "+
		"maxParts: %x, size: %x, content: %x]", mp.data, mp.tagFP,
		mp.nonce, mp.maxParts, mp.size, mp.contents)

}
