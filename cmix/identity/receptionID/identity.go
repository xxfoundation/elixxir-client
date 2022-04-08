package receptionID

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"strings"
	"time"
)

const identityStorageKey = "IdentityStorage"
const identityStorageVersion = 0

type EphemeralIdentity struct {
	// Identity
	EphId  ephemeral.Id
	Source *id.ID
}

type Identity struct {
	// Identity
	EphemeralIdentity
	AddressSize uint8

	// Usage variables
	End         time.Time // Timestamp when active polling will stop
	ExtraChecks uint      // Number of extra checks executed as active after the
	// ID exits active

	// Polling parameters
	StartValid time.Time // Timestamp when the ephID begins being valid
	EndValid   time.Time // Timestamp when the ephID stops being valid

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

// String returns a string representations of the ephemeral ID and source ID of
// the Identity. This function adheres to the fmt.Stringer interface.
func (i Identity) String() string {
	return strconv.FormatInt(i.EphId.Int64(), 16) + " " + i.Source.String()
}

// GoString returns a string representations of all the values in the Identity.
// This function adheres to the fmt.GoStringer interface.
func (i Identity) GoString() string {
	str := []string{
		"EphId:" + strconv.FormatInt(i.EphId.Int64(), 16),
		"Source:" + i.Source.String(),
		"AddressSize:" + strconv.FormatUint(uint64(i.AddressSize), 10),
		"End:" + i.End.String(),
		"ExtraChecks:" + strconv.FormatUint(uint64(i.ExtraChecks), 10),
		"StartValid:" + i.StartValid.String(),
		"EndValid:" + i.EndValid.String(),
		"Ephemeral:" + strconv.FormatBool(i.Ephemeral),
	}

	return "{" + strings.Join(str, ", ") + "}"
}

func (i Identity) Equal(b Identity) bool {
	return i.EphId == b.EphId &&
		i.Source.Cmp(b.Source) &&
		i.AddressSize == b.AddressSize &&
		i.End.Equal(b.End) &&
		i.ExtraChecks == b.ExtraChecks &&
		i.StartValid.Equal(b.StartValid) &&
		i.EndValid.Equal(b.EndValid) &&
		i.Ephemeral == b.Ephemeral
}

// BuildIdentityFromRound returns an EphemeralIdentity that the source would
// use to receive messages from the given round
func BuildIdentityFromRound(source *id.ID,
	round rounds.Round) EphemeralIdentity {
	ephID, _, _, _ := ephemeral.GetId(source, uint(round.AddressSpaceSize),
		round.Timestamps[states.QUEUED].UnixNano())
	return EphemeralIdentity{
		EphId:  ephID,
		Source: source,
	}
}
