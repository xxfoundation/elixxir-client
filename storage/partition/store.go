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
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
	"sync"
	"time"
)

type multiPartID [16]byte

const packagePrefix = "Partition"

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
	partNum, numParts uint8, timestamp time.Time,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.AddFirst(mt, partNum, numParts, timestamp, part)

	return mpm.IsComplete(relationshipFingerprint)
}

func (s *Store) Add(partner *id.ID, messageID uint64, partNum uint8,
	part []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	mpm := s.load(partner, messageID)

	mpm.Add(partNum, part)

	return mpm.IsComplete(relationshipFingerprint)
}

// todo: may need a way to clean up partitioned messages when deleting a contact
// todo: Possible options:
// todo: Store partition w/ a timestamp, periodically clear old timestamps
// todo: Don't clean, storage space is negligible
// todo: Misc: Store partitions in individual files under a folder?
//func (s *Store) Delete() error  {
//
//}

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
