package reception

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"

)

const maxRoundsPerSecond = 100
const knownRoundsStorageKey = "krStorage"


type registration struct{
	Identity
	knownRounds *knownRounds.KnownRounds
	knownRoundsStorage *utility.KnownRounds
	kv *versioned.KV
}

func newRegistration(reg Identity, kv *versioned.KV)(*registration, error){
	//round the times to remove the monotic clocks for future saving
	reg.StartValid = reg.StartValid.Round(0)
	reg.EndValid = reg.EndValid.Round(0)
	reg.End = reg.End.Round(0)

	now := time.Now()

	//do edge checks to determine if the identity is valid
	if now.After(reg.End) && reg.ExtraChecks<1{
		return nil, errors.New("Cannot create a registration for an " +
			"identity which has expired")
	}

	//set the prefix
	kv = kv.Prefix(regPrefix(reg.EphId, reg.Source))


	r := &registration{
		Identity:    reg,
		knownRounds: knownRounds.NewKnownRound(reg.calculateKrSize()),
		kv:          kv,
	}

	//if this isn't ephemeral, store everything
	if !reg.Ephemeral{
		//store known rounds
		var err error
		r.knownRoundsStorage, err = utility.NewKnownRounds(kv, knownRoundsStorageKey, r.knownRounds)
		if err!=nil{
			return nil, errors.WithMessage(err, "failed to store known rounds")
		}
		//store the registration
		if err = reg.store(kv); err!=nil{
			return nil, errors.WithMessage(err, "failed to store registration")
		}
	}

	return r, nil
}

func loadRegistration(EphId  ephemeral.Id, Source *id.ID, kv *versioned.KV)(*registration, error){
	kv = kv.Prefix(regPrefix(EphId, Source))

	reg, err := loadIdentity(kv)
	if err!=nil{
		return nil, errors.WithMessagef(err, "Failed to load identity " +
			"for %s", regPrefix(EphId, Source))
	}

	kr, err := utility.LoadKnownRounds(kv,knownRoundsStorageKey, reg.calculateKrSize())
	if err!=nil{
		return nil, errors.WithMessagef(err, "Failed to load known " +
			"rounds for %s", regPrefix(EphId, Source))
	}

	r := &registration{
		Identity:    reg,
		knownRoundsStorage: kr,
		kv:          kv,
	}

	return r, nil
}


func (r *registration)Delete()error{
	if !r.Ephemeral{
		if err:=r.knownRoundsStorage.Delete(); err!=nil{
			return errors.WithMessagef(err, "Failed to delete " +
				"registration known rounds %s", r)
		}
		if err:=r.delete(r.kv); err!=nil{
			return errors.WithMessagef(err, "Failed to delete " +
				"registration public data %s", r)
		}
	}
	return nil
}

func (r registration)getKR()KnownRounds{
	if r.Ephemeral{
		return r.knownRounds
	}else{
		return r.knownRoundsStorage
	}
}

func regPrefix(EphId  ephemeral.Id, Source *id.ID)string{
	return "receptionRegistration_" + string(EphId.Int64()) + Source.String()
}