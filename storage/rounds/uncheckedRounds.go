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
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

const (
	uncheckedRoundVersion = 0
	uncheckedRoundPrefix  = "uncheckedRoundPrefix"
	// Key to store rounds
	uncheckedRoundKey = "uncheckRounds"
	// Key to store individual round
	// Housekeeping constant (used for serializing uint64 ie id.Round)
	uint64Size = 8
	// Maximum checks that can be performed on a round. Intended so that
	// a round is checked no more than 1 week approximately (network/rounds.cappedTries + 7)
	maxChecks = 14
)

// Round identity information used in message retrieval
// Derived from reception.Identity saving data needed
// for message retrieval
type Identity struct {
	EpdId  ephemeral.Id
	Source *id.ID
}

// Unchecked round structure is rounds which failed on message retrieval
// These rounds are stored for retry of message retrieval
type UncheckedRound struct {
	Info *pb.RoundInfo
	Identity
	// Timestamp in which round has last been checked
	LastCheck time.Time
	// Number of times a round has been checked
	NumChecks uint64
}

// marshal serializes UncheckedRound r into a byte slice
func (r UncheckedRound) marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Write the round info
	b := make([]byte, uint64Size)
	infoBytes, err := proto.Marshal(r.Info)
	binary.LittleEndian.PutUint64(b, uint64(len(infoBytes)))
	buf.Write(b)
	buf.Write(infoBytes)

	b = make([]byte, uint64Size)

	// Write the round identity info
	buf.Write(r.Identity.EpdId[:])
	if r.Source != nil {
		buf.Write(r.Identity.Source.Marshal())
	} else {
		buf.Write(make([]byte, id.ArrIDLen))
	}

	// Write the time stamp bytes
	tsBytes, err := r.LastCheck.MarshalBinary()
	if err != nil {
		return nil, errors.WithMessage(err, "Could not marshal timestamp ")
	}
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(len(tsBytes)))
	buf.Write(b)
	buf.Write(tsBytes)

	// Write the number of tries for this round
	b = make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, r.NumChecks)
	buf.Write(b)

	return buf.Bytes(), nil
}

// unmarshal deserializes round data from buff into UncheckedRound r
func (r *UncheckedRound) unmarshal(buff *bytes.Buffer) error {
	// Deserialize the roundInfo
	roundInfoLen := binary.LittleEndian.Uint64(buff.Next(uint64Size))
	roundInfoBytes := buff.Next(int(roundInfoLen))
	ri := &pb.RoundInfo{}
	if err := proto.Unmarshal(roundInfoBytes, ri); err != nil {
		return errors.WithMessagef(err, "Failed to unmarshal roundInfo")
	}
	r.Info = ri

	// Deserialize the round identity information
	copy(r.EpdId[:], buff.Next(uint64Size))

	sourceId, err := id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return errors.WithMessage(err, "Failed to unmarshal round identity.source")
	}

	r.Source = sourceId

	// Deserialize the timestamp bytes
	timestampLen := binary.LittleEndian.Uint64(buff.Next(uint64Size))
	tsByes := buff.Next(int(uint64(timestampLen)))
	if err = r.LastCheck.UnmarshalBinary(tsByes); err != nil {
		return errors.WithMessage(err, "Failed to unmarshal round timestamp")
	}

	r.NumChecks = binary.LittleEndian.Uint64(buff.Next(uint64Size))

	return nil
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
	vo, err := kv.Get(uncheckedRoundKey, uncheckedRoundVersion)
	if err != nil {
		return nil, err
	}

	urs := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound),
		kv:   kv,
	}

	err = urs.unmarshal(vo.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load rounds from storage")
	}

	return urs, err
}

// Adds a round to check on the list and saves to memory
func (s *UncheckedRoundStore) AddRound(ri *pb.RoundInfo, ephID ephemeral.Id, source *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	rid := id.Round(ri.ID)

	if _, exists := s.list[rid]; !exists {
		newUncheckedRound := UncheckedRound{
			Info: ri,
			Identity: Identity{
				EpdId:  ephID,
				Source: source,
			},
			LastCheck: netTime.Now(),
			NumChecks: 0,
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

	// If a round has been checked the maximum amount of times,
	// we bail the round by removing it from store and no longer checking
	if rnd.NumChecks >= maxChecks {
		if err := s.remove(rid); err != nil {
			return errors.WithMessagef(err, "Round %d reached maximum checks "+
				"but could not be removed", rid)
		}
		return nil
	}

	// Update the rounds state
	rnd.LastCheck = netTime.Now()
	rnd.NumChecks++
	s.list[rid] = rnd
	return s.save()
}

// Remove deletes a round from UncheckedRoundStore's list and from storage
func (s *UncheckedRoundStore) Remove(rid id.Round) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.remove(rid)
}

// Remove is a helper function which removes the round from UncheckedRoundStore's list
// Note this method is unsafe and should only be used by methods with a lock
func (s *UncheckedRoundStore) remove(rid id.Round) error {
	if _, exists := s.list[rid]; !exists {
		return errors.Errorf("round %d does not exist in store", rid)
	}
	delete(s.list, rid)
	return s.save()
}

// save stores the information from the round list into storage
func (s *UncheckedRoundStore) save() error {
	// Store list of rounds
	data, err := s.marshal()
	if err != nil {
		return errors.WithMessagef(err, "Could not marshal data for unchecked rounds")
	}

	// Create the versioned object
	obj := &versioned.Object{
		Version:   uncheckedRoundVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = s.kv.Set(uncheckedRoundKey, uncheckedRoundVersion, obj)
	if err != nil {
		return errors.WithMessagef(err, "Could not store data for unchecked rounds")
	}

	return nil
}

// marshal is a helper function which serializes all rounds in list to bytes
func (s *UncheckedRoundStore) marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Write number of rounds the buffer
	b := make([]byte, 8)
	binary.PutVarint(b, int64(len(s.list)))
	buf.Write(b)

	for rid, rnd := range s.list {
		rndData, err := rnd.marshal()
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to marshal round %d", rid)
		}

		buf.Write(rndData)

	}

	return buf.Bytes(), nil
}

// unmarshal deserializes an UncheckedRound from its stored byte data
func (s *UncheckedRoundStore) unmarshal(data []byte) error {
	buff := bytes.NewBuffer(data)
	// Get number of rounds in list
	length, _ := binary.Varint(buff.Next(8))

	for i := 0; i < int(length); i++ {
		rnd := UncheckedRound{}
		err := rnd.unmarshal(buff)
		if err != nil {
			return errors.WithMessage(err, "Failed to unmarshal rounds in storage")
		}

		s.list[id.Round(rnd.Info.ID)] = rnd
	}

	return nil
}