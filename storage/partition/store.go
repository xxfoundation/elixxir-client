///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"sync"
	"time"
)

type multiPartID [16]byte

const packagePrefix = "Partition"
const clearPartitionInterval = 5*time.Hour
const clearPartitionThreshold = 24*time.Hour

type Store struct {
	multiParts map[multiPartID]*multiPartMessage
	kv         *versioned.KV
	mux        sync.Mutex
}

func New(kv *versioned.KV) *Store {
	return &Store{
		multiParts: make(map[multiPartID]*multiPartMessage),
		kv:         kv.Prefix(packagePrefix),
	}
}

func (s *Store) AddFirst(partner *id.ID, mt message.Type, messageID uint64,
	partNum, numParts uint8, senderTimestamp, storageTimestamp time.Time,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.AddFirst(mt, partNum, numParts, senderTimestamp, storageTimestamp, part)

	return mpm.IsComplete(relationshipFingerprint)
}

func (s *Store) Add(partner *id.ID, messageID uint64, partNum uint8,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.Add(partNum, part)

	return mpm.IsComplete(relationshipFingerprint)
}

// ClearMessages periodically clear old messages on it's stored timestamp
func (s *Store) ClearMessages() stoppable.Stoppable  {
	stop := stoppable.NewSingle("clearPartition")
	t := time.NewTicker(clearPartitionInterval)
	go func() {
		for {
			select {
			case <-stop.Quit():
				stop.ToStopped()
				t.Stop()
				return
			case <-t.C:
				s.clearMessages()
			}
		}
	}()
	return stop
}

// clearMessages is a helper function which clears
// old messages from storage
func (s *Store) clearMessages()  {
	s.mux.Lock()
	now := netTime.Now()
	for mpmId, mpm := range s.multiParts {
		if now.Sub(mpm.StorageTimestamp) >= clearPartitionThreshold {
			mpm.delete()
			delete(s.multiParts, mpmId)
		}
	}
	s.mux.Unlock()
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
