package reception

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/big"
	"time"
)

type IdentityUse struct{
	Identity

	//randomly generated time to poll between
	StartRequest time.Time	//timestamp to request the start of bloom filters
	EndRequest time.Time	//timestamp to request the End of bloom filters

	// denotes if the identity is fake, in which case we do not process
	// messages
	Fake bool

	//rounds data
	KR KnownRounds
}

func (iu IdentityUse)SetSamplingPeriod(rng io.Reader)(IdentityUse, error){

	//generate the seed
	seed := make([]byte,32)
	if _, err := rng.Read(seed);err!=nil{
		return IdentityUse{}, errors.WithMessage(err, "Failed to " +
			"choose id due to rng failure")
	}

	h, err := hash.NewCMixHash()
	if err==nil{
		return IdentityUse{}, err
	}

	//calculate the period offset
	periodOffset :=
		randomness.RandInInterval(big.NewInt(iu.RequestMask.Nanoseconds()),
			seed,h).Uint64()
	iu.StartRequest = iu.StartValid.Add(-time.Duration(periodOffset)*
		time.Nanosecond)
	iu.EndRequest = iu.EndValid.Add(iu.RequestMask -
		time.Duration(periodOffset)*time.Nanosecond)
	return iu, nil
}

type KnownRounds interface{
	Checked(rid id.Round) bool
	Check(rid id.Round)
	Forward(rid id.Round)
	RangeUnchecked(newestRid id.Round, roundCheck func(id id.Round) bool)
	RangeUncheckedMasked(mask *knownRounds.KnownRounds,
		roundCheck knownRounds.RoundCheckFunc, maxChecked int)
	RangeUncheckedMaskedRange(mask *knownRounds.KnownRounds,
		roundCheck knownRounds.RoundCheckFunc, start, end id.Round, maxChecked int)
}