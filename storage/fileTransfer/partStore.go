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
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"sync"
)

// Storage keys and versions.
const (
	partsStoreVersion = 0
	partsStoreKey     = "FileTransferPart"
	partsListVersion  = 0
	partsListKey      = "FileTransferList"
)

// Error messages.
const (
	loadPartListErr = "failed to get parts list from storage: %+v"
	loadPartsErr    = "failed to load part #%d from storage: %+v"
	savePartsErr    = "failed to save part #%d to storage: %+v"
)

// partStore stores the file parts in memory/storage.
type partStore struct {
	parts    map[uint16][]byte // File parts, keyed on their number in order
	numParts uint16            // Number of parts in full file
	kv       *versioned.KV
	mux      sync.RWMutex
}

// newPartStore generates a new empty partStore and saves it to storage.
func newPartStore(kv *versioned.KV, numParts uint16) (*partStore, error) {
	// Construct empty partStore of the specified size
	ps := &partStore{
		parts:    make(map[uint16][]byte, numParts),
		numParts: numParts,
		kv:       kv,
	}

	// Save to storage
	return ps, ps.save()
}

// newPartStore generates a new empty partStore and saves it to storage.
func newPartStoreFromParts(kv *versioned.KV, parts ...[]byte) (*partStore,
	error) {

	// Construct empty partStore of the specified size
	ps := &partStore{
		parts:    partSliceToMap(parts...),
		numParts: uint16(len(parts)),
		kv:       kv,
	}

	// Save to storage
	return ps, ps.save()
}

// addPart adds a file part to the list of parts, saves it to storage, and
// regenerates the list of all file parts and saves it to storage.
func (ps *partStore) addPart(part []byte, partNum uint16) error {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	ps.parts[partNum] = make([]byte, len(part))
	copy(ps.parts[partNum], part)

	err := ps.savePart(partNum)
	if err != nil {
		return err
	}

	return ps.saveList()
}

// getPart returns the part at the given part number.
func (ps *partStore) getPart(partNum uint16) ([]byte, bool) {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	part, exists := ps.parts[partNum]
	newPart := make([]byte, len(part))
	copy(newPart, part)

	return newPart, exists
}

// getFile returns all file parts concatenated into a single file. Returns the
// entire file as a byte slice and the number of parts missing. If the int is 0,
// then no parts are missing and the returned file is complete.
func (ps *partStore) getFile() ([]byte, int) {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	// get the length of one of the parts (all parts should be the same size)
	partLength := 0
	for _, part := range ps.parts {
		partLength = len(part)
		break
	}

	// Create new empty buffer of the size of the whole file
	buff := bytes.NewBuffer(nil)
	buff.Grow(int(ps.numParts) * partLength)

	// Loop through the map in order and add each file part that exists
	var missingParts int
	for i := uint16(0); i < ps.numParts; i++ {
		part, exists := ps.parts[i]
		if exists {
			buff.Write(part)
		} else {
			missingParts++
		}
	}

	return buff.Bytes(), missingParts
}

// len returns the number of parts stored.
func (ps *partStore) len() int {
	return len(ps.parts)
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadPartStore loads all the file parts from storage into memory.
func loadPartStore(kv *versioned.KV) (*partStore, error) {
	// get list of saved file parts
	vo, err := kv.Get(partsListKey, partsListVersion)
	if err != nil {
		return nil, errors.Errorf(loadPartListErr, err)
	}

	// Unmarshal saved data into a list
	numParts, list := unmarshalPartList(vo.Data)

	// Initialize part map
	ps := &partStore{
		parts:    make(map[uint16][]byte, numParts),
		numParts: numParts,
		kv:       kv,
	}

	// Load each part from storage and add to the map
	for _, partNum := range list {
		vo, err = kv.Get(makePartsKey(partNum), partsStoreVersion)
		if err != nil {
			return nil, errors.Errorf(loadPartsErr, partNum, err)
		}

		ps.parts[partNum] = vo.Data
	}

	return ps, nil
}

// save stores a list of all file parts and the individual parts in storage.
func (ps *partStore) save() error {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	// Save the individual file parts to storage
	for partNum := range ps.parts {
		err := ps.savePart(partNum)
		if err != nil {
			return errors.Errorf(savePartsErr, partNum, err)
		}
	}

	// Save the part list to storage
	return ps.saveList()
}

// saveList stores the list of all file parts in storage.
func (ps *partStore) saveList() error {
	obj := &versioned.Object{
		Version:   partsStoreVersion,
		Timestamp: netTime.Now(),
		Data:      ps.marshalList(),
	}

	return ps.kv.Set(partsListKey, partsListVersion, obj)
}

// savePart stores an individual file part to storage.
func (ps *partStore) savePart(partNum uint16) error {
	obj := &versioned.Object{
		Version:   partsStoreVersion,
		Timestamp: netTime.Now(),
		Data:      ps.parts[partNum],
	}

	return ps.kv.Set(makePartsKey(partNum), partsStoreVersion, obj)
}

// delete removes all the file parts and file list from storage.
func (ps *partStore) delete() error {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	for partNum := range ps.parts {
		err := ps.kv.Delete(makePartsKey(partNum), partsStoreVersion)
		if err != nil {
			return err
		}
	}

	return ps.kv.Delete(partsListKey, partsListVersion)
}

// marshalList creates a list of part numbers that are currently stored and
// returns them as a byte list to be saved to storage.
func (ps *partStore) marshalList() []byte {
	// Create new buffer of the correct size
	// (numParts (2 bytes) + (2*length of parts))
	buff := bytes.NewBuffer(nil)
	buff.Grow(2 + (2 * len(ps.parts)))

	// Write numParts to buffer
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, ps.numParts)
	buff.Write(b)

	for partNum := range ps.parts {
		b = make([]byte, 2)
		binary.LittleEndian.PutUint16(b, partNum)
		buff.Write(b)
	}

	return buff.Bytes()
}

// unmarshalPartList unmarshalls a byte slice into a list of part numbers.
func unmarshalPartList(b []byte) (uint16, []uint16) {
	buff := bytes.NewBuffer(b)

	// Read numParts from the buffer
	numParts := binary.LittleEndian.Uint16(buff.Next(2))

	// Initialize the list to the number of saved parts
	list := make([]uint16, 0, len(b)/2)

	// Read each uint16 from the buffer and save into the list
	for next := buff.Next(2); len(next) == 2; next = buff.Next(2) {
		part := binary.LittleEndian.Uint16(next)
		list = append(list, part)
	}

	return numParts, list
}

// partSliceToMap converts a slice of file parts, in order, to a map of file
// parts keyed on their part number.
func partSliceToMap(parts ...[]byte) map[uint16][]byte {
	// Initialise map to the correct size
	partMap := make(map[uint16][]byte, len(parts))

	// Add each file part to the map
	for partNum, part := range parts {
		partMap[uint16(partNum)] = make([]byte, len(part))
		copy(partMap[uint16(partNum)], part)
	}

	return partMap
}

// makePartsKey generates the key used to save a part to storage.
func makePartsKey(partNum uint16) string {
	return partsStoreKey + strconv.Itoa(int(partNum))
}
