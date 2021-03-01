package reception

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"strconv"
	"time"
)

const knownRoundsStorageKey = "krStorage"

type registration struct {
	Identity
	ur *UnknownRound
	kv                 *versioned.KV
}

func newRegistration(reg Identity, kv *versioned.KV) (*registration, error) {
	// Round the times to remove the monotonic clocks for future saving
	reg.StartValid = reg.StartValid.Round(0)
	reg.EndValid = reg.EndValid.Round(0)
	reg.End = reg.End.Round(0)

	now := time.Now()

	// Do edge checks to determine if the identity is valid
	if now.After(reg.End) && reg.ExtraChecks < 1 {
		return nil, errors.New("Cannot create a registration for an " +
			"identity which has expired")
	}

	// Set the prefix
	kv = kv.Prefix(regPrefix(reg.EphId, reg.Source, reg.StartValid))

	r := &registration{
		Identity:    reg,
		ur: NewUnknownRound(!reg.Ephemeral, kv),
		kv:          kv,
	}

	// If this is not ephemeral, then store everything
	if !reg.Ephemeral {
		// Store known rounds
		var err error
		// Store the registration
		if err = reg.store(kv); err != nil {
			return nil, errors.WithMessage(err, "failed to store registration")
		}
	}



	return r, nil
}

func loadRegistration(EphId ephemeral.Id, Source *id.ID, startValid time.Time,
	kv *versioned.KV) (*registration, error) {

	kv = kv.Prefix(regPrefix(EphId, Source, startValid))

	reg, err := loadIdentity(kv)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load identity "+
			"for %s", regPrefix(EphId, Source, startValid))
	}

	ur := LoadUnknownRound(kv)

	r := &registration{
		Identity:           reg,
		ur: ur,
		kv:                 kv,
	}

	return r, nil
}

func (r *registration) Delete() error {
	if !r.Ephemeral {
		r.ur.delete()
		if err := r.delete(r.kv); err != nil {
			return errors.WithMessagef(err, "Failed to delete registration "+
				"public data %s", r)
		}
	}

	return nil
}

func regPrefix(EphId ephemeral.Id, Source *id.ID, startTime time.Time) string {
	return "receptionRegistration_" +
		strconv.FormatInt(EphId.Int64(), 16) + Source.String() +
		strconv.FormatInt(startTime.Round(0).UnixNano(), 10)
}
