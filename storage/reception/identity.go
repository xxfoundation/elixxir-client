package reception

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"time"
)

const identityStorageKey = "IdentityStorage"
const identityStorageVersion = 0

type Identity struct {
	// Identity
	EphId  ephemeral.Id
	Source *id.ID

	// Usage variables
	End         time.Time // Timestamp when active polling will stop
	ExtraChecks uint      // Number of extra checks executed as active after the
	// ID exits active

	// Polling parameters
	StartValid  time.Time     // Timestamp when the ephID begins being valid
	EndValid    time.Time     // Timestamp when the ephID stops being valid
	RequestMask time.Duration // Amount of extra time requested for the poll in
	// order to mask the exact valid time for the ID

	// Makes the identity not store on disk
	Ephemeral bool
}

func loadIdentity(kv *versioned.KV) (Identity, error) {
	obj, err := kv.Get(identityStorageKey, identityStorageVersion)
	if err != nil {
		return Identity{}, errors.WithMessage(err, "Failed to load Identity")
	}

	r := Identity{}
	err = json.Unmarshal(obj.Data, &r)
	if err != nil {
		return Identity{}, errors.WithMessage(err, "Failed to unmarshal Identity")
	}

	return r, nil
}

func (i Identity) store(kv *versioned.KV) error {
	// Marshal the registration
	regStr, err := json.Marshal(&i)
	if err != nil {
		return errors.WithMessage(err, "Failed to marshal Identity")
	}

	// Create versioned object with data
	obj := &versioned.Object{
		Version:   identityStorageVersion,
		Timestamp: netTime.Now(),
		Data:      regStr,
	}

	// Store the data
	err = kv.Set(identityStorageKey, identityStorageVersion, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store Identity")
	}

	return nil
}

func (i Identity) delete(kv *versioned.KV) error {
	return kv.Delete(identityStorageKey, identityStorageVersion)
}

func (i *Identity) String() string {
	return strconv.FormatInt(i.EphId.Int64(), 16) + " " + i.Source.String()
}

func (i Identity) Equal(b Identity) bool {
	return i.EphId == b.EphId &&
		i.Source.Cmp(b.Source) &&
		i.End.Equal(b.End) &&
		i.ExtraChecks == b.ExtraChecks &&
		i.StartValid.Equal(b.StartValid) &&
		i.EndValid.Equal(b.EndValid) &&
		i.RequestMask == b.RequestMask &&
		i.Ephemeral == b.Ephemeral
}
