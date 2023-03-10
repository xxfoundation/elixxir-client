////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

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
	"strconv"
	"strings"
)

// Error messages.
const (
	// NewRequest
	errNewReqPayloadSize = "[SU] Failed to create new single-use request " +
		"message: external payload size (%d) is smaller than the public key " +
		"size (%d)."

	// UnmarshalRequest
	errReqDataSize = "size of data (%d) must be at least %d"

	// Request.SetPayload
	errReqPayloadSize = "[SU] Failed to set payload of single-use request " +
		"message: size of the supplied payload (%d) is larger than the max " +
		"message size (%d)."

	// NewRequestPayload
	errNewReqPayloadPayloadSize = "[SU] Failed to create new single-use " +
		"request payload message: payload size (%d) is smaller than the " +
		"minimum message size for a request payload (%d)."

	// UnmarshalRequestPayload
	errReqPayloadDataSize = "size of data (%d) must be at least %d"

	// RequestPayload.SetNonce
	errSetReqPayloadNonce = "failed to generate nonce: %+v"

	// RequestPayload.SetContents
	errReqPayloadContentsSize = "[SU] Failed to set contents of single-use " +
		"request payload message: size of the supplied contents (%d) is " +
		"larger than the max message size (%d)."
)

/*
+--------------------------------------------------------------------------------------------+
|                                   cMix Message Contents                                    |
+-----------+------------+-------------------------------------------------------------------+
|  Version  |   pubKey   |                      payload (RequestPayload)                     |
|  1 byte   | pubKeySize |             externalPayloadSize - 1 byte - pubKeySize             |
+-----------+------------+---------+-----------------+------------------+---------+----------+
                         |  nonce  | numRequestParts | maxResponseParts |  size   | contents |
                         | 8 bytes |     1 byte      |      1 byte      | 2 bytes | variable |
                         +---------+-----------------+------------------+---------+----------+
*/

const requestVersion = 0
const requestVersionSize = 1

type Request struct {
	data    []byte // Serial of all contents
	version []byte
	pubKey  []byte
	payload []byte // The encrypted payload containing reception ID and contents
}

// NewRequest generates a new empty message for a request that is the size of
// the specified external payload.
func NewRequest(externalPayloadSize, pubKeySize int) Request {
	if externalPayloadSize < pubKeySize {
		jww.FATAL.Panicf(errNewReqPayloadSize, externalPayloadSize, pubKeySize)
	}

	tm := mapRequest(make([]byte, externalPayloadSize), pubKeySize)
	tm.version[0] = requestVersion

	return tm
}

// GetRequestPayloadSize returns the size of the payload for the given external
// payload size and public key size.
func GetRequestPayloadSize(externalPayloadSize, pubKeySize int) int {
	return externalPayloadSize - requestVersionSize - pubKeySize
}

// mapRequest builds a message mapped to the passed in data. It is
// mapped by reference; a copy is not made.
func mapRequest(data []byte, pubKeySize int) Request {
	return Request{
		data:    data,
		version: data[:requestVersionSize],
		pubKey:  data[requestVersionSize : requestVersionSize+pubKeySize],
		payload: data[requestVersionSize+pubKeySize:],
	}
}

// UnmarshalRequest unmarshalls a byte slice into a Request. An
// error is returned if the slice is not large enough for the public key size.
func UnmarshalRequest(b []byte, pubKeySize int) (Request, error) {
	if len(b) < pubKeySize {
		return Request{}, errors.Errorf(errReqDataSize, len(b), pubKeySize)
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
func (m Request) SetPayload(payload []byte) {
	if len(payload) != len(m.payload) {
		jww.FATAL.Panicf(errReqPayloadSize, len(m.payload), len(payload))
	}

	copy(m.payload, payload)
}

/*
+-------------------------------------------------------------------+
|                          Request payload                          |
+---------+-----------------+------------------+---------+----------+
|  nonce  | numRequestParts | maxResponseParts |  size   | contents |
| 8 bytes |     1 byte      |      1 byte      | 2 bytes | variable |
+---------+-----------------+------------------+---------+----------+
*/

const (
	nonceSize            = 8
	numRequestPartsSize  = 1
	maxResponsePartsSize = 1
	sizeSize             = 2
	requestMinSize       = nonceSize + numRequestPartsSize + maxResponsePartsSize + sizeSize
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

// NewRequestPayload generates a new empty message for request that is the size
// of the specified payload, which should match the size of the payload in the
// corresponding Request.
func NewRequestPayload(payloadSize int, payload []byte, maxMsgs uint8) RequestPayload {
	if payloadSize < requestMinSize {
		jww.FATAL.Panicf(
			errNewReqPayloadPayloadSize, payloadSize, requestMinSize)
	}

	// Map fields to data
	mp := mapRequestPayload(make([]byte, payloadSize))

	mp.SetMaxResponseParts(maxMsgs)
	mp.SetContents(payload)
	return mp
}

// GetRequestContentsSize returns the size of the contents of a RequestPayload
// given the payload size.
func GetRequestContentsSize(payloadSize int) int {
	return payloadSize - requestMinSize
}

// mapRequestPayload builds a message payload mapped to the passed in
// data. It is mapped by reference; a copy is not made.
func mapRequestPayload(data []byte) RequestPayload {
	mp := RequestPayload{
		data:             data,
		nonce:            data[:nonceSize],
		numRequestParts:  data[nonceSize : nonceSize+numRequestPartsSize],
		maxResponseParts: data[nonceSize+numRequestPartsSize : nonceSize+maxResponsePartsSize+numRequestPartsSize],
		size:             data[nonceSize+numRequestPartsSize+maxResponsePartsSize : requestMinSize],
		contents:         data[requestMinSize:],
	}

	return mp
}

// UnmarshalRequestPayload unmarshalls a byte slice into a RequestPayload. An
// error is returned if the slice is not large enough for the reception ID and
// message count.
func UnmarshalRequestPayload(b []byte) (RequestPayload, error) {
	if len(b) < requestMinSize {
		return RequestPayload{},
			errors.Errorf(errReqPayloadDataSize, len(b), requestMinSize)
	}

	return mapRequestPayload(b), nil
}

// Marshal returns the serialised data of a RequestPayload.
func (mp RequestPayload) Marshal() []byte {
	return mp.data
}

// GetRecipientID generates the recipient ID from the bytes of the payload.
func (mp RequestPayload) GetRecipientID(pubKey *cyclic.Int) *id.ID {
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
		return errors.Errorf(errSetReqPayloadNonce, err)
	}

	return nil
}

// GetMaxResponseParts returns the maximum number of response messages allowed.
func (mp RequestPayload) GetMaxResponseParts() uint8 {
	return mp.maxResponseParts[0]
}

// SetMaxResponseParts sets the maximum number of response messages allowed.
func (mp RequestPayload) SetMaxResponseParts(num uint8) {
	copy(mp.maxResponseParts, []byte{num})
}

// GetNumRequestParts returns the number of messages in the request.
func (mp RequestPayload) GetNumRequestParts() uint8 {
	return mp.numRequestParts[0]
}

// GetNumParts returns the number of messages in the request. This function
// wraps GetMaxRequestParts so that RequestPayload adheres to the Part
// interface.
func (mp RequestPayload) GetNumParts() uint8 {
	return mp.GetNumRequestParts()
}

// SetNumRequestParts sets the number of messages in the request.
func (mp RequestPayload) SetNumRequestParts(num uint8) {
	copy(mp.numRequestParts, []byte{num})
}

// GetPartNum always returns 0 since it is the first message.
func (mp RequestPayload) GetPartNum() uint8 {
	return 0
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
		jww.FATAL.Panicf(
			errReqPayloadContentsSize, len(contents), len(mp.contents))
	}

	binary.BigEndian.PutUint16(mp.size, uint16(len(contents)))

	copy(mp.contents, contents)
}

// String returns the contents of a RequestPayload as a human-readable string.
// This function adheres to the fmt.Stringer interface.
func (mp RequestPayload) String() string {
	str := []string{
		"nonce: " + strconv.Itoa(int(mp.GetNonce())),
		"numRequestParts: " + strconv.Itoa(int(mp.GetNumRequestParts())),
		"maxResponseParts: " + strconv.Itoa(int(mp.GetMaxResponseParts())),
		"size: " + strconv.Itoa(mp.GetContentsSize()),
		"contents: " + fmt.Sprintf("%q", mp.GetContents()),
	}

	return "{" + strings.Join(str, ", ") + "}"

}
