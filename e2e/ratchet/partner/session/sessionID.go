////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
)

const sessionIDLen = 32

type SessionID [sessionIDLen]byte

func (sid SessionID) Marshal() []byte {
	return sid[:]
}

func (sid SessionID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

func (sid *SessionID) Unmarshal(b []byte) error {
	if len(b) != sessionIDLen {
		return errors.New("SessionID of invalid length received")
	}
	copy(sid[:], b)
	return nil
}

// underlying definition of session id
func GetSessionIDFromBaseKey(baseKey *cyclic.Int) SessionID {
	// no lock is needed because this cannot be edited
	sid := SessionID{}
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	copy(sid[:], h.Sum(nil))
	return sid
}
