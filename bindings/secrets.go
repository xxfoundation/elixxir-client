package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/csprng"
)

// GenerateSecret creates a secret password using a system-based
// pseudorandom number generator. It takes 1 parameter, `numBytes`,
// which should be set to 32, but can be set higher in certain cases.
func GenerateSecret(numBytes int) []byte {
	if numBytes < 32 {
		jww.FATAL.Panicf("Secrets must have at least 32 bytes " +
			"(256 bits) of entropy.")
	}

	out := make([]byte, numBytes)
	rng := csprng.NewSystemRNG()
	numRead, err := rng.Read(out)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	if numRead != numBytes {
		jww.FATAL.Panicf("Unable to read %d bytes", numBytes)
	}
	return out
}
