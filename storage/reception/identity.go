package reception

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"strconv"
	"time"
)

const identityStorageKey = "IdentityStorage"
const identityStorageVersion = 0

type Identity struct{
	//identity
	EphId  ephemeral.Id
	Source *id.ID

	//usage variables
	End         time.Time // timestamp when active polling will stop
	ExtraChecks uint      // number of extra checks executed as active
	// after the id exits active

	//polling parameters
	StartValid  time.Time     // timestamp when the ephID begins being valid
	EndValid    time.Time     // timestamp when the ephID stops being valid
	RequestMask time.Duration // amount of extra time requested for the poll
	// in order to mask the exact valid time for
	// the id

	//makes the identity not store on disk
	Ephemeral bool
}

func loadIdentity(kv *versioned.KV)(Identity, error){
	obj, err := kv.Get(identityStorageKey)
	if err!=nil{
		return Identity{}, errors.WithMessage(err, "Failed to load Identity")
	}

	r := Identity{}
	err = json.Unmarshal(obj.Data, &r)
	if err!=nil{
		return Identity{}, errors.WithMessage(err, "Failed to unmarshal Identity")
	}
	return r, nil
}


func (i Identity)store(kv *versioned.KV)error{
	//marshal the registration
	regStr, err := json.Marshal(&i)
	if err!=nil{
		return errors.WithMessage(err, "Failed to marshal Identity")
	}

	// Create versioned object with data
	obj := &versioned.Object{
		Version:   identityStorageVersion,
		Timestamp: time.Now(),
		Data:      regStr,
	}

	//store the data
	err = kv.Set(identityStorageKey, obj)
	if err!=nil{
		return errors.WithMessage(err, "Failed to store Identity")
	}

	return nil
}

func (i Identity)delete(kv *versioned.KV)error{
	return kv.Delete(identityStorageKey)
}

func (i Identity)calculateKrSize()int{
	return int(i.EndValid.Sub(i.StartValid).Seconds()+1)*maxRoundsPerSecond
}

func (i *Identity)String()string{
	return strconv.FormatInt(i.EphId.Int64(), 16) + " " + i.Source.String()
}

