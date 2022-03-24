package preimage

import (
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

func Generate(data []byte, t string) []byte {
	if t == Default {
		return data
	}
	// Hash fingerprints
	h, _ := blake2b.New256(nil)
	h.Write(data)
	h.Write([]byte(t))

	// Base 64 encode hash and truncate
	return h.Sum(nil)
}

func GenerateRequest(recipient *id.ID) []byte {
	// Hash fingerprints
	h, _ := blake2b.New256(nil)
	h.Write(recipient[:])
	h.Write([]byte(Request))

	// Base 64 encode hash and truncate
	return h.Sum(nil)
}
