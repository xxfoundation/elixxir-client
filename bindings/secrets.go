///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/csprng"
)

// GenerateSecret creates a secret password using a system-based pseudorandom
// number generator.
//
// Parameters:
//  - numBytes - The size of secret. It should be set to 32, but can be set
//   higher in certain cases.
func GenerateSecret(numBytes int) []byte {
	if numBytes < 32 {
		jww.FATAL.Panic(
			"Secrets must have at least 32 bytes (256 bits) of entropy.")
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
