package reception

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/rounds"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/randomness"
	"io"
	"math/big"
	"time"
)

type IdentityUse struct {
	Identity

	// Randomly generated time to poll between
	StartRequest time.Time // Timestamp to request the start of bloom filters
	EndRequest   time.Time // Timestamp to request the End of bloom filters

	// Denotes if the identity is fake, in which case we do not process messages
	Fake bool

	UR *rounds.UnknownRounds
	ER *rounds.EarliestRound
}

// setSamplingPeriod add the Request mask as a random buffer around the sampling
// time to obfuscate it.
func (iu IdentityUse) setSamplingPeriod(rng io.Reader) (IdentityUse, error) {

	// Generate the seed
	seed := make([]byte, 32)
	if _, err := rng.Read(seed); err != nil {
		return IdentityUse{}, errors.WithMessage(err, "Failed to choose ID "+
			"due to rng failure")
	}

	h, err := hash.NewCMixHash()
	if err != nil {
		return IdentityUse{}, err
	}

	// Calculate the period offset
	periodOffset := randomness.RandInInterval(
		big.NewInt(iu.RequestMask.Nanoseconds()), seed, h).Int64()
	iu.StartRequest = iu.StartValid.Add(-time.Duration(periodOffset))
	iu.EndRequest = iu.EndValid.Add(iu.RequestMask - time.Duration(periodOffset))
	return iu, nil
}
