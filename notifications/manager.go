package notifications

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

type manager struct {
	transmissionRSA                             rsa.PrivateKey
	transmissionRegistrationValidationSignature []byte
	registrationTimestamp                       time.Time
	registrationSalt                            []byte

	comms *client.Comms
	rng   *fastRNG.StreamGenerator

	notificationHost *connect.Host

	mux sync.Mutex
}

func (m *manager) getIidAndSig(toBeNotified *id.ID, timestamp time.Time, operation string) (
	intermediaryReceptionID, sig []byte, err error) {
	intermediaryReceptionID, err = ephemeral.GetIntermediaryId(toBeNotified)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to form intermediary ID")
	}
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to create cMix hash")
	}
	_, err = h.Write(intermediaryReceptionID)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to write intermediary ID to hash")
	}
	stream := m.rng.GetStream()
	defer stream.Close()
	sig, err = m.transmissionRSA.SignPSS(stream, hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to sign intermediary ID")
	}
	return intermediaryReceptionID, sig, nil
}
