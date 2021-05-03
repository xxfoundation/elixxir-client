package rounds

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

const (
	uncheckRoundVersion = 0
	uncheckRoundPrefix  = "uncheckedRoundPrefix"
	uncheckedRoundKey   = "uncheckRounds"
)

// Unchecked round structure is rounds which failed on message retrieval
// These rounds are stored for retry of message retrieval
type UncheckedRound struct {
	//
	Rid id.Round
	roundIdentity
	// Timestamp in which round has been stored
	Ts time.Time
	// Number of times a round has been checked
	NumTries uint
}

// Round identity information used in message retrieval
// Derived from reception.Identity
type roundIdentity struct {
	EpdId  ephemeral.Id
	Source *id.ID
}

// Storage object saving rounds to retry for message retrieval
type UncheckedRoundStore struct {
	list map[id.Round]UncheckedRound
	mux  sync.RWMutex
	kv   *versioned.KV
}

// Constructor for a UncheckedRoundStore
func NewUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {
	kv = kv.Prefix(uncheckRoundPrefix)

	urs := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound, 0),
		kv:   kv,
	}

	return urs, urs.save()

}

// Loads an deserializes a UncheckedRoundStore from memory
func LoadUncheckedStore(kv *versioned.KV) (*UncheckedRoundStore, error) {
	urs := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound, 0),
		kv:   kv,
	}

	kv = kv.Prefix(uncheckRoundPrefix)
	uncheckedRoundsData, err := kv.Get(uncheckedRoundKey, uncheckRoundVersion)
	if err != nil {
		return nil, err
	}

	urs.list, err = deserializeRounds(uncheckedRoundsData.Data)

	return urs, err
}

// Adds a round to check on the list and saves to memory
func (s *UncheckedRoundStore) AddRound(rid id.Round, ephID ephemeral.Id, source *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	newUncheckedRound := UncheckedRound{
		Rid: rid,
		roundIdentity: roundIdentity{
			EpdId:  ephID,
			Source: source,
		},
		Ts:       netTime.Now(),
		NumTries: 0,
	}

	s.list[rid] = newUncheckedRound

	return s.save()
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
func (s *UncheckedRoundStore) IncrementCheck(rid id.Round) {
	s.mux.Lock()
	defer s.mux.Unlock()
	rnd, exists := s.list[rid]
	if !exists {
		return
	}

	rnd.NumTries++
}

func (s *UncheckedRoundStore) save() error {
	data, err := s.serializeRounds()
	if err != nil {
		return errors.WithMessage(err, "Failed to serialize")
	}
	// Create the versioned object
	obj := &versioned.Object{
		Version:   uncheckRoundVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return s.kv.Set(uncheckedRoundKey, uncheckRoundVersion, obj)
}

// Helper function which serializes all rounds into byte data
func (s *UncheckedRoundStore) serializeRounds() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// Write number of rounds the buffer
	b := make([]byte, 8)
	binary.PutVarint(b, int64(len(s.list)))
	buf.Write(b)

	for _, rnd := range s.list {
		// Write the round ID
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(rnd.Rid))
		buf.Write(b)

		// Write the round identity info
		buf.Write(rnd.roundIdentity.EpdId[:])
		if rnd.Source != nil {
			buf.Write(rnd.roundIdentity.Source.Marshal())
		} else {
			buf.Write(make([]byte, id.ArrIDLen))
		}

		// Write the time stamp bytes
		tsBytes, err := rnd.Ts.MarshalBinary()
		if err != nil {
			return nil, errors.WithMessagef(err, "Could not marshal timestamp for round %d", rnd.Rid)
		}
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(len(tsBytes)))
		buf.Write(b)
		buf.Write(tsBytes)

		// Write the number of tries for this round
		b = make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(rnd.NumTries))
		buf.Write(b)
	}

	return buf.Bytes(), nil
}

// Helper function which deserializes round data retrieved from storage into
// a map of UncheckedRound's
func deserializeRounds(data []byte) (map[id.Round]UncheckedRound, error) {
	roundMap := make(map[id.Round]UncheckedRound)
	buff := bytes.NewBuffer(data)
	// Get number of rounds in list
	length, _ := binary.Varint(buff.Next(8))

	for i := 0; i < int(length); i++ {
		rnd := UncheckedRound{}
		rid, _ := binary.Varint(buff.Next(8))
		rnd.Rid = id.Round(rid)

		// Deserialize the round identity information
		copy(rnd.EpdId[:], buff.Next(8))

		sourceId, err := id.Unmarshal(buff.Next(id.ArrIDLen))
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to unmarshal round identity.source")
		}

		rnd.Source = sourceId

		// Deserialize the timestamp bytes
		timestampLen, _ := binary.Varint(buff.Next(8))
		tsByes := buff.Next(int(timestampLen))
		if err = rnd.Ts.UnmarshalBinary(tsByes); err != nil {
			return nil, errors.WithMessage(err, "Failed to unmarshal round timestamp")
		}

		numTries, _ := binary.Varint(buff.Next(8))
		rnd.NumTries = uint(numTries)

		roundMap[id.Round(rid)] = rnd

	}

	return roundMap, nil

}
