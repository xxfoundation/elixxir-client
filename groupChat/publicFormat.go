////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/group"
)

// Sizes of marshaled data, in bytes.
const (
	saltLen      = group.SaltLen
	publicMinLen = saltLen
)

// Error messages
const (
	newPublicSizeErr       = "max message size %d < %d minimum required"
	unmarshalPublicSizeErr = "size of data %d < %d minimum required"
)

// publicMsg is contains the salt and encrypted data in a group message.
//
// +---------------------+
// |        data         |
// +----------+----------+
// |   salt   | payload  |
// | 32 bytes | variable |
// +----------+----------+
type publicMsg struct {
	data    []byte // Serial of all the parts of the message
	salt    []byte // 256-bit sender salt
	payload []byte // Encrypted internalMsg
}

// newPublicMsg creates a new publicMsg of size maxDataSize. An error is
// returned if the maxDataSize is smaller than the minimum newPublicMsg size.
func newPublicMsg(maxDataSize int) (publicMsg, error) {
	if maxDataSize < publicMinLen {
		return publicMsg{},
			errors.Errorf(newPublicSizeErr, maxDataSize, publicMinLen)
	}

	return mapPublicMsg(make([]byte, maxDataSize)), nil
}

// mapPublicMsg maps all the parts of the publicMsg to the passed in data.
func mapPublicMsg(data []byte) publicMsg {
	return publicMsg{
		data:    data,
		salt:    data[:saltLen],
		payload: data[saltLen:],
	}
}

// unmarshalPublicMsg unmarshal the data into an publicMsg.  An error is
// returned if the data length is smaller than the minimum allowed size.
func unmarshalPublicMsg(data []byte) (publicMsg, error) {
	if len(data) < publicMinLen {
		return publicMsg{},
			errors.Errorf(unmarshalPublicSizeErr, len(data), publicMinLen)
	}

	return mapPublicMsg(data), nil
}

// Marshal returns the serial of the publicMsg.
func (pm publicMsg) Marshal() []byte {
	return pm.data
}

// GetSalt returns the 256-bit salt.
func (pm publicMsg) GetSalt() [group.SaltLen]byte {
	var salt [group.SaltLen]byte
	copy(salt[:], pm.salt)
	return salt
}

// SetSalt sets the 256-bit salt.
func (pm publicMsg) SetSalt(salt [group.SaltLen]byte) {
	copy(pm.salt, salt[:])
}

// GetPayload returns the payload truncated to the correct size.
func (pm publicMsg) GetPayload() []byte {
	return pm.payload
}

// SetPayload sets the payload and saves it size.
func (pm publicMsg) SetPayload(payload []byte) {
	copy(pm.payload, payload)
}

// GetPayloadSize returns the maximum size of the payload.
func (pm publicMsg) GetPayloadSize() int {
	return len(pm.payload)
}

// String prints a string representation of publicMsg. This functions satisfies
// the fmt.Stringer interface.
func (pm publicMsg) String() string {
	salt := "<nil>"
	if len(pm.salt) > 0 {
		salt = base64.StdEncoding.EncodeToString(pm.salt)
	}

	payload := "<nil>"
	if len(pm.payload) > 0 {
		payload = fmt.Sprintf("%q", pm.GetPayload())
	}

	return "{salt:" + salt + ", payload:" + payload + "}"
}
