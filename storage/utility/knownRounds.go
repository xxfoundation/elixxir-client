////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

/*
// Sub key used in building keys for saving the message to the key value store
const knownRoundsPrefix = "knownRound"

// Version of the file saved to the key value store
const currentKnownRoundsVersion = 0

// KnownRounds stores a buffer of which rounds have been checked and those
// that have yet to be checked. The buffer is saved in a key value store so that
// the values can be recovered if something happens to the buffer in memory.
type KnownRounds struct {
	rounds *knownRounds.KnownRounds
	kv     versioned.KV
	key    string
	mux    sync.RWMutex
}

// NewKnownRounds creates a new empty KnownRounds and saves it to the passed
// in key value store at the specified key. An error is returned on an
// unsuccessful save.
func NewKnownRounds(kv versioned.KV, key string, known *knownRounds.KnownRounds) (*KnownRounds, error) {
	// Create new empty struct
	kr := &KnownRounds{
		rounds: known,
		kv:     kv.Prefix(knownRoundsPrefix),
		key:    key,
	}

	// Save the struct
	err := kr.save()

	// Return the new KnownRounds or an error if the saving failed
	return kr, err
}

// LoadKnownRounds loads and existing KnownRounds from the key value store
// into memory at the given key. Returns an error if it cannot be loaded.
func LoadKnownRounds(kv versioned.KV, key string, size int) (*KnownRounds, error) {
	// Create new empty struct
	kr := &KnownRounds{
		rounds: knownRounds.NewKnownRound(size),
		kv:     kv.Prefix(knownRoundsPrefix),
		key:    key,
	}

	// Load the KnownRounds into the new buffer
	err := kr.load()

	// Return the loaded buffer or an error if loading failed
	return kr, err
}

// save saves the round buffer as a versioned object to the key value store.
func (kr *KnownRounds) save() error {
	now := netTime.Now()

	// Marshal list of rounds
	data, err := kr.rounds.Marshal()
	if err != nil {
		return err
	}

	// Create versioned object with data
	obj := versioned.Object{
		Version:   currentKnownRoundsVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save versioned object
	return kr.kv.Set(kr.key, &obj)
}

// load retrieves the list of rounds from the key value store and stores them
// in the buffer.
func (kr *KnownRounds) load() error {

	// Load the versioned object
	vo, err := kr.kv.get(kr.key)
	if err != nil {
		return err
	}

	// Unmarshal the list of rounds
	err = kr.rounds.Unmarshal(vo.Data)
	if err != nil {
		return err
	}

	return nil
}

// Deletes a known rounds object from disk and memory
func (kr *KnownRounds) DeleteFingerprint() error {
	err := kr.kv.DeleteFingerprint(kr.key)
	if err != nil {
		return err
	}
	return nil
}

// Checked determines if the round has been checked.
func (kr *KnownRounds) Checked(rid id.Round) bool {
	kr.mux.RLock()
	defer kr.mux.RUnlock()

	return kr.rounds.Checked(rid)
}

// Check denotes a round has been checked.
func (kr *KnownRounds) Check(rid id.Round) {
	kr.mux.Lock()
	defer kr.mux.Unlock()

	kr.rounds.Check(rid)

	err := kr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// Forward sets all rounds before the given round ID as checked.
func (kr *KnownRounds) Forward(rid id.Round) {
	kr.mux.Lock()
	defer kr.mux.Unlock()

	kr.rounds.Forward(rid)

	err := kr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// RangeUnchecked runs the passed function over the range of all unchecked round
// IDs up to the passed newestRound to determine if they should be checked.
func (kr *KnownRounds) RangeUnchecked(newestRid id.Round,
	roundCheck func(id id.Round) bool) {
	kr.mux.Lock()
	defer kr.mux.Unlock()

	kr.rounds.RangeUnchecked(newestRid, roundCheck)

	err := kr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// RangeUncheckedMasked checks rounds based off the provided mask.
func (kr *KnownRounds) RangeUncheckedMasked(mask *knownRounds.KnownRounds,
	roundCheck knownRounds.RoundCheckFunc, maxChecked int) {
	kr.mux.Lock()
	defer kr.mux.Unlock()

	kr.rounds.RangeUncheckedMasked(mask, roundCheck, maxChecked)

	err := kr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}

// RangeUncheckedMasked checks rounds based off the provided mask.
func (kr *KnownRounds) RangeUncheckedMaskedRange(mask *knownRounds.KnownRounds,
	roundCheck knownRounds.RoundCheckFunc, start, end id.Round, maxChecked int) {
	kr.mux.Lock()
	defer kr.mux.Unlock()

	kr.rounds.RangeUncheckedMaskedRange(mask, roundCheck, start, end, maxChecked)

	err := kr.save()
	if err != nil {
		jww.FATAL.Panicf("Error saving list of checked rounds: %v", err)
	}
}*/
