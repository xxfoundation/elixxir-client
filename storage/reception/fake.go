package reception

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io"
	"time"
)

// generateFakeIdentity generates a fake identity of the given size with the
// given random number generator
func generateFakeIdentity(rng io.Reader, addressSize uint8,
	now time.Time) (IdentityUse, error) {
	// Randomly generate an identity
	randIdBytes := make([]byte, id.ArrIDLen-1)
	if _, err := rng.Read(randIdBytes); err != nil {
		return IdentityUse{}, errors.WithMessage(err, "failed to "+
			"generate a random identity when none is available")
	}

	randID := &id.ID{}
	copy(randID[:id.ArrIDLen-1], randIdBytes)
	randID.SetType(id.User)

	// Generate the current ephemeral ID from the random identity
	ephID, start, end, err := ephemeral.GetId(
		randID, uint(addressSize), now.UnixNano())
	if err != nil {
		return IdentityUse{}, errors.WithMessage(err, "failed to generate an "+
			"ephemeral ID for random identity when none is available")
	}

	return IdentityUse{
		Identity: Identity{
			EphId:       ephID,
			Source:      randID,
			AddressSize: addressSize,
			End:         end,
			ExtraChecks: 0,
			StartValid:  start,
			EndValid:    end,
			RequestMask: 24 * time.Hour,
			Ephemeral:   true,
		},
		Fake: true,
	}, nil
}
