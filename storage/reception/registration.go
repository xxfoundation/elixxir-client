package reception

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"time"
)

const knownRoundsStorageKey = "krStorage"

type registration struct {
	Identity
	UR *rounds.UnknownRounds
	ER *rounds.EarliestRound
	CR *rounds.CheckedRounds
	kv *versioned.KV
}

func newRegistration(reg Identity, kv *versioned.KV) (*registration, error) {
	// Round the times to remove the monotonic clocks for future saving
	reg.StartValid = reg.StartValid.Round(0)
	reg.EndValid = reg.EndValid.Round(0)
	reg.End = reg.End.Round(0)

	now := netTime.Now()

	// Do edge checks to determine if the identity is valid
	if now.After(reg.End) && reg.ExtraChecks < 1 {
		return nil, errors.New("Cannot create a registration for an " +
			"identity which has expired")
	}

	// Set the prefix
	kv = kv.Prefix(regPrefix(reg.EphId, reg.Source, reg.StartValid))

	r := &registration{
		Identity: reg,
		kv:       kv,
	}

	urParams := rounds.DefaultUnknownRoundsParams()
	urParams.Stored = !reg.Ephemeral
	r.UR = rounds.NewUnknownRounds(kv, urParams)
	r.ER = rounds.NewEarliestRound(!reg.Ephemeral, kv)
	cr, err := rounds.NewCheckedRounds(int(params.GetDefaultNetwork().KnownRoundsThreshold), kv)
	if err != nil {
		jww.FATAL.Printf("Failed to create new CheckedRounds for registration: %+v", err)
	}
	r.CR = cr

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

	cr, err := rounds.LoadCheckedRounds(int(params.GetDefaultNetwork().KnownRoundsThreshold), kv)
	if err != nil {
		jww.ERROR.Printf("Making new CheckedRounds, loading of CheckedRounds "+
			"failed: %+v", err)

		cr, err = rounds.NewCheckedRounds(int(params.GetDefaultNetwork().KnownRoundsThreshold), kv)
		if err != nil {
			jww.FATAL.Printf("Failed to create new CheckedRounds for "+
				"registration after CheckedRounds load failure: %+v", err)
		}
	}

	r := &registration{
		Identity: reg,
		kv:       kv,
		UR:       rounds.LoadUnknownRounds(kv, rounds.DefaultUnknownRoundsParams()),
		ER:       rounds.LoadEarliestRound(kv),
		CR:       cr,
	}

	return r, nil
}

func (r *registration) Delete() error {
	if !r.Ephemeral {
		r.UR.Delete()
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
