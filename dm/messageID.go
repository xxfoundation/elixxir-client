////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"golang.org/x/crypto/blake2b"
)

const (
	// MessageIDLen is the length of a MessageID.
	MessageIDLen  = 32
	messageIDSalt = "DirectMessageIdSalt"
)

// Error messages.
const (
	unmarshalMessageIdDataLenErr = "received %d bytes when %d bytes required"
)

// MessageID is the unique identifier of a channel message.
type MessageID [MessageIDLen]byte

// DeriveDirectMessageID hashes the parts relevant to a direct message
// to create a shared message ID between both parties.
// Round ID, Pubey, and DMToken is not hashed, so this is not replay
// resistant from a malicious attacker, but DMs prevent parties without the
// keys of one half the connection from participating.
func DeriveDirectMessageID(msg *DirectMessage) MessageID {
	h, err := blake2b.New256(nil)
	if err != nil {
		jww.FATAL.Panicf("Failed to get Hash: %+v", err)
	}
	h.Write(msg.GetPayload())

	pty := make([]byte, 4)
	binary.LittleEndian.PutUint32(pty, msg.GetPayloadType())
	h.Write(pty)

	h.Write([]byte(msg.GetNickname()))
	// Note: It is imperative to include this so messages are not repeated
	h.Write(msg.Nonce)

	ts := make([]byte, 8)
	binary.LittleEndian.PutUint64(ts, uint64(msg.GetLocalTimestamp()))
	h.Write(ts)

	midBytes := h.Sum(nil)
	mid := MessageID{}
	copy(mid[:], midBytes)
	return mid
}

// Equals checks if two message IDs are the same.
//
// Not constant time.
func (mid MessageID) Equals(mid2 MessageID) bool {
	return hmac.Equal(mid[:], mid2[:])
}

// String returns a base64 encoded MessageID for debugging. This function
// adheres to the fmt.Stringer interface.
func (mid MessageID) String() string {
	return "DMMsgID-" + base64.StdEncoding.EncodeToString(mid[:])
}

// Bytes returns a copy of the bytes in the message.
func (mid MessageID) Bytes() []byte {
	return mid.Marshal()
}

// DeepCopy returns a copy Message ID
func (mid MessageID) DeepCopy() MessageID {
	return mid
}

// Marshal marshals the MessageID into a byte slice.
func (mid MessageID) Marshal() []byte {
	bytesCopy := make([]byte, len(mid))
	copy(bytesCopy, mid[:])
	return bytesCopy
}

// UnmarshalMessageID unmarshalls the byte slice into a MessageID.
func UnmarshalMessageID(data []byte) (MessageID, error) {
	mid := MessageID{}
	if len(data) != MessageIDLen {
		return mid, errors.Errorf(
			unmarshalMessageIdDataLenErr, len(data), MessageIDLen)
	}

	copy(mid[:], data)
	return mid, nil
}

// MarshalJSON handles the JSON marshaling of the MessageID. This function
// adheres to the [json.Marshaler] interface.
func (mid MessageID) MarshalJSON() ([]byte, error) {
	// Note: Any changes to the output of this function can break storage in
	// higher levels. Take care not to break the consistency test.
	return json.Marshal(mid.Marshal())
}

// UnmarshalJSON handles the JSON unmarshalling of the MessageID. This function
// adheres to the [json.Unmarshaler] interface.
func (mid *MessageID) UnmarshalJSON(b []byte) error {
	var buff []byte
	if err := json.Unmarshal(b, &buff); err != nil {
		return err
	}

	newMID, err := UnmarshalMessageID(buff)
	if err != nil {
		return err
	}

	*mid = newMID

	return nil
}
