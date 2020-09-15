package e2e

import (
	"encoding/base64"
	"github.com/pkg/errors"
)

const sessionIDLen = 32

type SessionID [sessionIDLen]byte

func (sid SessionID) Marshal() []byte {
	return sid[:]
}

func (sid SessionID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

func (sid SessionID) Unmarshal(b []byte) error {
	if len(b) != sessionIDLen {
		return errors.New("SessionID of invalid length received")
	}

	copy(sid[:], b)
	return nil
}



