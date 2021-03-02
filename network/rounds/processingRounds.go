///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

// File for storing info about which rounds are processing

import (
	"crypto/md5"
	"encoding/binary"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
)

type status struct {
	failCount  uint
	processing bool
	done bool
}

// processing struct with a lock so it can be managed with concurrent threads.
type processing struct {
	rounds map[hashID]*status
	sync.RWMutex
}

type hashID [16]byte

func makeHashID(round id.Round, eph ephemeral.Id, source *id.ID)hashID{
	h := md5.New()
	ridbytes := make([]byte, 8)
	binary.BigEndian.PutUint64(ridbytes, uint64(round))
	h.Write(ridbytes)
	h.Write(eph[:])
	h.Write(source.Bytes())

	hBytes := h.Sum(nil)
	hid := hashID{}
	copy(hid[:], hBytes)
	return hid
}

// newProcessingRounds returns a new processing rounds object.
func newProcessingRounds() *processing {
	return &processing{
		rounds: make(map[hashID]*status),
	}
}

// Process adds a round to the list of processing rounds. The returned boolean
// is true when the round changes from "not processing" to "processing". The
// returned count is the number of times the round has been processed.
func (pr *processing) Process(round id.Round, eph ephemeral.Id, source *id.ID) (bool, bool, uint) {
	hid := makeHashID(round, eph, source)

	pr.Lock()
	defer pr.Unlock()

	if rs, ok := pr.rounds[hid]; ok {
		if rs.processing {
			return false, false, rs.failCount
		} else if rs.done{
			return false, true, 0
		}
		rs.processing = true

		return true, false, rs.failCount
	}

	pr.rounds[hid] = &status{
		failCount:  0,
		processing: true,
	}

	return true, false, 0
}

// IsProcessing determines if a round ID is marked as processing.
func (pr *processing) IsProcessing(round id.Round, eph ephemeral.Id, source *id.ID) bool {
	hid := makeHashID(round, eph, source)
	pr.RLock()
	defer pr.RUnlock()

	if rs, ok := pr.rounds[hid]; ok {
		return rs.processing
	}

	return false
}

// Fail sets a round's processing status to failed and increments its fail
// counter so that it can be retried.
func (pr *processing) Fail(round id.Round, eph ephemeral.Id, source *id.ID) {
	hid := makeHashID(round, eph, source)
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[hid]; ok {
		rs.processing = false
		rs.failCount++
	}
}

// Done deletes a round from the processing list.
func (pr *processing) Done(round id.Round, eph ephemeral.Id, source *id.ID) {
	hid := makeHashID(round, eph, source)
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[hid]; ok {
		rs.processing = false
		rs.done = true
	}
}

// NotProcessing sets a round's processing status to failed so that it can be
// retried but does not increment its fail counter.
func (pr *processing) NotProcessing(round id.Round, eph ephemeral.Id, source *id.ID) {
	hid := makeHashID(round, eph, source)
	pr.Lock()
	defer pr.Unlock()
	if rs, ok := pr.rounds[hid]; ok {
		rs.processing = false
	}
}

// Done deletes a round from the processing list.
func (pr *processing) Delete(round id.Round, eph ephemeral.Id, source *id.ID) {
	hid := makeHashID(round, eph, source)
	pr.Lock()
	defer pr.Unlock()
	delete(pr.rounds, hid)
}
