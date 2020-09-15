package utility

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// Version of the file saved to the key value store
const currentCheckedRoundsVersion = 0

// CheckedRounds stores a buffer of which rounds have been checked and those
// that have yet to be checked. The buffer is saved in a key value store so that
// the values can be recovered if something happens to the buffer in memory.
type CheckedRounds struct {
	rounds *knownRounds.KnownRounds
	kv     *versioned.KV
	key    string
	mux    sync.RWMutex
}

// NewCheckedRounds creates a new empty CheckedRounds and saves it to the passed
// in key value store at the specified key. An error is returned on an
// unsuccessful save.
func NewCheckedRounds(kv *versioned.KV, key string, size int) (*CheckedRounds, error) {
	// Create new empty struct
	cr := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     kv,
		key:    key,
	}

	// Save the struct
	err := cr.save()

	// Return the new CheckedRounds or an error if the saving failed
	return cr, err
}

// LoadCheckedRounds loads and existing CheckedRounds from the key value store
// into memory at the given key. Returns an error if it cannot be loaded.
func LoadCheckedRounds(kv *versioned.KV, key string, size int) (*CheckedRounds, error) {
	// Create new empty struct
	cr := &CheckedRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     kv,
		key:    key,
	}

	// Load the KnownRounds into the new buffer
	err := cr.load()

	// Return the loaded buffer or an error if loading failed
	return cr, err
}

// save saves the round buffer as a versioned object to the key value store.
func (cr *CheckedRounds) save() error {
	now := time.Now()

	// Marshal list of rounds
	data, err := cr.rounds.Marshal()
	if err != nil {
		return err
	}

	// Create versioned object with data
	obj := versioned.Object{
		Version:   currentCheckedRoundsVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save versioned object
	return cr.kv.Set(cr.key, &obj)
}

// load retrieves the list of rounds from the key value store and stores them
// in the buffer.
func (cr *CheckedRounds) load() error {

	// Load the versioned object
	vo, err := cr.kv.Get(cr.key)
	if err != nil {
		return err
	}

	// Unmarshal the list of rounds
	err = cr.rounds.Unmarshal(vo.Data)
	if err != nil {
		return err
	}

	return nil
}

// Checked determines if the round has been checked.
func (cr *CheckedRounds) Checked(rid id.Round) bool {
	cr.mux.RLock()
	defer cr.mux.RUnlock()

	return cr.rounds.Checked(rid)
}

// Check denotes a round has been checked.
func (cr *CheckedRounds) Check(rid id.Round) {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.rounds.Check(rid)

	err := cr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// Forward sets all rounds before the given round ID as checked.
func (cr *CheckedRounds) Forward(rid id.Round) {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.rounds.Forward(rid)

	err := cr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// RangeUnchecked runs the passed function over the range of all unchecked round
// IDs up to the passed newestRound to determine if they should be checked.
func (cr *CheckedRounds) RangeUnchecked(newestRid id.Round,
	roundCheck func(id id.Round) bool) {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.rounds.RangeUnchecked(newestRid, roundCheck)

	err := cr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// RangeUncheckedMasked checks rounds based off the provided mask.
func (cr *CheckedRounds) RangeUncheckedMasked(mask *knownRounds.KnownRounds,
	roundCheck func(id id.Round) bool, maxChecked int) {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	cr.rounds.RangeUncheckedMasked(mask, roundCheck, maxChecked)

	err := cr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}
