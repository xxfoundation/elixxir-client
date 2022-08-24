///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/e2e"
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
	activeParts map[*multiPartMessage]bool
	kv          *versioned.KV
	mux         sync.Mutex
}

func NewOrLoad(kv *versioned.KV) *Store {
	partitionStore := &Store{
		multiParts:  make(map[multiPartID]*multiPartMessage),
		activeParts: make(map[*multiPartMessage]bool),
		kv:          kv.Prefix(packagePrefix),
	}

	partitionStore.loadActivePartitions()

	partitionStore.prune()

	return partitionStore
}

func (s *Store) AddFirst(partner *id.ID, mt catalog.MessageType,
	messageID uint64, partNum, numParts uint8, senderTimestamp,
	storageTimestamp time.Time, part []byte, relationshipFingerprint []byte,
	residue e2e.KeyResidue) (
	receive.Message, e2e.KeyResidue, bool) {

	mpm := s.load(partner, messageID)
	mpm.AddFirst(mt, partNum, numParts, senderTimestamp, storageTimestamp, part)
	if bytes.Equal(residue.Marshal(), []byte{}) {
		// fixme: should this error or crash?
		jww.WARN.Printf("Key reside from first message " +
			"is empty, continuing...")
	}

	mpm.KeyResidue = residue
	msg, ok := mpm.IsComplete(relationshipFingerprint)

	s.mux.Lock()
	defer s.mux.Unlock()

	keyRes := e2e.KeyResidue{}
	if !ok {
		s.activeParts[mpm] = true
		s.saveActiveParts()
	} else {
		keyRes = mpm.KeyResidue
		mpID := getMultiPartID(mpm.Sender, mpm.MessageID)
		delete(s.multiParts, mpID)
	}

	return msg, keyRes, ok
}

func (s *Store) Add(partner *id.ID, messageID uint64, partNum uint8,
	part []byte, relationshipFingerprint []byte) (
	receive.Message, e2e.KeyResidue, bool) {

	mpm := s.load(partner, messageID)

	mpm.Add(partNum, part)

	msg, ok := mpm.IsComplete(relationshipFingerprint)
	keyRes := e2e.KeyResidue{}
	if !ok {
		s.activeParts[mpm] = true
		s.saveActiveParts()
	} else {
		keyRes = mpm.KeyResidue
		mpID := getMultiPartID(mpm.Sender, mpm.MessageID)
		delete(s.multiParts, mpID)
	}

	return msg, keyRes, ok
}

// prune clears old messages on it's stored timestamp.
func (s *Store) prune() {
	s.mux.Lock()
	defer s.mux.Unlock()

	now := netTime.Now()
	for mpm := range s.activeParts {
		if now.Sub(mpm.StorageTimestamp) >= clearPartitionThreshold {
			jww.INFO.Printf("Prune partition: %v", mpm)
			mpm.mux.Lock()
			mpm.delete()
			mpID := getMultiPartID(mpm.Sender, mpm.MessageID)
			mpm.mux.Unlock()
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
	jww.INFO.Printf("Saving %d active partitions", len(s.activeParts))

	activeList := make([]*multiPartMessage, 0, len(s.activeParts))
	for mpm := range s.activeParts {
		mpm.mux.Lock()
		jww.INFO.Printf("saveActiveParts saving %v", mpm)
		activeList = append(activeList, mpm)
		mpm.mux.Unlock()
	}

	data, err := json.Marshal(&activeList)
	if err != nil {
		jww.FATAL.Panicf("Could not save active partitions: %+v", err)
	}

	obj := versioned.Object{
		Version:   activePartitionVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = s.kv.Set(activePartitions, activePartitionVersion, &obj)
	if err != nil {
		jww.FATAL.Panicf("Could not save active partitions: %+v", err)
	}
}

func (s *Store) loadActivePartitions() {
	s.mux.Lock()
	defer s.mux.Unlock()
	obj, err := s.kv.Get(activePartitions, activePartitionVersion)
	if err != nil {
		jww.DEBUG.Printf("Could not load active partitions: %+v", err)
		return
	}

	activeList := make([]*multiPartMessage, 0)
	if err = json.Unmarshal(obj.Data, &activeList); err != nil {
		jww.FATAL.Panicf("Failed to unmarshal active partitions: %+v", err)
	}
	jww.INFO.Printf("loadActivePartitions found %d active", len(activeList))

	for _, activeMpm := range activeList {
		mpm := loadOrCreateMultiPartMessage(
			activeMpm.Sender, activeMpm.MessageID, s.kv)
		s.activeParts[mpm] = true
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
