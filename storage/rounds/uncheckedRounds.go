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
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"testing"
	"time"
)

const (
	uncheckedRoundVersion = 0
	roundInfoVersion      = 0
	uncheckedRoundPrefix  = "uncheckedRoundPrefix"
	roundKeyPrefix        = "roundInfo:"

	// Key to store rounds
	uncheckedRoundKey = "uncheckRounds"

	// Housekeeping constant (used for serializing uint64 ie id.Round)
	uint64Size = 8

	// Maximum checks that can be performed on a round. Intended so that a round
	// is checked no more than 1 week approximately (network/rounds.cappedTries + 7)
	maxChecks = 14
)

// Identity contains round identity information used in message retrieval.
// Derived from reception.Identity saving data needed for message retrieval.
type Identity struct {
	EpdId  ephemeral.Id
	Source *id.ID
}

// UncheckedRound contains rounds that failed on message retrieval. These rounds
// are stored for retry of message retrieval.
type UncheckedRound struct {
	Info *pb.RoundInfo
	Id   id.Round

	Identity
	// Timestamp in which round has last been checked
	LastCheck time.Time
	// Number of times a round has been checked
	NumChecks uint64
}

// marshal serializes UncheckedRound r into a byte slice.
func (r UncheckedRound) marshal(kv *versioned.KV) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Store teh round info
	if r.Info != nil {
		if err := storeRoundInfo(kv, r.Info, r.Source, r.EpdId); err != nil {
			return nil, errors.WithMessagef(err,
				"failed to marshal unchecked rounds")
		}
	}

	// Marshal the round ID
	b := make([]byte, uint64Size)
	binary.LittleEndian.PutUint64(b, uint64(r.Id))
	buf.Write(b)

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

// unmarshal deserializes round data from buff into UncheckedRound r.
func (r *UncheckedRound) unmarshal(kv *versioned.KV, buff *bytes.Buffer) error {
	// Deserialize the roundInfo
	r.Id = id.Round(binary.LittleEndian.Uint64(buff.Next(uint64Size)))

	// Deserialize the round identity information
	copy(r.EpdId[:], buff.Next(uint64Size))

	sourceId, err := id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return errors.WithMessagef(err,
			"Failed to unmarshal round identity.source of %d", r.Id)
	}

	r.Source = sourceId

	// Deserialize the timestamp bytes
	timestampLen := binary.LittleEndian.Uint64(buff.Next(uint64Size))
	tsByes := buff.Next(int(timestampLen))
	if err = r.LastCheck.UnmarshalBinary(tsByes); err != nil {
		return errors.WithMessagef(err,
			"Failed to unmarshal round timestamp of %d", r.Id)
	}

	r.NumChecks = binary.LittleEndian.Uint64(buff.Next(uint64Size))

	r.Info, _ = loadRoundInfo(kv, r.Id, r.Source, r.EpdId)

	return nil
}

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

	if !exists || stored.Info == nil {
		newUncheckedRound := UncheckedRound{
			Id:   rid,
			Info: ri,
			Identity: Identity{
				EpdId:  ephID,
				Source: source,
			},
			LastCheck: netTime.Now(),
			NumChecks: 0,
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

func (s *UncheckedRoundStore) GetList(*testing.T) map[roundIdentity]UncheckedRound {
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
		jww.DEBUG.Printf("rnd for lookup: %d, %+v\n", rnd.Id, rnd)
		go func(localRid id.Round,
			localRnd UncheckedRound) {
			iterator(localRid, localRnd)
		}(rnd.Id, rnd)
	}
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
	err = s.kv.Set(uncheckedRoundKey, uncheckedRoundVersion, obj)
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

func storeRoundInfo(kv *versioned.KV, info *pb.RoundInfo, recipient *id.ID,
	ephID ephemeral.Id) error {
	now := netTime.Now()

	data, err := proto.Marshal(info)
	if err != nil {
		return errors.WithMessagef(err,
			"Failed to store individual unchecked round")
	}

	obj := versioned.Object{
		Version:   roundInfoVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(
		roundKey(id.Round(info.ID), recipient, ephID), roundInfoVersion, &obj)
}

func loadRoundInfo(kv *versioned.KV, id id.Round, recipient *id.ID,
	ephID ephemeral.Id) (*pb.RoundInfo, error) {

	vo, err := kv.Get(roundKey(id, recipient, ephID), roundInfoVersion)
	if err != nil {
		return nil, err
	}

	ri := &pb.RoundInfo{}
	if err = proto.Unmarshal(vo.Data, ri); err != nil {
		return nil, errors.WithMessagef(err, "Failed to unmarshal roundInfo")
	}

	return ri, nil
}

func deleteRoundInfo(kv *versioned.KV, id id.Round, recipient *id.ID,
	ephID ephemeral.Id) error {
	return kv.Delete(roundKey(id, recipient, ephID), roundInfoVersion)
}

func roundKey(roundID id.Round, recipient *id.ID, ephID ephemeral.Id) string {
	return roundKeyPrefix + newRoundIdentity(roundID, recipient, ephID).String()
}
