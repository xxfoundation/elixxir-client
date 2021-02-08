package reception

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io"
	"math/big"
	"strconv"
	"sync"
	"time"
)

const receptionPrefix = "reception"
const receptionStoreStorageKey = "receptionStoreKey"
const receptionStoreStorageVersion = 0
const receptionIDSizeStorageKey = "receptionIDSizeKey"
const receptionIDSizeStorageVersion = 0
const defaultIDSize = 12

type Store struct {
	// Identities which are being actively checked
	active []*registration
	idSize int

	kv *versioned.KV

	mux sync.Mutex
}

type storedReference struct {
	Eph        ephemeral.Id
	Source     *id.ID
	StartValid time.Time
}

// NewStore creates a new reception store that starts empty.
func NewStore(kv *versioned.KV) *Store {
	kv = kv.Prefix(receptionPrefix)
	s := &Store{
		active: make([]*registration, 0),
		idSize: defaultIDSize,
		kv:     kv,
	}

	// Store the empty list
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save new reception store: %+v", err)
	}

	// Update the size so queries can be made
	s.UpdateIDSize(defaultIDSize)

	return s
}

func LoadStore(kv *versioned.KV) *Store {
	kv = kv.Prefix(receptionPrefix)
	s := &Store{
		kv: kv,
	}

	// Load the versioned object for the reception list
	vo, err := kv.Get(receptionStoreStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the reception storage list: %+v",
			err)
	}

	identities := make([]storedReference, len(s.active))
	err = json.Unmarshal(vo.Data, &identities)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal the reception storage "+
			"list: %+v", err)
	}

	s.active = make([]*registration, len(identities))
	for i, sr := range identities {
		s.active[i], err = loadRegistration(sr.Eph, sr.Source, sr.StartValid, s.kv)
		if err != nil {
			jww.FATAL.Panicf("Failed to load registration for %s: %+v",
				regPrefix(sr.Eph, sr.Source, sr.StartValid), err)
		}
	}

	// Load the ephemeral ID length
	vo, err = kv.Get(receptionIDSizeStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the reception ID size: %+v",
			err)
	}

	if s.idSize, err = strconv.Atoi(string(vo.Data)); err != nil {
		jww.FATAL.Panicf("Failed to unmarshal the reception ID size: %+v",
			err)
	}

	return s
}

func (s *Store) save() error {
	identities := make([]storedReference, len(s.active))
	i := 0
	for _, reg := range s.active {
		if !reg.Ephemeral {
			identities[i] = storedReference{
				Eph:        reg.EphId,
				Source:     reg.Source,
				StartValid: reg.StartValid.Round(0),
			}
			i++
		}
	}
	identities = identities[:i]

	data, err := json.Marshal(&identities)
	if err != nil {
		return errors.WithMessage(err, "failed to store reception "+
			"store")
	}

	// Create versioned object with data
	obj := &versioned.Object{
		Version:   receptionStoreStorageVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	err = s.kv.Set(receptionStoreStorageKey, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store reception store")
	}

	return nil
}

func (s *Store) GetIdentity(rng io.Reader) (IdentityUse, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	now := time.Now()

	// Remove any now expired identities
	s.prune(now)

	var identity IdentityUse
	var err error

	// If the list is empty, then we return a randomly generated identity to
	// poll with so we can continue tracking the network and to further
	// obfuscate network identities.
	if len(s.active) == 0 {
		identity, err = generateFakeIdentity(rng, uint(s.idSize), now)
		if err != nil {
			jww.FATAL.Panicf("Failed to generate a new ID when none "+
				"available: %+v", err)
		}
	} else {
		identity, err = s.selectIdentity(rng, now)
		if err != nil {
			jww.FATAL.Panicf("Failed to select an ID: %+v", err)
		}
	}

	// Calculate the sampling period
	identity, err = identity.setSamplingPeriod(rng)
	if err != nil {
		jww.FATAL.Panicf("Failed to calculate the sampling period: "+
			"%+v", err)
	}

	return identity, nil
}

func (s *Store) AddIdentity(identity Identity) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	reg, err := newRegistration(identity, s.kv)
	if err != nil {
		return errors.WithMessage(err, "failed to add new identity to "+
			"reception store")
	}

	s.active = append(s.active, reg)
	if !identity.Ephemeral {
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to save reception store after identity " +
				"addition")
		}
	}

	return nil
}

func (s *Store) RemoveIdentity(ephID ephemeral.Id) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for i := 0; i < len(s.active); i++ {
		inQuestion := s.active[i]
		if bytes.Equal(inQuestion.EphId[:], ephID[:]) {
			s.active = append(s.active[:i], s.active[i+1:]...)
			err := inQuestion.Delete()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete identity: %+v", err)
			}
			if !inQuestion.Ephemeral {
				if err := s.save(); err != nil {
					jww.FATAL.Panicf("Failed to save reception store after " +
						"identity removal")
				}
			}
			return
		}
	}
}

func (s *Store) UpdateIDSize(idSize uint) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.idSize == int(idSize){
		return
	}

	s.idSize = int(idSize)

	// Store the ID size
	obj := &versioned.Object{
		Version:   receptionIDSizeStorageVersion,
		Timestamp: time.Now(),
		Data:      []byte(strconv.Itoa(s.idSize)),
	}

	err := s.kv.Set(receptionIDSizeStorageKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store reception ID size: %+v", err)
	}
}

func (s *Store)GetIDSize()uint {
	s.mux.Lock()
	defer s.mux.Unlock()
	return uint(s.idSize)
}

func (s *Store) prune(now time.Time) {
	lengthBefore := len(s.active)

	// Prune the list
	for i := 0; i < len(s.active); i++ {
		inQuestion := s.active[i]
		if now.After(inQuestion.End) && inQuestion.ExtraChecks == 0 {
			if err := inQuestion.Delete(); err != nil {
				jww.ERROR.Printf("Failed to delete Identity for %s: "+
					"%+v", inQuestion, err)
			}

			s.active = append(s.active[:i-1], s.active[i:]...)

			i--
		}
	}

	// Save the list if it changed
	if lengthBefore != len(s.active) {
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to store reception storage")
		}
	}
}

func (s *Store) selectIdentity(rng io.Reader, now time.Time) (IdentityUse, error) {

	// Choose a member from the list
	var selected *registration

	if len(s.active) == 1 {
		selected = s.active[0]
	} else {
		seed := make([]byte, 32)
		if _, err := rng.Read(seed); err != nil {
			return IdentityUse{}, errors.WithMessage(err, "Failed to "+
				"choose ID due to rng failure")
		}

		h, err := hash.NewCMixHash()
		if err == nil {
			return IdentityUse{}, err
		}

		selectedNum := randomness.RandInInterval(
			big.NewInt(int64(len(s.active)-1)), seed, h)
		selected = s.active[selectedNum.Uint64()]
	}

	if now.After(selected.End) {
		selected.ExtraChecks--
	}

	return IdentityUse{
		Identity: selected.Identity,
		Fake:     false,
		KR:       selected.getKR(),
	}, nil
}
