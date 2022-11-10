////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

// UncheckedRoundStore stores rounds to retry for message retrieval.
type UncheckedRoundStore struct {
	list map[roundIdentity]UncheckedRound
	mux  sync.RWMutex
	kv   *versioned.KV
}

// NewUncheckedStore is a constructor for a UncheckedRoundStore.
func NewUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {
	kv = kv.Prefix(uncheckedRoundPrefix)

	urs := &UncheckedRoundStore{
		list: make(map[roundIdentity]UncheckedRound, 0),
		kv:   kv,
	}

	return urs, urs.save()
}

// NewOrLoadUncheckedStore is a constructor for a UncheckedRoundStore.
func NewOrLoadUncheckedStore(kv *versioned.KV) *UncheckedRoundStore {
	kv = kv.Prefix(uncheckedRoundPrefix)

	urs, err := LoadUncheckedStore(kv)
	if err == nil {
		return urs
	}

	urs = &UncheckedRoundStore{
		list: make(map[roundIdentity]UncheckedRound, 0),
		kv:   kv,
	}

	if err = urs.save(); err != nil {
		jww.FATAL.Panicf("Failed to save a new unchecked round store: %v", err)
	}

	return urs
}

// LoadUncheckedStore loads a deserializes a UncheckedRoundStore from memory.
func LoadUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {
	kv = kv.Prefix(uncheckedRoundPrefix)
	vo, err := kv.Get(uncheckedRoundKey, uncheckedRoundVersion)
	if err != nil {
		return nil, err
	}

	urs := &UncheckedRoundStore{
		list: make(map[roundIdentity]UncheckedRound),
		kv:   kv,
	}

	err = urs.unmarshal(vo.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load rounds from storage")
	}

	return urs, err
}

// AddRound adds a round to check on the list and saves to memory.
func (s *UncheckedRoundStore) AddRound(rid id.Round, ri *pb.RoundInfo,
	source *id.ID, ephID ephemeral.Id) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	roundId := newRoundIdentity(rid, source, ephID)

	stored, exists := s.list[roundId]

	if !exists || (stored.Info == nil && ri != nil) {
		newUncheckedRound := UncheckedRound{
			Id:   rid,
			Info: ri,
			Identity: Identity{
				EpdId:  ephID,
				Source: source,
			},
			LastCheck: netTime.Now(),
			NumChecks: stored.NumChecks,
		}

		s.list[roundId] = newUncheckedRound
		return s.save()
	}

	return nil
}

// GetRound retrieves an UncheckedRound from the map, if it exists.
func (s *UncheckedRoundStore) GetRound(rid id.Round, recipient *id.ID,
	ephId ephemeral.Id) (UncheckedRound, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	rnd, exists := s.list[newRoundIdentity(rid, recipient, ephId)]
	return rnd, exists
}

func (s *UncheckedRoundStore) GetList(_ *testing.T) map[roundIdentity]UncheckedRound {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.list
}

// IterateOverList retrieves the list of rounds.
func (s *UncheckedRoundStore) IterateOverList(iterator func(rid id.Round,
	rnd UncheckedRound)) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	for _, rnd := range s.list {
		if rnd.beingChecked {
			continue
		}
		jww.DEBUG.Printf("Round for lookup: %d, %+v\n", rnd.Id, rnd)
		go func(localRid id.Round, localRnd UncheckedRound) {
			iterator(localRid, localRnd)
		}(rnd.Id, rnd)
	}
}

// EndCheck increments the amount of checks performed on this stored
// round.
func (s *UncheckedRoundStore) EndCheck(rid id.Round, recipient *id.ID,
	ephId ephemeral.Id) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	nri := newRoundIdentity(rid, recipient, ephId)
	rnd, exists := s.list[nri]
	if !exists {
		return errors.Errorf("round %d could not be found in RAM", rid)
	}

	rnd.beingChecked = false
	s.list[nri] = rnd
	return nil
}

// IncrementCheck increments the amount of checks performed on this stored
// round.
func (s *UncheckedRoundStore) IncrementCheck(rid id.Round, recipient *id.ID,
	ephId ephemeral.Id) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	nri := newRoundIdentity(rid, recipient, ephId)
	rnd, exists := s.list[nri]
	if !exists {
		return errors.Errorf("round %d could not be found in RAM", rid)
	}

	// If a round has been checked the maximum amount of times, then bail the
	// round by removing it from store and no longer checking
	if rnd.NumChecks >= maxChecks {
		if err := s.remove(rid, rnd.Identity.Source, ephId); err != nil {
			return errors.WithMessagef(err, "Round %d reached maximum checks "+
				"but could not be removed", rid)
		}
		return nil
	}

	// Update the rounds state
	rnd.LastCheck = netTime.Now()
	rnd.NumChecks++
	rnd.storageUpToDate = false
	s.list[nri] = rnd
	return s.save()
}

// Remove deletes a round from UncheckedRoundStore's list and from storage.
func (s *UncheckedRoundStore) Remove(rid id.Round, source *id.ID,
	ephId ephemeral.Id) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.remove(rid, source, ephId)
}

// Remove is a helper function which removes the round from
// UncheckedRoundStore's list. Note that this method is unsafe and should only
// be used by methods with a lock.
func (s *UncheckedRoundStore) remove(rid id.Round, recipient *id.ID,
	ephId ephemeral.Id) error {
	roundId := newRoundIdentity(rid, recipient, ephId)
	ur, exists := s.list[roundId]
	if !exists {
		return errors.Errorf("round %d does not exist in store", rid)
	}

	delete(s.list, roundId)
	if err := s.save(); err != nil {
		return errors.WithMessagef(err,
			"Failed to delete round %d from unchecked round store", rid)
	}

	// Do not delete round infos if none exist
	if ur.Info == nil {
		return nil
	}

	if err := deleteRoundInfo(s.kv, rid, recipient, ephId); err != nil {
		return errors.WithMessagef(err,
			"Failed to delete round %d's roundinfo from unchecked round store, "+
				"round itself deleted. This is a storage leak", rid)
	}

	return nil
}

// save stores the information from the round list into storage.
func (s *UncheckedRoundStore) save() error {
	// Store list of rounds
	data, err := s.marshal()
	if err != nil {
		return errors.WithMessagef(err,
			"Could not marshal data for unchecked rounds")
	}

	// Create the versioned object
	obj := &versioned.Object{
		Version:   uncheckedRoundVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = s.kv.Set(uncheckedRoundKey, obj)
	if err != nil {
		return errors.WithMessagef(err,
			"Could not store data for unchecked rounds")
	}

	return nil
}

// marshal is a helper function which serializes all rounds in list to bytes.
func (s *UncheckedRoundStore) marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Write number of rounds the buffer
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(len(s.list)))
	buf.Write(b)

	for rid, rnd := range s.list {
		rndData, err := rnd.marshal(s.kv)
		if err != nil {
			return nil, errors.WithMessagef(err,
				"Failed to marshal round %d", rid)
		}

		buf.Write(rndData)

	}

	return buf.Bytes(), nil
}

// unmarshal deserializes an UncheckedRound from its stored byte data.
func (s *UncheckedRoundStore) unmarshal(data []byte) error {
	buff := bytes.NewBuffer(data)

	// get number of rounds in list
	length := binary.BigEndian.Uint32(buff.Next(8))

	for i := 0; i < int(length); i++ {
		rnd := UncheckedRound{}
		err := rnd.unmarshal(s.kv, buff)
		if err != nil {
			return errors.WithMessage(err,
				"Failed to unmarshal rounds in storage")
		}

		s.list[newRoundIdentity(rnd.Id, rnd.Source, rnd.EpdId)] = rnd
	}

	return nil
}
