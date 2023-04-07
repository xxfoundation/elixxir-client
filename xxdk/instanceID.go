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
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage/versioned"
)

const (
	instanceIDLength  = 8
	instanceIDVersion = 0
	instanceIDKey     = "ThisInstanceID"
	emptyInstanceErr  = "empty instance ID"
	shortReadErr      = "short read generating instance ID"
	incorrectSizeErr  = "incorrect instance ID size: %d != %d"
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
	instanceID := InstanceID{}
	obj, err := kv.Get(instanceIDKey, instanceIDVersion)
	idBytes := obj.Data
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
func StoreInstanceID(id InstanceID, kv versioned.KV) error {
	obj := versioned.Object{
		Data:      id[:],
		Timestamp: time.Now(),
		Version:   instanceIDVersion,
	}
	return kv.Set(instanceIDKey, &obj)
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
