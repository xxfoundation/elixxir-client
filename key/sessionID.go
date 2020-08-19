package key

import "encoding/base64"

type SessionID [32]byte

func (sid SessionID) Bytes() []byte {
	return sid[:]
}

func (sid SessionID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

//builds the
func makeSessionKey(sid SessionID) string {
	return sid.String()
}
