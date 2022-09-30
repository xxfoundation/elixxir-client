////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
)

const idLen = 32

type ID [idLen]byte

func (sid ID) Marshal() []byte {
	return sid[:]
}

func (sid ID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

func (sid *ID) Unmarshal(b []byte) error {
	if len(b) != idLen {
		return errors.New("ID of invalid length received")
	}
	copy(sid[:], b)
	return nil
}

// underlying definition of session id
func GetIDFromBaseKey(baseKey *cyclic.Int) ID {
	// no lock is needed because this cannot be edited
	sid := ID{}
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	copy(sid[:], h.Sum(nil))
	return sid
}
