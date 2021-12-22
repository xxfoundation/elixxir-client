////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"time"
) // Error messages.

const (
	payloadSizeRngErr = "failed to generate random payload size: %+v"
)

// intRng returns, as an int, a non-negative, non-zero random number in [1, n)
// from the csprng.Source.
func intRng(n int, rng csprng.Source) (int, error) {
	v, err := csprng.Generate(8, rng)
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint64(v)%uint64(n-1)) + 1, nil
}

// durationRng returns a duration that is the base duration plus or minus a
// random duration of max randomRange.
func durationRng(base, randomRange time.Duration, rng csprng.Source) (
	time.Duration, error) {
	delta, err := intRng(int(2*randomRange), rng)
	if err != nil {
		return 0, err
	}

	return base + randomRange - time.Duration(delta), nil
}

// newRandomPayload generates a random payload of a random length.
func newRandomPayload(maxPayloadSize int, rng csprng.Source) ([]byte, error) {
	// Generate random payload size
	randomPayloadSize, err := intRng(maxPayloadSize, rng)
	if err != nil {
		return nil, errors.Errorf(payloadSizeRngErr, err)
	}

	randomMsg, err := csprng.Generate(randomPayloadSize, rng)
	if err != nil {
		return nil, err
	}

	return randomMsg, nil
}

// newRandomFingerprint generates a random format.Fingerprint.
func newRandomFingerprint(rng csprng.Source) (format.Fingerprint, error) {
	fingerprintBytes, err := csprng.Generate(format.KeyFPLen, rng)
	if err != nil {
		return format.Fingerprint{}, err
	}

	// Create new fingerprint from bytes
	fingerprint := format.NewFingerprint(fingerprintBytes)

	// Set the first bit to be 0 to comply with the cMix group
	fingerprint[0] &= 0x7F

	return fingerprint, nil
}

// newRandomMAC generates a random MAC.
func newRandomMAC(rng csprng.Source) ([]byte, error) {
	mac, err := csprng.Generate(format.MacLen, rng)
	if err != nil {
		return nil, err
	}

	// Set the first bit to be 0 to comply with the cMix group
	mac[0] &= 0x7F

	return mac, nil
}
