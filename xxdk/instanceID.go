////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/base64"
	"io"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/ekv"
)

const (
	instanceIDLength = 8
	instanceIDKey    = "ThisInstanceID"
	emptyInstanceErr = "empty instance ID"
	shortReadErr     = "short read generating instance ID"
	incorrectSizeErr = "incorrect instance ID size: %d != %d"
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
func LoadInstanceID(kv ekv.KeyValue) (InstanceID, error) {
	instanceID := InstanceID{}
	idBytes, err := kv.GetBytes(instanceIDKey)
	if err == nil && len(idBytes) == 0 {
		return instanceID, errors.New(emptyInstanceErr)
	} else if len(idBytes) != instanceIDLength {
		return instanceID, errors.Errorf(incorrectSizeErr,
			instanceIDLength, len(idBytes))
	} else {
		copy(instanceID[:], idBytes)
	}
	return instanceID, err
}

// StoreInstanceID saves an instance ID to kv storage.
func StoreInstanceID(id InstanceID, kv ekv.KeyValue) error {
	return kv.SetBytes(instanceIDKey, id[:])
}

func generateInstanceID(csprng io.Reader) (InstanceID, error) {
	id := InstanceID{}
	instanceIDBytes := make([]byte, instanceIDLength)
	n, err := csprng.Read(instanceIDBytes)
	if n != instanceIDLength {
		return id, errors.New(shortReadErr)
	}
	if err == nil {
		copy(id[:], instanceIDBytes)
	}
	return id, err
}
