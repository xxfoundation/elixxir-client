////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
)

const sessionIDLen = 32

type SessionID [sessionIDLen]byte

// GetSessionIDFromBaseKey generates a SessionID from a base key. This is the
// underlying definition of a session ID.
func GetSessionIDFromBaseKey(baseKey *cyclic.Int) SessionID {
	// No lock is needed because this cannot be edited
	sid := SessionID{}
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	copy(sid[:], h.Sum(nil))
	return sid
}

// String returns a human-readable form of the [SessionID] for logging and
// debugging purposes. This functions adheres to the [fmt.Stringer] interface.
func (sid SessionID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

// Marshal marshals the [SessionID] to a byte slice.
func (sid SessionID) Marshal() []byte {
	return sid[:]
}

// Unmarshal unmarshals the byte slice into the [SessionID].
func (sid *SessionID) Unmarshal(b []byte) error {
	if len(b) != sessionIDLen {
		return errors.New("SessionID of invalid length received")
	}
	copy(sid[:], b)
	return nil
}

// MarshalJSON adheres to the [json.Marshaler] interface.
func (sid SessionID) MarshalJSON() ([]byte, error) {
	return json.Marshal(sid.Marshal())
}

// UnmarshalJSON adheres to the [json.Unmarshaler] interface.
func (sid *SessionID) UnmarshalJSON(b []byte) error {
	var buff []byte
	if err := json.Unmarshal(b, &buff); err != nil {
		return err
	}

	err := sid.Unmarshal(buff)
	if err != nil {
		return err
	}

	return nil
}
