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
func generateFakeIdentity(rng io.Reader, idSize uint, now time.Time)(IdentityUse, error){
	//randomly generate an identity
	randIDbytes := make([]byte, id.ArrIDLen-1)
	if _, err := rng.Read(randIDbytes); err!=nil{
		return IdentityUse{}, errors.WithMessage(err, "failed to " +
			"generate a random identity when none is available")
	}

	randID := &id.ID{}
	copy(randID[:id.ArrIDLen-1], randIDbytes)
	randID.SetType(id.User)

	//generate the current ephemeral ID from the random identity
	ephID, start, end, err := ephemeral.GetId(randID, idSize,
		now.UnixNano())
	if err!=nil{
		return IdentityUse{}, errors.WithMessage(err, "failed to " +
			"generate an ephemral ID for random identity when none is " +
			"available")
	}

	return IdentityUse{
		Identity:     Identity{
			EphId:       ephID,
			Source:      randID,
			End:         end,
			ExtraChecks: 0,
			StartValid:  start,
			EndValid:    end,
			RequestMask: 24 * time.Hour,
			Ephemeral:   true,
		},
		Fake:         true,
	}, nil
}
