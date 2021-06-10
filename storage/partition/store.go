///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"encoding/binary"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"sync"
	"time"
)

type multiPartID [16]byte

const packagePrefix = "Partition"
const clearPartitionThreshold = 24 * time.Hour
const activePartitions = "activePartitions"
const activePartitionVersion = 0

type Store struct {
	multiParts  map[multiPartID]*multiPartMessage
	activeParts []*multiPartMessage
	kv          *versioned.KV
	mux         sync.Mutex
}

func New(kv *versioned.KV) *Store {
	return &Store{
		multiParts:  make(map[multiPartID]*multiPartMessage),
		activeParts: make([]*multiPartMessage, 0),
		kv:          kv.Prefix(packagePrefix),
	}
}

func Load(kv *versioned.KV) *Store {
	partitionStore := &Store{
		multiParts:  make(map[multiPartID]*multiPartMessage),
		activeParts: make([]*multiPartMessage, 0),
		kv:          kv.Prefix(packagePrefix),
	}

	partitionStore.loadActivePartitions()

	partitionStore.Prune()

	return partitionStore
}

func (s *Store) AddFirst(partner *id.ID, mt message.Type, messageID uint64,
	partNum, numParts uint8, senderTimestamp, storageTimestamp time.Time,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.AddFirst(mt, partNum, numParts, senderTimestamp, storageTimestamp, part)
	msg, ok := mpm.IsComplete(relationshipFingerprint)

	if !ok {
		s.activeParts = append(s.activeParts, mpm)
		s.saveActiveParts()
	}

	return msg, ok
}

func (s *Store) Add(partner *id.ID, messageID uint64, partNum uint8,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.Add(partNum, part)

	msg, ok := mpm.IsComplete(relationshipFingerprint)
	if !ok {
		s.activeParts = append(s.activeParts, mpm)
		s.saveActiveParts()
	}

	return msg, ok
}

// Prune clear old messages on it's stored timestamp
func (s *Store) Prune() {
	s.mux.Lock()
	defer s.mux.Unlock()
	now := netTime.Now()
	for _, mpm := range s.activeParts {
		if now.Sub(mpm.StorageTimestamp) >= clearPartitionThreshold {
			mpm.delete()
			mpID := getMultiPartID(mpm.Sender, mpm.MessageID)
			delete(s.multiParts, mpID)
		}
	}
}

func (s *Store) load(partner *id.ID, messageID uint64) *multiPartMessage {
	mpID := getMultiPartID(partner, messageID)
	s.mux.Lock()
	mpm, ok := s.multiParts[mpID]
	if !ok {
		mpm = loadOrCreateMultiPartMessage(partner, messageID, s.kv)
		s.multiParts[mpID] = mpm
	}
	s.mux.Unlock()

	return mpm
}

func (s *Store) saveActiveParts() {
	data, err := json.Marshal(s.activeParts)
	if err != nil {
		jww.FATAL.Panicf("Could not save active partitions: %v", err)
	}

	obj := versioned.Object{
		Version:   activePartitionVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = s.kv.Set(activePartitions, activePartitionVersion, &obj)
	if err != nil {
		jww.FATAL.Panicf("Could not save active partitions: %v", err)
	}
}

func (s *Store) loadActivePartitions() {
	s.mux.Lock()
	defer s.mux.Unlock()
	obj, err := s.kv.Get(activePartitions, activePartitionVersion)
	if err != nil {
		jww.DEBUG.Printf("Could not load active partitions: %v", err)
		return
	}

	err = json.Unmarshal(obj.Data, &s.activeParts)
	if err != nil {
		jww.FATAL.Panicf("Could not load active partitions: %v", err)
	}

}

func getMultiPartID(partner *id.ID, messageID uint64) multiPartID {
	h, _ := blake2b.New256(nil)

	h.Write(partner.Bytes())
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, messageID)
	h.Write(b)

	var mpID multiPartID
	copy(mpID[:], h.Sum(nil))

	return mpID
}
