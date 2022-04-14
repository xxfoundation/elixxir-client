///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

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
+-------------------------------------------------------------------------------------+
|                                CMIX Message Contents                                |
+-----------+------------+------------------------------------------------------------+
|  Version  |   pubKey   |                  payload (RequestPayload)                  |
|  1 byte   | pubKeySize |              externalPayloadSize - pubKeySize              |
+-----------+------------+----------+---------+------------------+---------+----------+
                         |  Tag FP  |  nonce  | maxResponseParts |  size   | contents |
                         | 16 bytes | 8 bytes |      1 byte      | 2 bytes | variable |
                         +----------+---------+------------------+---------+----------+
*/

const transmitMessageVersion = 0
const transmitMessageVersionSize = 1

type Request struct {
	data    []byte // Serial of all contents
	version []byte
	pubKey  []byte
	payload []byte // The encrypted payload containing reception ID and contents
}

// NewRequest generates a new empty message for transmission that is the
// size of the specified external payload.
func NewRequest(externalPayloadSize, pubKeySize int) Request {
	if externalPayloadSize < pubKeySize {
		jww.FATAL.Panicf("Payload size of single-use transmission message "+
			"(%d) too small to contain the public key (%d).",
			externalPayloadSize, pubKeySize)
	}

	tm := mapRequest(make([]byte, externalPayloadSize), pubKeySize)
	tm.version[0] = transmitMessageVersion

	return tm
}

func GetRequestPayloadSize(externalPayloadSize, pubKeySize int) uint {
	return uint(externalPayloadSize - transmitMessageVersionSize - pubKeySize)
}

// mapRequest builds a message mapped to the passed in data. It is
// mapped by reference; a copy is not made.
func mapRequest(data []byte, pubKeySize int) Request {
	return Request{
		data:    data,
		version: data[:transmitMessageVersionSize],
		pubKey:  data[transmitMessageVersionSize : transmitMessageVersionSize+pubKeySize],
		payload: data[transmitMessageVersionSize+pubKeySize:],
	}
}

// UnmarshalRequest unmarshalls a byte slice into a Request. An
// error is returned if the slice is not large enough for the public key size.
func UnmarshalRequest(b []byte, pubKeySize int) (Request, error) {
	if len(b) < pubKeySize {
		return Request{}, errors.Errorf("Length of marshaled bytes "+
			"(%d) too small to contain public key (%d).", len(b), pubKeySize)
	}

	return mapRequest(b, pubKeySize), nil
}

// Marshal returns the serialised data of a Request.
func (m Request) Marshal() []byte {
	return m.data
}

// GetPubKey returns the public key that is part of the given group.
func (m Request) GetPubKey(grp *cyclic.Group) *cyclic.Int {
	return grp.NewIntFromBytes(m.pubKey)
}

// Version returns the version of the message.
func (m Request) Version() uint8 {
	return m.version[0]
}

// GetPubKeySize returns the length of the public key.
func (m Request) GetPubKeySize() int {
	return len(m.pubKey)
}

// SetPubKey saves the public key to the message as bytes.
func (m Request) SetPubKey(pubKey *cyclic.Int) {
	copy(m.pubKey, pubKey.LeftpadBytes(uint64(len(m.pubKey))))
}

// GetPayload returns the encrypted payload of the message.
func (m Request) GetPayload() []byte {
	return m.payload
}

// GetPayloadSize returns the length of the encrypted payload.
func (m Request) GetPayloadSize() int {
	return len(m.payload)
}

// SetPayload saves the supplied bytes as the payload of the message, if the
// size is correct.
func (m Request) SetPayload(b []byte) {
	if len(b) != len(m.payload) {
		jww.FATAL.Panicf("Size of payload of single-use transmission message "+
			"(%d) is not the same as the size of the supplied payload (%d).",
			len(m.payload), len(b))
	}

	copy(m.payload, b)
}

const (
	nonceSize            = 8
	numRequestPartsSize  = 1
	maxResponsePartsSize = 1
	sizeSize             = 2
	transmitPlMinSize    = nonceSize + numRequestPartsSize + maxResponsePartsSize + sizeSize
)

// RequestPayload is the structure of Request's payload.
type RequestPayload struct {
	data             []byte // Serial of all contents
	nonce            []byte
	numRequestParts  []byte // Number of parts in the request, currently always 1
	maxResponseParts []byte // Max number of messages expected in response
	size             []byte // Size of the contents
	contents         []byte
}

// NewRequestPayload generates a new empty message for transmission that is the
// size of the specified payload, which should match the size of the payload in
// the corresponding Request.
func NewRequestPayload(payloadSize int, payload []byte, maxMsgs uint8) RequestPayload {
	if payloadSize < transmitPlMinSize {
		jww.FATAL.Panicf("Size of single-use transmission message payload "+
			"(%d) too small to contain the necessary data (%d).",
			payloadSize, transmitPlMinSize)
	}

	// Map fields to data
	mp := mapRequestPayload(make([]byte, payloadSize))

	mp.SetMaxResponseParts(maxMsgs)
	mp.SetContents(payload)
	return mp
}

func GetRequestContentsSize(payloadSize uint) uint {
	return payloadSize - transmitPlMinSize
}

// mapRequestPayload builds a message payload mapped to the passed in
// data. It is mapped by reference; a copy is not made.
func mapRequestPayload(data []byte) RequestPayload {
	mp := RequestPayload{
		data:             data,
		nonce:            data[:nonceSize],
		numRequestParts:  data[nonceSize : nonceSize+numRequestPartsSize],
		maxResponseParts: data[nonceSize+numRequestPartsSize : nonceSize+maxResponsePartsSize+numRequestPartsSize],
		size:             data[nonceSize+numRequestPartsSize+maxResponsePartsSize : transmitPlMinSize],
		contents:         data[transmitPlMinSize:],
	}
	mp.numRequestParts[0] = 1

	return mp
}

// UnmarshalRequestPayload unmarshalls a byte slice into a
// RequestPayload. An error is returned if the slice is not large enough
// for the reception ID and message count.
func UnmarshalRequestPayload(b []byte) (RequestPayload, error) {
	if len(b) < transmitPlMinSize {
		return RequestPayload{}, errors.Errorf("Length of marshaled "+
			"bytes(%d) too small to contain the necessary data (%d).",
			len(b), transmitPlMinSize)
	}

	return mapRequestPayload(b), nil
}

// Marshal returns the serialised data of a RequestPayload.
func (mp RequestPayload) Marshal() []byte {
	return mp.data
}

// GetRID generates the reception ID from the bytes of the payload.
func (mp RequestPayload) GetRID(pubKey *cyclic.Int) *id.ID {
	return singleUse.NewRecipientID(pubKey, mp.Marshal())
}

// GetNonce returns the nonce as an uint64.
func (mp RequestPayload) GetNonce() uint64 {
	return binary.BigEndian.Uint64(mp.nonce)
}

// SetNonce generates a random nonce from the RNG. An error is returned if the
// reader fails.
func (mp RequestPayload) SetNonce(rng io.Reader) error {
	if _, err := rng.Read(mp.nonce); err != nil {
		return errors.Errorf("failed to generate nonce: %+v", err)
	}

	return nil
}

// GetMaxResponseParts returns the number of messages expected in response.
func (mp RequestPayload) GetMaxResponseParts() uint8 {
	return mp.maxResponseParts[0]
}

// SetMaxResponseParts sets the number of expected messages.
func (mp RequestPayload) SetMaxResponseParts(num uint8) {
	copy(mp.maxResponseParts, []byte{num})
}

// GetNumRequestParts returns the number of messages expected in the request.
func (mp RequestPayload) GetNumRequestParts() uint8 {
	return mp.numRequestParts[0]
}

// SetNumRequestParts sets the number of expected messages.
func (mp RequestPayload) SetNumRequestParts(num uint8) {
	copy(mp.numRequestParts, []byte{num})
}

// GetContents returns the payload's contents.
func (mp RequestPayload) GetContents() []byte {
	return mp.contents[:binary.BigEndian.Uint16(mp.size)]
}

// GetContentsSize returns the length of payload's contents.
func (mp RequestPayload) GetContentsSize() int {
	return int(binary.BigEndian.Uint16(mp.size))
}

// GetMaxContentsSize returns the max capacity of the contents.
func (mp RequestPayload) GetMaxContentsSize() int {
	return len(mp.contents)
}

// SetContents saves the contents to the payload, if the size is correct. Does
// not zero out previous content.
func (mp RequestPayload) SetContents(contents []byte) {
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
func (mp RequestPayload) String() string {
	return fmt.Sprintf("Data: %x [nonce: %x, "+
		"maxResponseParts: %x, size: %x, content: %x]", mp.data,
		mp.nonce, mp.maxResponseParts, mp.size, mp.contents)

}
