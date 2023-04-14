////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"encoding/base64"
	"io"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage/versioned"
)

const (
	instanceIDLength  = 8
	instanceIDVersion = 0
	instanceIDKey     = "ThisInstanceID"
)

var (
	ErrEmptyInstance = errors.New("empty instance ID")
	ErrShortRead     = errors.New("short read generating instance ID")
	ErrIncorrectSize = errors.New("incorrect instance ID size")
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

// LoadInstanceID loads an InstanceID from storage.
func LoadInstanceID(kv versioned.KV) (InstanceID, error) {
	var idBytes []byte
	obj, err := kv.Get(instanceIDKey, instanceIDVersion)
	if obj != nil {
		idBytes = obj.Data
	}
	// If there's an error, just return the empty object and the error
	// Otherwise decode the bytes into the object.
	if err == nil {
		return NewInstanceIDFromBytes(idBytes)
	}
	return InstanceID{}, err
}

// StoreInstanceID saves an instance ID to kv storage.
func StoreInstanceID(id InstanceID, kv versioned.KV) error {
	obj := versioned.Object{
		Data:      id[:],
		Timestamp: time.Now(),
		Version:   instanceIDVersion,
	}
	return kv.Set(instanceIDKey, &obj)
}

// NewInstanceIDFromBytes creates an InstanceID from raw bytes
// This returns errors if the number of bytes is incorrect or the
// slice is empty.
func NewInstanceIDFromBytes(idBytes []byte) (InstanceID, error) {
	instanceID := InstanceID{}
	if len(idBytes) == 0 {
		// Error if we got an empty instance id entry
		return instanceID, ErrEmptyInstance
	} else if len(idBytes) != instanceIDLength {
		// Error if it is the wrong size
		return instanceID, errors.Wrapf(ErrIncorrectSize,
			"%d != %d", instanceIDLength, len(idBytes))
	}
	copy(instanceID[:], idBytes)
	return instanceID, nil
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
