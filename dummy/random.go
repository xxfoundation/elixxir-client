////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"time"
) // Error messages.

// Error constants for Manager.newRandomCmixMessage and it's helper functions..
const (
	payloadSizeRngErr = "failed to generate random payload size: %+v"
	payloadRngErr     = "failed to generate random payload: %+v"
	fingerprintRngErr = "failed to generate random fingerprint: %+v"
	macRngErr         = "failed to generate random MAC: %+v"
	recipientRngErr   = "failed to generate random recipient: %+v"
)

// newRandomCmixMessage returns random format.Message data.
//
// Returns in order a:
//   - Recipient (id.ID)
//   - Message fingerprint (format.Fingerprint)
//   - Message service (message.Service)
//   - Payload ([]byte)
//   - MAC ([]byte)
//   - Error if there was an issue randomly generating any of the above data.
//     The error will specify which of the above failed to be randomly generated.
func (m *Manager) newRandomCmixMessage(rng csprng.Source) (
	recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service,
	payload, mac []byte, err error) {

	// Generate random recipient
	recipient, err = id.NewRandomID(rng, id.User)
	if err != nil {
		return nil, format.Fingerprint{}, message.Service{}, nil, nil,
			errors.Errorf(recipientRngErr, err)
	}

	// Generate random message payload
	payloadSize := m.net.GetMaxMessageLength()
	payload, err = newRandomPayload(payloadSize, rng)
	if err != nil {
		return nil, format.Fingerprint{}, message.Service{}, nil, nil,
			errors.Errorf(payloadRngErr, err)
	}

	// Generate random fingerprint
	fingerprint, err = newRandomFingerprint(rng)
	if err != nil {
		return nil, format.Fingerprint{}, message.Service{}, nil, nil,
			errors.Errorf(fingerprintRngErr, err)
	}

	// Generate random MAC
	mac, err = newRandomMAC(rng)
	if err != nil {
		return nil, format.Fingerprint{}, message.Service{}, nil, nil,
			errors.Errorf(macRngErr, err)
	}

	// Generate random service
	service = message.GetRandomService(rng)

	return
}

// newRandomPayload generates a random payload of a random length
// within the maxPayloadSize.
func newRandomPayload(maxPayloadSize int, rng csprng.Source) ([]byte, error) {

	randomMsg, err := csprng.Generate(maxPayloadSize, rng)
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

//////////////////////////////////////////////////////////////////////////////////
// Miscellaneous
//////////////////////////////////////////////////////////////////////////////////

// randomDuration returns a duration that is the base duration plus or minus a
// random duration of max randomRange.
func randomDuration(base, randomRange time.Duration, rng csprng.Source) (
	time.Duration, error) {

	// Generate a random duration
	delta, err := randomInt(int(2*randomRange), rng)
	if err != nil {
		return 0, err
	}

	return base + randomRange - time.Duration(delta), nil
}

// randomInt returns, as an int, a non-negative, non-zero random number in [1, n)
// from the csprng.Source.
func randomInt(n int, rng csprng.Source) (int, error) {
	v, err := csprng.Generate(8, rng)
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint64(v)%uint64(n-1)) + 1, nil
}
