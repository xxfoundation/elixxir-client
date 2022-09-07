////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"sync"
)

// Storage key and version.
const (
	multiStateVectorKey     = "multiStateVector/"
	multiStateVectorVersion = 0
)

// Minimum and maximum allowed number of states.
const (
	minNumStates = 2
	maxNumStates = 8
)

// Error messages.
const (
	keyNumMaxErr = "key number %d greater than max %d"
	stateMaxErr  = "given state %d greater than max %d"

	// NewMultiStateVector
	numStateMinErr = "number of received states %d must be greater than %d"
	numStateMaxErr = "number of received states %d greater than max %d"

	// MultiStateVector.Set
	setStateErr     = "failed to set state of key %d: %+v"
	saveSetStateErr = "failed to save MultiStateVector after setting key %d state to %d: %+v"

	// MultiStateVector.SetMany
	setManyStateErr     = "failed to set state of key %d (%d of %d): %+v"
	saveManySetStateErr = "failed to save MultiStateVector after setting keys %d state to %d: %+v"

	// MultiStateVector.set
	setGetStateErr = "could not get state of key %d: %+v"
	oldStateMaxErr = "stored state %d greater than max %d (this should not happen; maybe a storage failure)"
	stateChangeErr = "cannot change state of key %d from %d to %d"

	// LoadMultiStateVector
	loadGetMsvErr       = "failed to load MultiStateVector from storage: %+v"
	loadUnmarshalMsvErr = "failed to unmarshal MultiStateVector loaded from storage: %+v"

	// MultiStateVector.save
	saveUnmarshalMsvErr = "failed to marshal MultiStateVector for storage: %+v"

	// MultiStateVector.marshal
	buffWriteNumKeysErr         = "numKeys: %+v"
	buffWriteNumStatesErr       = "numStates: %+v"
	buffWriteVectLenErr         = "length of vector: %+v"
	buffWriteVectBlockErr       = "vector block %d/%d: %+v"
	buffWriteStateCountBlockErr = "state use counter %d: %+v"

	// MultiStateVector.unmarshal
	buffReadNumKeysErr         = "numKeys: %+v"
	buffReadNumStatesErr       = "numStates: %+v"
	buffReadVectLenErr         = "length of vector: %+v"
	buffReadVectBlockErr       = "vector block %d/%d: %+v"
	buffReadStateCountBlockErr = "state use counter %d: %+v"

	// checkStateMap
	stateMapLenErr      = "state map has %d states when %d are required"
	stateMapStateLenErr = "state %d in state map has %d state transitions when %d are required"
)

// MultiStateVector stores a list of a set number of keys and their state. It
// supports any number of states, unlike the StateVector. It is storage backed.
type MultiStateVector struct {
	numKeys         uint16 // Total number of keys
	numStates       uint8  // Number of state per key
	bitSize         uint8  // Number of state bits per key
	numKeysPerBlock uint8  // Number of keys that fit in a block
	wordSize        uint8  // Size of each word

	// Bitfield for key states
	vect []uint64

	// A list of states and which states they can move to
	stateMap [][]bool

	// Stores the number of keys per state
	stateUseCount []uint16

	key string // Unique string used to save/load object from storage
	kv  *versioned.KV
	mux sync.RWMutex
}

// NewMultiStateVector generates a new MultiStateVector with the specified
// number of keys and bits pr state per keys. The max number of states in 64.
func NewMultiStateVector(numKeys uint16, numStates uint8, stateMap [][]bool,
	key string, kv *versioned.KV) (*MultiStateVector, error) {

	// Return an error if the number of states is out of range
	if numStates < minNumStates {
		return nil, errors.Errorf(numStateMinErr, numStates, minNumStates)
	} else if numStates > maxNumStates {
		return nil, errors.Errorf(numStateMaxErr, numStates, maxNumStates)
	}

	// Return an error if the state map is of the wrong length
	err := checkStateMap(numStates, stateMap)
	if err != nil {
		return nil, err
	}

	// Calculate the number of bits needed to represent all the states
	numBits := uint8(math.Ceil(math.Log2(float64(numStates))))

	// Calculate number of 64-bit blocks needed to store bitSize for numKeys
	numBlocks := ((numKeys * uint16(numBits)) + 63) / 64

	msv := &MultiStateVector{
		numKeys:         numKeys,
		numStates:       numStates,
		bitSize:         numBits,
		numKeysPerBlock: 64 / numBits,
		wordSize:        (64 / numBits) * numBits,
		vect:            make([]uint64, numBlocks),
		stateMap:        stateMap,
		stateUseCount:   make([]uint16, numStates),
		key:             makeMultiStateVectorKey(key),
		kv:              kv,
	}

	msv.stateUseCount[0] = numKeys

	return msv, msv.save()
}

// Get returns the state of the key.
func (msv *MultiStateVector) Get(keyNum uint16) (uint8, error) {
	msv.mux.RLock()
	defer msv.mux.RUnlock()

	return msv.get(keyNum)
}

// get returns the state of the specified key. This function is not thread-safe.
func (msv *MultiStateVector) get(keyNum uint16) (uint8, error) {
	// Return an error if the key number is greater than the number of keys
	if keyNum > msv.numKeys-1 {
		return 0, errors.Errorf(keyNumMaxErr, keyNum, msv.numKeys-1)
	}

	// Calculate block and position of the keyNum
	block, pos := msv.getMultiBlockAndPos(keyNum)

	bitMask := uint64(math.MaxUint64) >> (64 - msv.bitSize)

	shift := 64 - pos - uint16(msv.bitSize)

	return uint8((msv.vect[block] >> shift) & bitMask), nil
}

// Set marks the key with the given state. Returns an error for an invalid state
// or if the state change is not allowed.
func (msv *MultiStateVector) Set(keyNum uint16, state uint8) error {
	msv.mux.Lock()
	defer msv.mux.Unlock()

	// Set state of the key
	err := msv.set(keyNum, state)
	if err != nil {
		return errors.Errorf(setStateErr, keyNum, err)
	}

	// Save changes to storage
	err = msv.save()
	if err != nil {
		return errors.Errorf(saveSetStateErr, keyNum, state, err)
	}

	return nil
}

// SetMany marks each of the keys with the given state. Returns an error for an
// invalid state or if the state change is not allowed.
func (msv *MultiStateVector) SetMany(keyNums []uint16, state uint8) error {
	msv.mux.Lock()
	defer msv.mux.Unlock()

	// Set state of each key
	for i, keyNum := range keyNums {
		err := msv.set(keyNum, state)
		if err != nil {
			return errors.Errorf(setManyStateErr, keyNum, i+1, len(keyNums), err)
		}
	}

	// Save changes to storage
	err := msv.save()
	if err != nil {
		return errors.Errorf(saveManySetStateErr, keyNums, state, err)
	}

	return nil
}

// set sets the specified key to the specified state. An error is returned if
// the given state is invalid or the state change is not allowed by the state
// map.
func (msv *MultiStateVector) set(keyNum uint16, state uint8) error {
	// Return an error if the key number is greater than the number of keys
	if keyNum > msv.numKeys-1 {
		return errors.Errorf(keyNumMaxErr, keyNum, msv.numKeys-1)
	}

	// Return an error if the state is larger than the max states
	if state > msv.numStates-1 {
		return errors.Errorf(stateMaxErr, state, msv.numStates-1)
	}

	// get the current state
	oldState, err := msv.get(keyNum)
	if err != nil {
		return errors.Errorf(setGetStateErr, keyNum, err)
	}

	// Check that the current state is within the allowed states
	if oldState > msv.numStates-1 {
		return errors.Errorf(oldStateMaxErr, oldState, msv.numStates-1)
	}

	// Check that the state change is allowed (only if state map was supplied)
	if msv.stateMap != nil && !msv.stateMap[oldState][state] {
		return errors.Errorf(stateChangeErr, keyNum, oldState, state)
	}

	// Calculate block and position of the key
	block, _ := msv.getMultiBlockAndPos(keyNum)

	// Clear the key's state
	msv.vect[block] &= ^getSelectionMask(keyNum, msv.bitSize)

	// Set the state
	msv.vect[block] |= shiftBitsOut(uint64(state), keyNum, msv.bitSize)

	// Increment/decrement state counters
	msv.stateUseCount[oldState]--
	msv.stateUseCount[state]++

	return nil
}

// GetNumKeys returns the total number of keys.
func (msv *MultiStateVector) GetNumKeys() uint16 {
	msv.mux.RLock()
	defer msv.mux.RUnlock()
	return msv.numKeys
}

// GetCount returns the number of keys with the given state.
func (msv *MultiStateVector) GetCount(state uint8) (uint16, error) {
	msv.mux.RLock()
	defer msv.mux.RUnlock()

	// Return an error if the state is larger than the max states
	if int(state) >= len(msv.stateUseCount) {
		return 0, errors.Errorf(stateMaxErr, state, msv.numStates-1)
	}

	return msv.stateUseCount[state], nil
}

// GetKeys returns a list of all keys with the specified status.
func (msv *MultiStateVector) GetKeys(state uint8) ([]uint16, error) {
	msv.mux.RLock()
	defer msv.mux.RUnlock()

	// Return an error if the state is larger than the max states
	if state > msv.numStates-1 {
		return nil, errors.Errorf(stateMaxErr, state, msv.numStates-1)
	}

	// Initialise list with capacity set to number of keys in the state
	keys := make([]uint16, 0, msv.stateUseCount[state])

	// Loop through each key and add any unused to the list
	for keyNum := uint16(0); keyNum < msv.numKeys; keyNum++ {
		keyState, err := msv.get(keyNum)
		if err != nil {
			return nil, errors.Errorf(keyNumMaxErr, keyNum, err)
		}
		if keyState == state {
			keys = append(keys, keyNum)
		}
	}

	return keys, nil
}

// DeepCopy creates a deep copy of the MultiStateVector without a storage
// backend. The deep copy can only be used for functions that do not access
// storage.
func (msv *MultiStateVector) DeepCopy() *MultiStateVector {
	msv.mux.RLock()
	defer msv.mux.RUnlock()

	// Copy all primitive values into the new MultiStateVector
	newMSV := &MultiStateVector{
		numKeys:         msv.numKeys,
		numStates:       msv.numStates,
		bitSize:         msv.bitSize,
		numKeysPerBlock: msv.numKeysPerBlock,
		wordSize:        msv.wordSize,
		vect:            make([]uint64, len(msv.vect)),
		stateMap:        make([][]bool, len(msv.stateMap)),
		stateUseCount:   make([]uint16, len(msv.stateUseCount)),
		key:             msv.key,
	}

	// Copy over all values in the vector
	copy(newMSV.vect, msv.vect)

	// Copy over all values in the state map
	if msv.stateMap == nil {
		newMSV.stateMap = nil
	} else {
		for state, stateTransitions := range msv.stateMap {
			newMSV.stateMap[state] = make([]bool, len(stateTransitions))
			copy(newMSV.stateMap[state], stateTransitions)
		}
	}

	// Copy over all values in the state use counter
	copy(newMSV.stateUseCount, msv.stateUseCount)

	return newMSV
}

// getMultiBlockAndPos calculates the block index and the position within that
// block of the key.
func (msv *MultiStateVector) getMultiBlockAndPos(keyNum uint16) (
	block, pos uint16) {
	block = keyNum / uint16(msv.numKeysPerBlock)
	pos = (keyNum % uint16(msv.numKeysPerBlock)) * uint16(msv.bitSize)

	return block, pos
}

// checkStateMap checks that the state map has the correct number of states and
// correct number of state transitions per state. Returns an error if any of the
// lengths are incorrect. Returns nil if they are all correct or if the stateMap
// is nil.
func checkStateMap(numStates uint8, stateMap [][]bool) error {
	if stateMap == nil {
		return nil
	}

	// Checks the length of the first dimension of the state map
	if len(stateMap) != int(numStates) {
		return errors.Errorf(stateMapLenErr, len(stateMap), numStates)
	}

	// Checks the length of each transition slice for each state
	for i, state := range stateMap {
		if len(state) != int(numStates) {
			return errors.Errorf(stateMapStateLenErr, i, len(state), numStates)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Selection Bit Mask                                                         //
////////////////////////////////////////////////////////////////////////////////

// getSelectionMask returns the selection bit mask for the given key number and
// state bit size. The masks for each state is found by right shifting
// bitSize * keyNum.
func getSelectionMask(keyNum uint16, bitSize uint8) uint64 {
	// get the mask at the zeroth position at the bit size; these masks look
	// like the following for bit sizes 1 through 4
	//	0b1000000000000000000000000000000000000000000000000000000000000000
	//	0b1100000000000000000000000000000000000000000000000000000000000000
	//	0b1110000000000000000000000000000000000000000000000000000000000000
	//	0b1111000000000000000000000000000000000000000000000000000000000000
	initialMask := uint64(math.MaxUint64) << (64 - bitSize)

	// Shift the mask to the keyNum location; for example, a mask of size 3
	// in position 3 would be:
	//	0b1110000000000000000000000000000000000000000000000000000000000000 ->
	//	0b0000000001110000000000000000000000000000000000000000000000000000
	shiftedMask := shiftBitsIn(initialMask, keyNum, bitSize)

	return shiftedMask
}

// shiftBitsOut shifts the most significant bits of size bitSize to the left to
// the position of keyNum.
func shiftBitsOut(bits uint64, keyNum uint16, bitSize uint8) uint64 {
	// Return original bits when the bit size is zero
	if bitSize == 0 {
		return bits
	}

	// Calculate the number of keys stored in each block; blocks are designed to
	// only the states that cleaning fit fully within the block. All reminder
	// trailing bits are unused. The below code calculates the number of actual
	// keys in a block.
	keysPerBlock := 64 / uint16(bitSize)

	// Calculate the index of the key in the local block
	keyIndex := keyNum % keysPerBlock

	// The starting bit position of the key in the block
	pos := keyIndex * uint16(bitSize)

	// Shift the bits left to account for the size of the key
	bitSizeShift := 64 - uint64(bitSize)
	leftShift := bits << bitSizeShift

	// Shift the initial mask based upon the bit position of the key
	shifted := leftShift >> pos

	return shifted
}

// shiftBitsIn shifts the least significant bits of size bitSize to the right to
// the position of keyNum.
func shiftBitsIn(bits uint64, keyNum uint16, bitSize uint8) uint64 {
	// Return original bits when the bit size is zero
	if bitSize == 0 {
		return bits
	}

	// Calculate the number of keys stored in each block; blocks are designed to
	// only the states that cleaning fit fully within the block. All reminder
	// trailing bits are unused. The below code calculates the number of actual
	// keys in a block.
	keysPerBlock := 64 / uint16(bitSize)

	// Calculate the index of the key in the local block
	keyIndex := keyNum % keysPerBlock

	// The starting bit position of the key in the block
	pos := keyIndex * uint16(bitSize)

	// Shift the initial mask based upon the bit position of the key
	shifted := bits >> pos

	return shifted
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadMultiStateVector loads a MultiStateVector with the specified key from the
// given versioned storage.
func LoadMultiStateVector(stateMap [][]bool, key string, kv *versioned.KV) (
	*MultiStateVector, error) {
	msv := &MultiStateVector{
		stateMap: stateMap,
		key:      makeMultiStateVectorKey(key),
		kv:       kv,
	}

	// Load MultiStateVector data from storage
	obj, err := kv.Get(msv.key, multiStateVectorVersion)
	if err != nil {
		return nil, errors.Errorf(loadGetMsvErr, err)
	}

	// Unmarshal data
	err = msv.unmarshal(obj.Data)
	if err != nil {
		return nil, errors.Errorf(loadUnmarshalMsvErr, err)
	}

	return msv, nil
}

// save stores the MultiStateVector in storage.
func (msv *MultiStateVector) save() error {
	// Marshal the MultiStateVector
	data, err := msv.marshal()
	if err != nil {
		return errors.Errorf(saveUnmarshalMsvErr, err)
	}

	// Create the versioned object
	obj := &versioned.Object{
		Version:   multiStateVectorVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return msv.kv.Set(msv.key, obj)
}

// Delete removes the MultiStateVector from storage.
func (msv *MultiStateVector) Delete() error {
	msv.mux.Lock()
	defer msv.mux.Unlock()
	return msv.kv.Delete(msv.key, multiStateVectorVersion)
}

// marshal serialises the MultiStateVector into a byte slice.
func (msv *MultiStateVector) marshal() ([]byte, error) {
	var buff bytes.Buffer
	buff.Grow((4 * 4) + 8 + (8 * len(msv.vect)))

	// Write numKeys to buffer
	err := binary.Write(&buff, binary.LittleEndian, msv.numKeys)
	if err != nil {
		return nil, errors.Errorf(buffWriteNumKeysErr, err)
	}

	// Write numStates to buffer
	err = binary.Write(&buff, binary.LittleEndian, msv.numStates)
	if err != nil {
		return nil, errors.Errorf(buffWriteNumStatesErr, err)
	}

	// Write length of vect to buffer
	err = binary.Write(&buff, binary.LittleEndian, uint64(len(msv.vect)))
	if err != nil {
		return nil, errors.Errorf(buffWriteVectLenErr, err)
	}

	// Write vector to buffer
	for i, block := range msv.vect {
		err = binary.Write(&buff, binary.LittleEndian, block)
		if err != nil {
			return nil, errors.Errorf(
				buffWriteVectBlockErr, i, len(msv.vect), err)
		}
	}

	// Write state use counter slice to buffer
	for state, count := range msv.stateUseCount {
		err = binary.Write(&buff, binary.LittleEndian, count)
		if err != nil {
			return nil, errors.Errorf(buffWriteStateCountBlockErr, state, err)
		}
	}

	return buff.Bytes(), nil
}

// unmarshal deserializes a byte slice into a MultiStateVector.
func (msv *MultiStateVector) unmarshal(b []byte) error {
	buff := bytes.NewReader(b)

	// Write numKeys to buffer
	err := binary.Read(buff, binary.LittleEndian, &msv.numKeys)
	if err != nil {
		return errors.Errorf(buffReadNumKeysErr, err)
	}

	// Write numStates to buffer
	err = binary.Read(buff, binary.LittleEndian, &msv.numStates)
	if err != nil {
		return errors.Errorf(buffReadNumStatesErr, err)
	}

	// Calculate the number of bits needed to represent all the states
	msv.bitSize = uint8(math.Ceil(math.Log2(float64(msv.numStates))))

	// Calculate numbers of keys per block
	msv.numKeysPerBlock = 64 / msv.bitSize

	// Calculate the word size
	msv.wordSize = msv.numKeysPerBlock * msv.bitSize

	// Write vect to buffer
	var vectLen uint64
	err = binary.Read(buff, binary.LittleEndian, &vectLen)
	if err != nil {
		return errors.Errorf(buffReadVectLenErr, err)
	}

	// Create new vector
	msv.vect = make([]uint64, vectLen)

	// Write vect to buffer
	for i := range msv.vect {
		err = binary.Read(buff, binary.LittleEndian, &msv.vect[i])
		if err != nil {
			return errors.Errorf(buffReadVectBlockErr, i, vectLen, err)
		}
	}

	// Create new state use counter
	msv.stateUseCount = make([]uint16, msv.numStates)
	for i := range msv.stateUseCount {
		err = binary.Read(buff, binary.LittleEndian, &msv.stateUseCount[i])
		if err != nil {
			return errors.Errorf(buffReadStateCountBlockErr, i, err)
		}
	}

	return nil
}

// makeMultiStateVectorKey generates the unique key used to save a
// MultiStateVector to storage.
func makeMultiStateVectorKey(key string) string {
	return multiStateVectorKey + key
}
