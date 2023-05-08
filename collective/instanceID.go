////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/base64"
	"io"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv"
)

const (
	instanceIDLength  = 8
	instanceIDVersion = 0
	instanceIDKey     = "ThisInstanceID"
)

var (
	ErrEmptyInstance                = errors.New("empty instance ID")
	ErrShortRead                    = errors.New("short read generating instance ID")
	ErrIncorrectSize                = errors.New("incorrect instance ID size")
	ErrInstanceIDAlreadyInitialized = errors.New("instanceID already " +
		"initialized, cannot do a second instanceID")
)

// InstanceID is a random, URL Safe, base64 string generated when an
// xxdk client is initialized.  It is used to differentiate different
// instances of clients using the same cMix identity. We represent
// it internally as a byte array.
type InstanceID [instanceIDLength]byte

// String implements the [Stringer.String] interface function
func (i InstanceID) String() string {
	return base64.RawURLEncoding.EncodeToString(i[:])
}

// MarshalText implements the [encoding.MarshalText] interface function
func (i InstanceID) MarshalText() (text []byte, err error) {
	return i[:], nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface function
func (i *InstanceID) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		// Error if we got an empty instance id entry
		return ErrEmptyInstance
	} else if len(text) != instanceIDLength {
		// Error if it is the wrong size
		return errors.Wrapf(ErrIncorrectSize,
			"%d != %d", instanceIDLength, len(text))
	}
	copy(i[:], text)
	return nil
}

// Cmp determines which instance id is greater assuming they are numbers.
// -1 - i and j are lesser
//
//	0 - i and j are equal
//	1 - i is greater than j
func (i InstanceID) Cmp(j InstanceID) int {
	for x := 0; x < instanceIDLength; x++ {
		if i[x] > j[x] {
			return 1
		} else if i[x] < j[x] {
			return -1
		}
	}
	return 0
}

// Equals determines if the two instances are the same.
func (i InstanceID) Equals(j InstanceID) bool {
	for x := 0; x < instanceIDLength; x++ {
		if i[x] != j[x] {
			return false
		}
	}
	return true
}

// InitInstanceID creates the InstanceID and returns it if it doesnt exist.
// If it already exists, Init will return an error.
func InitInstanceID(kv ekv.KeyValue, rng io.Reader) (InstanceID, error) {

	//check if the instance ID already exists. If it does refuse to operate
	idBytes, err := kv.GetBytes(instanceIDKey)
	if err != nil && !ekv.Exists(err) {
		return InstanceID{}, err
	}
	if idBytes != nil {
		return InstanceID{}, errors.WithStack(ErrInstanceIDAlreadyInitialized)
	}

	// create a new instance ID
	instanceID, err := NewRandomInstanceID(rng)
	if err != nil {
		return InstanceID{}, err
	}

	// Store the new instance ID
	idBytes, err = instanceID.MarshalText()
	if err != nil {
		return InstanceID{}, err
	}

	err = kv.SetBytes(instanceIDKey, idBytes)
	if err != nil {
		return InstanceID{}, err
	}

	return instanceID, nil
}

func IsLocalInstanceID(kv ekv.KeyValue, expected InstanceID) (bool, error) {
	instanceIDBytes, err := kv.GetBytes(instanceIDKey)
	if err != nil {
		return false, err
	}

	storedInstanceID := InstanceID{}
	err = storedInstanceID.UnmarshalText(instanceIDBytes)
	if err != nil {
		return false, err
	}

	return storedInstanceID.Equals(expected), nil
}

func GetInstanceID(kv ekv.KeyValue) (InstanceID, error) {
	instanceIDBytes, err := kv.GetBytes(instanceIDKey)
	if err != nil {
		return InstanceID{}, err
	}

	storedInstanceID := InstanceID{}
	err = storedInstanceID.UnmarshalText(instanceIDBytes)
	if err != nil {
		return InstanceID{}, err
	}

	return storedInstanceID, nil
}

// NewInstanceIDFromBytes creates an InstanceID from raw bytes
// This returns errors if the number of bytes is incorrect or the
// slice is empty.
func NewInstanceIDFromBytes(idBytes []byte) (InstanceID, error) {
	instanceID := InstanceID{}
	return instanceID, (&instanceID).UnmarshalText(idBytes)
}

// NewInstanceIDFromString creates an instanceID from a string object
// This returns errors if the number of bytes is incorrect or the
// slice is empty.
func NewInstanceIDFromString(idStr string) (InstanceID, error) {
	bytes, err := base64.RawURLEncoding.Strict().DecodeString(idStr)
	if err == nil {
		return NewInstanceIDFromBytes(bytes)
	}
	return InstanceID{}, err
}

// NewRandomInstanceID creates a new random InstanceID from the provided
// cryptographically secured random number generator.
func NewRandomInstanceID(csprng io.Reader) (InstanceID, error) {
	id := InstanceID{}
	instanceIDBytes := make([]byte, instanceIDLength)
	n, err := csprng.Read(instanceIDBytes)
	if n != instanceIDLength {
		return id, ErrShortRead
	}
	if err == nil {
		copy(id[:], instanceIDBytes)
	}
	return id, err
}
