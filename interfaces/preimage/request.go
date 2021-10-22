package preimage

import (
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

func MakeRequest(uid *id.ID) []byte {
	h, _ := blake2b.New256(nil)
	h.Write(uid[:])
	h.Write([]byte(Request))

	// Base 64 encode hash and truncate
	return h.Sum(nil)
}

func MakeDefault(uid *id.ID) []byte {
	// Base 64 encode hash and truncate
	return uid[:]
}
