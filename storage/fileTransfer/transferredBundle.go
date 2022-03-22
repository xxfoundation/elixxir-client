////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage keys and versions.
const (
	transferredBundleVersion = 0
	transferredBundleKey     = "FileTransferBundle"
	inProgressKey            = "inProgressTransfers"
	finishedKey              = "finishedTransfers"
)

// Error messages.
const (
	loadTransferredBundleErr = "failed to get transferredBundle from storage: %+v"
)

// transferredBundle lists the file parts sent per round ID.
type transferredBundle struct {
	list     map[id.Round][]uint16
	numParts uint16
	key      string
	kv       *versioned.KV
}

// newTransferredBundle generates a new transferredBundle and saves it to
// storage.
func newTransferredBundle(key string, kv *versioned.KV) (
	*transferredBundle, error) {
	tb := &transferredBundle{
		list: make(map[id.Round][]uint16),
		key:  key,
		kv:   kv,
	}

	return tb, tb.save()
}

// addPartNums adds a round to the map with the specified part numbers.
func (tb *transferredBundle) addPartNums(rid id.Round, partNums ...uint16) error {
	tb.list[rid] = append(tb.list[rid], partNums...)

	// Increment number of parts
	tb.numParts += uint16(len(partNums))

	return tb.save()
}

// getPartNums returns the list of part numbers for the given round ID. If there
// are no part numbers for the round ID, then it returns false.
func (tb *transferredBundle) getPartNums(rid id.Round) ([]uint16, bool) {
	partNums, exists := tb.list[rid]
	return partNums, exists
}

// getNumParts returns the number of file parts stored in list.
func (tb *transferredBundle) getNumParts() uint16 {
	return tb.numParts
}

// deletePartNums deletes the round and its part numbers from the map.
func (tb *transferredBundle) deletePartNums(rid id.Round) error {
	// Decrement number of parts
	tb.numParts -= uint16(len(tb.list[rid]))

	// Remove from list
	delete(tb.list, rid)

	// Remove from storage
	return tb.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadTransferredBundle loads a transferredBundle from storage.
func loadTransferredBundle(key string, kv *versioned.KV) (*transferredBundle,
	error) {
	vo, err := kv.Get(makeTransferredBundleKey(key), transferredBundleVersion)
	if err != nil {
		return nil, errors.Errorf(loadTransferredBundleErr, err)
	}

	tb := &transferredBundle{
		list: make(map[id.Round][]uint16),
		key:  key,
		kv:   kv,
	}

	tb.unmarshal(vo.Data)

	return tb, nil
}

// save stores the transferredBundle to storage.
func (tb *transferredBundle) save() error {
	obj := &versioned.Object{
		Version:   transferredBundleVersion,
		Timestamp: netTime.Now(),
		Data:      tb.marshal(),
	}

	return tb.kv.Set(
		makeTransferredBundleKey(tb.key), transferredBundleVersion, obj)
}

// delete remove the transferredBundle from storage.
func (tb *transferredBundle) delete() error {
	return tb.kv.Delete(
		makeTransferredBundleKey(tb.key), transferredBundleVersion)
}

// marshal serialises the map into a byte slice.
func (tb *transferredBundle) marshal() []byte {
	// Create buffer
	buff := bytes.NewBuffer(nil)

	for rid, partNums := range tb.list {
		// Write round ID to buffer
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(rid))
		buff.Write(b)

		// Write number of part numbers to buffer
		b = make([]byte, 2)
		binary.LittleEndian.PutUint16(b, uint16(len(partNums)))
		buff.Write(b)

		// Write list of part numbers to buffer
		for _, partNum := range partNums {
			b = make([]byte, 2)
			binary.LittleEndian.PutUint16(b, partNum)
			buff.Write(b)
		}
	}

	return buff.Bytes()
}

// unmarshal deserializes the byte slice into the transferredBundle.
func (tb *transferredBundle) unmarshal(b []byte) {
	buff := bytes.NewBuffer(b)

	// Iterate over all map entries
	for n := buff.Next(8); len(n) == 8; n = buff.Next(8) {
		// get the round ID from the first 8 bytes
		rid := id.Round(binary.LittleEndian.Uint64(n))

		// get number of part numbers listed
		partNumsLen := binary.LittleEndian.Uint16(buff.Next(2))

		// Increment number of parts
		tb.numParts += partNumsLen

		// Initialize part number list to the correct size
		tb.list[rid] = make([]uint16, 0, partNumsLen)

		// Add all part numbers to list
		for i := uint16(0); i < partNumsLen; i++ {
			partNum := binary.LittleEndian.Uint16(buff.Next(2))
			tb.list[rid] = append(tb.list[rid], partNum)
		}
	}
}

// makeTransferredBundleKey concatenates a unique key with a constant to create
// a key for saving a transferredBundle to storage.
func makeTransferredBundleKey(key string) string {
	return transferredBundleKey + key
}
