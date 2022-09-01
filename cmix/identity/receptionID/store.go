package receptionID

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"io"
	"sync"
	"time"
)

const (
	receptionPrefix              = "reception"
	receptionStoreStorageKey     = "receptionStoreKey"
	receptionStoreStorageVersion = 0
)

var InvalidRequestedNumIdentities = errors.New("cannot get less than one identity(s)")

type Store struct {
	// Identities which are being actively checked
	active  []*registration
	present map[idHash]struct{}

	kv *versioned.KV

	mux sync.Mutex
}

type storedReference struct {
	Eph        ephemeral.Id
	Source     *id.ID
	StartValid time.Time
}

type idHash [16]byte

func makeIdHash(ephID ephemeral.Id, source *id.ID) idHash {
	h, _ := blake2b.New256(nil)
	h.Write(ephID[:])
	h.Write(source.Bytes())
	idH := idHash{}
	copy(idH[:], h.Sum(nil))
	return idH
}

// NewOrLoadStore creates a new reception store that starts empty.
func NewOrLoadStore(kv *versioned.KV) *Store {

	s, err := loadStore(kv)
	if err != nil {
		jww.WARN.Printf(
			"ReceptionID store not found, creating a new one: %+v", err)

		s = &Store{
			active:  []*registration{},
			present: make(map[idHash]struct{}),
			kv:      kv.Prefix(receptionPrefix),
		}

		// Store the empty list
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to save new reception store: %+v", err)
		}
	}

	return s
}

func loadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(receptionPrefix)

	// Load the versioned object for the reception list
	vo, err := kv.Get(receptionStoreStorageKey, receptionStoreStorageVersion)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to get the reception storage list")
	}

	// JSON unmarshal identities list
	var identities []storedReference
	if err = json.Unmarshal(vo.Data, &identities); err != nil {
		return nil, errors.WithMessage(err,
			"Failed to unmarshal the stored identity list")
	}

	s := &Store{
		active:  make([]*registration, len(identities)),
		present: make(map[idHash]struct{}, len(identities)),
		kv:      kv,
	}

	for i, sr := range identities {
		s.active[i], err = loadRegistration(
			sr.Eph, sr.Source, sr.StartValid, s.kv)
		if err != nil {
			return nil, errors.WithMessagef(err,
				"failed to load registration for: %+v",
				regPrefix(sr.Eph, sr.Source, sr.StartValid))
		}
		s.present[makeIdHash(sr.Eph, sr.Source)] = struct{}{}
	}

	return s, nil
}

func (s *Store) save() error {
	identities := s.makeStoredReferences()
	data, err := json.Marshal(&identities)
	if err != nil {
		return errors.WithMessage(err, "failed to store reception store")
	}

	// Create versioned object with data
	obj := &versioned.Object{
		Version:   receptionStoreStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = s.kv.Set(receptionStoreStorageKey, receptionStoreStorageVersion, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store reception store")
	}

	return nil
}

// makeStoredReferences generates a reference of any non-address identities
// for storage.
func (s *Store) makeStoredReferences() []storedReference {
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

	return identities[:i]
}

// GetIdentity will return a single identity. If none are available, it will
// return a fake one
func (s *Store) GetIdentity(rng io.Reader, addressSize uint8) IdentityUse {
	s.mux.Lock()
	defer s.mux.Unlock()

	now := netTime.Now()

	// Remove any now expired identities
	s.prune(now)

	var identity IdentityUse
	var err error

	// If the list is empty, then return a randomly generated identity to poll
	// with so that we can continue tracking the network and to further
	// obfuscate network identities.
	if len(s.active) == 0 {
		identity, err = generateFakeIdentity(rng, addressSize, now)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to generate a new ID when none available: %+v", err)
		}
	} else {
		identity, err = s.selectIdentity(rng, now)
		if err != nil {
			jww.FATAL.Panicf("Failed to select an ID: %+v", err)
		}
	}

	return identity
}

// GetIdentities will return up to 'n' identities randomly in a random order.
// if no identities exist, it will return a single fake identity
func (s *Store) GetIdentities(n int, rng io.Reader,
	addressSize uint8) ([]IdentityUse, error) {

	if n < 1 {
		return nil, InvalidRequestedNumIdentities
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	now := netTime.Now()

	// Remove any now expired identities
	s.prune(now)

	var identities []IdentityUse
	var err error

	// If the list is empty, then return a randomly generated identity to poll
	// with so that we can continue tracking the network and to further
	// obfuscate network identities.
	if len(s.active) == 0 {
		fakeIdentity, err := generateFakeIdentity(rng, addressSize, now)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to generate a new ID when none available: %+v", err)
		}
		identities = append(identities, fakeIdentity)
		// otherwise, select identities to return using a fisher-yates
	} else {
		identities, err = s.selectIdentities(n, rng, now)
		if err != nil {
			jww.FATAL.Panicf("Failed to select a list of IDs: %+v", err)
		}
	}

	return identities, nil
}

func (s *Store) AddIdentity(identity Identity) error {
	idH := makeIdHash(identity.EphId, identity.Source)
	s.mux.Lock()
	defer s.mux.Unlock()

	// Do not make duplicates of IDs
	if _, ok := s.present[idH]; ok {
		jww.DEBUG.Printf("Ignoring duplicate identity for %d (%s)",
			identity.EphId.Int64(), identity.Source)
		return nil
	}

	if identity.StartValid.After(identity.EndValid) {
		return errors.Errorf("Cannot add an identity which start valid "+
			"time (%s) is after its end valid time (%s)", identity.StartValid,
			identity.EndValid)
	}

	reg, err := newRegistration(identity, s.kv)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to add new identity to reception store")
	}

	s.active = append(s.active, reg)
	s.present[idH] = struct{}{}
	if !identity.Ephemeral {
		if err = s.save(); err != nil {
			jww.FATAL.Panicf("Failed to save reception store after identity "+
				"addition: %+v", err)
		}
	}

	return nil
}

func (s *Store) RemoveIdentity(ephID ephemeral.Id) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for i, inQuestion := range s.active {
		if inQuestion.EphId == ephID {
			s.active = append(s.active[:i], s.active[i+1:]...)

			err := inQuestion.Delete()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete identity: %+v", err)
			}

			if !inQuestion.Ephemeral {
				if err := s.save(); err != nil {
					jww.FATAL.Panicf("Failed to save reception store after "+
						"identity removal: %+v", err)
				}
			}

			return
		}
	}
}

func (s *Store) RemoveIdentities(source *id.ID) {
	s.mux.Lock()
	defer s.mux.Unlock()

	doSave := false
	for i, inQuestion := range s.active {
		if inQuestion.Source.Cmp(source) {
			s.active = append(s.active[:i], s.active[i+1:]...)

			err := inQuestion.Delete()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete identity: %+v", err)
			}

			doSave = doSave || !inQuestion.Ephemeral
		}
	}
	if doSave {
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to save reception store after "+
				"identity removal: %+v", err)
		}
	}
}

func (s *Store) SetToExpire(addressSize uint8) {
	s.mux.Lock()
	defer s.mux.Unlock()

	expire := netTime.Now().Add(5 * time.Minute)

	for i, active := range s.active {
		if active.AddressSize < addressSize && active.EndValid.After(expire) {
			s.active[i].EndValid = expire
			err := s.active[i].store(s.kv)
			if err != nil {
				jww.ERROR.Printf("Failed to store identity %d: %+v", i, err)
			}
		}
	}
}

func (s *Store) prune(now time.Time) {
	lengthBefore := len(s.active)
	var pruned []int64
	// Prune the list
	for i := 0; i < len(s.active); i++ {
		inQuestion := s.active[i]
		if now.After(inQuestion.End) && inQuestion.ExtraChecks == 0 {
			if err := inQuestion.Delete(); err != nil {
				jww.ERROR.Printf("Failed to delete Identity for %s: %+v",
					inQuestion, err)
			}
			pruned = append(pruned, inQuestion.EphId.Int64())

			s.active = append(s.active[:i], s.active[i+1:]...)

			i--
		}
	}

	// Save the list if it changed
	if lengthBefore != len(s.active) {
		jww.INFO.Printf(
			"Pruned %d identities [%+v]", lengthBefore-len(s.active), pruned)
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to store reception storage: %+v", err)
		}
	}
}

// selectIdentity returns a random identity in an IdentityUse object and
// increments its usage if necessary
func (s *Store) selectIdentity(rng io.Reader, now time.Time) (IdentityUse, error) {
	// Choose a member from the list
	var selected *registration

	if len(s.active) == 1 {
		selected = s.active[0]
	} else {
		seed := make([]byte, 32) //use 256 bits of entropy for the seed
		if _, err := rng.Read(seed); err != nil {
			return IdentityUse{}, errors.WithMessage(err, "Failed to choose "+
				"ID due to RNG failure")
		}

		selectedNum := large.NewInt(1).Mod(
			large.NewIntFromBytes(seed),
			large.NewInt(int64(len(s.active))),
		)

		selected = s.active[selectedNum.Uint64()]
	}

	jww.TRACE.Printf("Selected identity: EphId: %d  ID: %s  End: %s  "+
		"StartValid: %s  EndValid: %s",
		selected.EphId.Int64(), selected.Source,
		selected.End.Format("01/02/06 03:04:05 pm"),
		selected.StartValid.Format("01/02/06 03:04:05 pm"),
		selected.EndValid.Format("01/02/06 03:04:05 pm"))

	return useIdentity(selected, now), nil
}

// selectIdentities returns up to 'n' identities in an IdentityUse object
// selected via fisher-yates and increments their usage if necessary
func (s *Store) selectIdentities(n int, rng io.Reader, now time.Time) ([]IdentityUse, error) {
	// Choose a member from the list
	selected := make([]IdentityUse, 0, n)

	if len(s.active) == 1 {
		selected = append(selected, useIdentity(s.active[0], now))
	} else {

		// make the seed
		seed := make([]byte, 32) //use 256 bits of entropy for the seed
		if _, err := rng.Read(seed); err != nil {
			return nil, errors.WithMessage(err, "Failed to choose "+
				"ID due to RNG failure")
		}

		// make the list to shuffle
		registered := make([]*registration, 0, len(s.active))
		for i := 0; i < len(s.active); i++ {
			registered = append(registered, s.active[i])
		}

		//shuffle the list via fisher-yates
		shuffle.ShuffleSwap(seed, len(s.active), func(i int, j int) {
			registered[i], registered[j] = registered[j], registered[i]
		})

		//convert the list to identity use
		for i := 0; i < len(registered) && (i < n); i++ {
			selected = append(selected, useIdentity(registered[i], now))
		}

	}

	jww.TRACE.Printf("Selected %d identities, first identity: EphId: %d  ID: %s  End: %s  "+
		"StartValid: %s  EndValid: %s", len(selected),
		selected[0].EphId.Int64(), selected[0].Source,
		selected[0].End.Format("01/02/06 03:04:05 pm"),
		selected[0].StartValid.Format("01/02/06 03:04:05 pm"),
		selected[0].EndValid.Format("01/02/06 03:04:05 pm"))

	return selected, nil
}

// useIdentity makes the public IdentityUse object from a private *registration
// and deals with denoting the usage in the *registration if nessessay
func useIdentity(selected *registration, now time.Time) IdentityUse {
	if now.After(selected.End) {
		selected.ExtraChecks--
	}
	return IdentityUse{
		Identity: selected.Identity,
		Fake:     false,
		UR:       selected.UR,
		ER:       selected.ER,
		CR:       selected.CR,
	}
}
