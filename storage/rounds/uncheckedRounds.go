///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"sync"
	"time"
)

const (
	uncheckedRoundVersion = 0
	uncheckedRoundPrefix  = "uncheckedRoundPrefix"
	// Key to store round list
	uncheckedRoundListKey = "uncheckRounds"
	// Key to store individual round
	uncheckedRoundKey = "uncheckedRound-"
	// Housekeeping constant (used for serializing uint64 ie id.Round)
	uint64Size = 8
)

// Round identity information used in message retrieval
// Derived from reception.Identity
type Identity struct {
	EpdId  ephemeral.Id
	Source *id.ID
}

// Unchecked round structure is rounds which failed on message retrieval
// These rounds are stored for retry of message retrieval
type UncheckedRound struct {
	//
	Rid id.Round
	Identity
	// Timestamp in which round has been stored
	StoredTimestamp time.Time
	// Number of times a round has been checked
	NumTries uint
}

// Storage object saving rounds to retry for message retrieval
type UncheckedRoundStore struct {
	list map[id.Round]UncheckedRound
	mux  sync.RWMutex
	kv   *versioned.KV
}

// Constructor for a UncheckedRoundStore
func NewUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {
	kv = kv.Prefix(uncheckedRoundPrefix)

	urs := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound, 0),
		kv:   kv,
	}

	return urs, urs.save()

}

// Loads an deserializes a UncheckedRoundStore from memory
func LoadUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {

	kv = kv.Prefix(uncheckedRoundPrefix)
	vo, err := kv.Get(uncheckedRoundListKey, uncheckedRoundVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal list of round IDs
	var roundIDs []id.Round
	buff := bytes.NewBuffer(vo.Data)
	for next := buff.Next(uint64Size); len(next) == uint64Size; next = buff.Next(uint64Size) {
		rid, _ := binary.Varint(next)
		roundIDs = append(roundIDs, id.Round(rid))
	}

	urs := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound, len(roundIDs)),
		kv:   kv,
	}

	// Load each round from storage
	for _, roundId := range roundIDs {
		rnd, err := loadRound(roundId, kv)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to load round %d from storage", roundId)
		}

		urs.list[roundId] = rnd
	}

	return urs, err
}

// Adds a round to check on the list and saves to memory
func (s *UncheckedRoundStore) AddRound(rid id.Round, ephID ephemeral.Id, source *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.list[rid]; !exists {
		newUncheckedRound := UncheckedRound{
			Rid: rid,
			Identity: Identity{
				EpdId:  ephID,
				Source: source,
			},
			StoredTimestamp: netTime.Now(),
			NumTries:        0,
		}

		s.list[rid] = newUncheckedRound

		return s.save()
	}

	return nil
}

// Retrieves an UncheckedRound from the map, if it exists
func (s *UncheckedRoundStore) GetRound(rid id.Round) (UncheckedRound, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	rnd, exists := s.list[rid]
	return rnd, exists
}

// Retrieves the list of rounds
func (s *UncheckedRoundStore) GetList() map[id.Round]UncheckedRound {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.list
}

// Increments the amount of checks performed on this stored round
func (s *UncheckedRoundStore) IncrementCheck(rid id.Round) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	rnd, exists := s.list[rid]
	if !exists {
		return errors.Errorf("round %d could not be found in RAM", rid)
	}

	rnd.NumTries++

	return rnd.store(s.kv)

}

// Remove deletes a round from UncheckedRoundStore's list and from storage
func (s *UncheckedRoundStore) Remove(rid id.Round) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	delete(s.list, rid)

	return s.kv.Delete(roundStoreKey(rid), uncheckedRoundVersion)

}

// save stores the round list and individual rounds to storage
func (s *UncheckedRoundStore) save() error {
	// Store list of rounds
	err := s.saveRoundList()
	if err != nil {
		return errors.WithMessage(err, "Failed to save list of rounds")
	}

	// Store individual rounds
	for rid, rnd := range s.list {
		if err = rnd.store(s.kv); err != nil {
			return errors.WithMessagef(err, "Failed to save round %d to storage", rid)
		}
	}

	return nil
}

// saveRoundList saves the list of rounds to storage
func (s *UncheckedRoundStore) saveRoundList() error {

	// Create the versioned object
	obj := &versioned.Object{
		Version:   uncheckedRoundVersion,
		Timestamp: netTime.Now(),
		Data:      serializeRoundList(s.list),
	}

	// Save to storage
	err := s.kv.Set(uncheckedRoundListKey, uncheckedRoundVersion, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store round ID list")
	}

	return nil
}

// serializeRoundList is a helper function which serializes the list of rounds
// to bytes
func serializeRoundList(list map[id.Round]UncheckedRound) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(uint64Size * len(list))
	for rid := range list {
		b := make([]byte, uint64Size)
		binary.LittleEndian.PutUint64(b, uint64(rid))
		buff.Write(b)
	}
	return buff.Bytes()
}

// store puts serialized UncheckedRound r into storage
func (r UncheckedRound) store(kv *versioned.KV) error {
	data, err := r.Serialize()
	if err != nil {

	}
	obj := &versioned.Object{
		Version:   uncheckedRoundVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return kv.Set(roundStoreKey(r.Rid), uncheckedRoundVersion, obj)

}

// Serialize serializes UncheckedRound r into a byte slice
func (r UncheckedRound) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Write the round ID
	b := make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(r.Rid))
	buf.Write(b)

	// Write the round identity info
	buf.Write(r.Identity.EpdId[:])
	if r.Source != nil {
		buf.Write(r.Identity.Source.Marshal())
	} else {
		buf.Write(make([]byte, id.ArrIDLen))
	}

	// Write the time stamp bytes
	tsBytes, err := r.StoredTimestamp.MarshalBinary()
	if err != nil {
		return nil, errors.WithMessage(err, "Could not marshal timestamp ")
	}
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(len(tsBytes)))
	buf.Write(b)
	buf.Write(tsBytes)

	// Write the number of tries for this round
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(r.NumTries))
	buf.Write(b)

	return buf.Bytes(), nil
}

// loadRound pulls an UncheckedRound corresponding to roundId from storage
func loadRound(roundId id.Round, kv *versioned.KV) (UncheckedRound, error) {
	vo, err := kv.Get(roundStoreKey(roundId), uncheckedRoundVersion)
	if err != nil {
		return UncheckedRound{}, errors.WithMessagef(err, "Could not find %d in storage", roundId)
	}

	ur, err := deserializeRound(vo.Data)
	if err != nil {
		return UncheckedRound{}, errors.WithMessagef(err, "Could not deserialize round %d", roundId)
	}

	return ur, nil
}

// deserializeRound deserializes an UncheckedRound from its stored byte data
func deserializeRound(data []byte) (UncheckedRound, error) {
	buff := bytes.NewBuffer(data)
	rnd := UncheckedRound{}
	rid, _ := binary.Varint(buff.Next(uint64Size))
	rnd.Rid = id.Round(rid)

	// Deserialize the round identity information
	copy(rnd.EpdId[:], buff.Next(uint64Size))

	sourceId, err := id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return UncheckedRound{}, errors.WithMessage(err, "Failed to unmarshal round identity.source")
	}

	rnd.Source = sourceId

	// Deserialize the timestamp bytes
	timestampLen := binary.LittleEndian.Uint64(buff.Next(uint64Size))
	tsByes := buff.Next(int(uint64(timestampLen)))
	if err = rnd.StoredTimestamp.UnmarshalBinary(tsByes); err != nil {
		return UncheckedRound{}, errors.WithMessage(err, "Failed to unmarshal round timestamp")
	}

	numTries, _ := binary.Varint(buff.Next(uint64Size))
	rnd.NumTries = uint(numTries)

	return rnd, nil
}

// roundStoreKey generates a unique key to save and load a round to/from storage.
func roundStoreKey(roundId id.Round) string {
	return uncheckedRoundKey + strconv.Itoa(int(roundId))
}
