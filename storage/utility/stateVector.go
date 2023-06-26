////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage key and version.
const (
	stateVectorKey            = "stateVector"
	currentStateVectorVersion = 0
)

// Error messages.
const (
	saveUsedKeyErr    = "Failed to save %s after marking key %d as used: %+v"
	saveUsedKeysErr   = "Failed to save %s after marking keys %d as used: %+v"
	saveUnusedKeyErr  = "Failed to save %s after marking key %d as unused: %+v"
	saveUnusedKeysErr = "Failed to save %s after marking keys %d as unused: %+v"
	saveNextErr       = "failed to save %s after getting next available key: %+v"
	noKeysErr         = "all keys used"
	loadUnmarshalErr  = "failed to unmarshal from storage: %+v"
	testInterfaceErr  = "%s can only be used for testing."
)

// StateVector stores a list of a set number of items and their binary state.
// It is storage backed.
type StateVector struct {
	// Bitfield for key states; if a key is unused, then it is set to 0;
	// otherwise, it is used/not available and is set 1
	vect []uint64

	firstAvailable uint32 // Sequentially, the first unused key (equal to 0)
	numKeys        uint32 // Total number of keys
	numAvailable   uint32 // Number of unused keys

	disableKV bool   // Toggles use of KV storage
	key       string // Unique string used to save/load object from storage
	kv        versioned.KV
	mux       sync.RWMutex
}

// NewStateVector generates a new StateVector with the specified number of keys.
func NewStateVector(numKeys uint32, disableKV bool, key string,
	kv versioned.KV) (*StateVector, error) {

	// Calculate the number of 64-bit blocks needed to store numKeys
	numBlocks := (numKeys + 63) / 64

	sv := &StateVector{
		vect:           make([]uint64, numBlocks),
		firstAvailable: 0,
		numKeys:        numKeys,
		numAvailable:   numKeys,
		disableKV:      disableKV,
		key:            makeStateVectorKey(key),
		kv:             kv,
	}

	return sv, sv.save()
}

// Use marks the key as used (sets it to 1).
func (sv *StateVector) Use(keyNum uint32) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	// Mark the key as used
	sv.use(keyNum)

	// Save changes to storage
	if err := sv.save(); err != nil {
		jww.FATAL.Printf(saveUsedKeyErr, sv, keyNum, err)
	}
}

// UseMany marks all of the keys as used (sets them to 1). Saves only after all
// of the keys are set.
func (sv *StateVector) UseMany(keyNums ...uint32) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	// Mark the keys as used
	for _, keyNum := range keyNums {
		sv.use(keyNum)
	}

	// Save changes to storage
	if err := sv.save(); err != nil {
		jww.FATAL.Printf(saveUsedKeysErr, sv, keyNums, err)
	}
}

// use marks the key as used (sets it to 1). It is not thread-safe and does not
// save to storage.
func (sv *StateVector) use(keyNum uint32) {
	// If the key is already used, then exit
	if sv.used(keyNum) {
		return
	}

	// Calculate block and position of the key
	block, pos := getBlockAndPos(keyNum)

	// Set the key to used (1)
	sv.vect[block] |= 1 << pos

	// Decrement number available unused keys
	sv.numAvailable--

	// If this is the first available key, then advanced to the next available
	if keyNum == sv.firstAvailable {
		sv.nextAvailable()
	}
}

// Unuse marks the key as unused (sets it to 0).
func (sv *StateVector) Unuse(keyNum uint32) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	// Mark the key as used
	sv.unuse(keyNum)

	// Save changes to storage
	if err := sv.save(); err != nil {
		jww.FATAL.Printf(saveUnusedKeyErr, sv, keyNum, err)
	}
}

// UnuseMany marks all the key as unused (sets them to 0). Saves only after all
// of the keys are set.
func (sv *StateVector) UnuseMany(keyNums ...uint32) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	// Mark all of the keys as unused
	for _, keyNum := range keyNums {
		sv.unuse(keyNum)
	}

	// Save changes to storage
	if err := sv.save(); err != nil {
		jww.FATAL.Printf(saveUnusedKeysErr, sv, keyNums, err)
	}
}

// unuse marks the key as unused (sets it to 0). It is not thread-safe and does
// not save to storage.
func (sv *StateVector) unuse(keyNum uint32) {
	// If the key is already unused, then exit
	if !sv.used(keyNum) {
		return
	}

	// Calculate block and position of the key
	block, pos := getBlockAndPos(keyNum)

	// Set the key to unused (0)
	sv.vect[block] &= ^(1 << pos)

	// Increment number available unused keys
	sv.numAvailable++

	// If this is before the first available key, then set to the next available
	if keyNum < sv.firstAvailable {
		sv.firstAvailable = keyNum
	}
}

// Used returns true if the key is used (set to 1) or false if the key is unused
// (set to 0).
func (sv *StateVector) Used(keyNum uint32) bool {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	return sv.used(keyNum)
}

// used determines if the key is used or unused. This function is not thread
// safe.
func (sv *StateVector) used(keyNum uint32) bool {
	// Calculate block and position of the keyNum
	block, pos := getBlockAndPos(keyNum)

	return (sv.vect[block]>>pos)&1 == 1
}

// Next marks the first unused key as used. An error is returned if all keys are
// used or if the save to storage fails.
func (sv *StateVector) Next() (uint32, error) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	// Return an error if all keys are used
	if sv.firstAvailable >= sv.numKeys {
		return sv.numKeys, errors.New(noKeysErr)
	}

	// Mark the first available as used (which also advanced firstAvailable)
	nextKey := sv.firstAvailable
	sv.use(nextKey)

	// Save to storage
	if err := sv.save(); err != nil {
		jww.FATAL.Panicf(saveNextErr, sv, err)
	}

	return nextKey, nil
}

// nextAvailable finds the next unused key and sets it as the firstAvailable. It
// is not thread-safe and does not save to storage.
func (sv *StateVector) nextAvailable() {
	// Add one to start at the next position
	pos := sv.firstAvailable + 1
	block := pos / 64

	// Loop through each key until the first unused key is found
	for block < uint32(len(sv.vect)) && (sv.vect[block]>>(pos%64))&1 == 1 {
		pos++
		block = pos / 64
	}

	sv.firstAvailable = pos
}

// GetNumAvailable returns the number of unused keys.
func (sv *StateVector) GetNumAvailable() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numAvailable
}

// GetNumUsed returns the number of used keys.
func (sv *StateVector) GetNumUsed() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numKeys - sv.numAvailable
}

// GetNumKeys returns the total number of keys.
func (sv *StateVector) GetNumKeys() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numKeys
}

// GetUnusedKeyNums returns a list of all unused keys.
func (sv *StateVector) GetUnusedKeyNums() []uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	// Initialise list with capacity set to number of unused keys
	keyNums := make([]uint32, 0, sv.numAvailable)

	// Loop through each key and add any unused to the list
	for keyNum := sv.firstAvailable; keyNum < sv.numKeys; keyNum++ {
		if !sv.used(keyNum) {
			keyNums = append(keyNums, keyNum)
		}
	}

	return keyNums
}

// GetUsedKeyNums returns a list of all used keys.
func (sv *StateVector) GetUsedKeyNums() []uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	// Initialise list with capacity set to the number of used keys
	keyNums := make([]uint32, 0, sv.numKeys-sv.numAvailable)

	// Loop through each key and add any used key numbers to the list
	for keyNum := uint32(0); keyNum < sv.numKeys; keyNum++ {
		if sv.used(keyNum) {
			keyNums = append(keyNums, keyNum)
		}
	}

	return keyNums
}

// DeepCopy creates a deep copy of the StateVector without a storage backend.
// The deep copy can only be used for functions that do not access storage.
func (sv *StateVector) DeepCopy() *StateVector {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	newSV := &StateVector{
		vect:           make([]uint64, len(sv.vect)),
		firstAvailable: sv.firstAvailable,
		numKeys:        sv.numKeys,
		numAvailable:   sv.numAvailable,
		disableKV:      true,
		key:            sv.key,
	}

	for i, val := range sv.vect {
		newSV.vect[i] = val
	}

	return newSV
}

// getBlockAndPos calculates the block index and the position within that block
// of a key number.
func getBlockAndPos(keyNum uint32) (block, pos uint32) {
	block = keyNum / 64
	pos = keyNum % 64

	return block, pos
}

// String returns a human-readable representing of the StateVector for logging
// and debugging. This functions adheres to the fmt.Stringer interface.
func (sv *StateVector) String() string {
	fields := []string{
		"vect:" + fmt.Sprintf("%d", sv.vect),
		"firstAvailable:" + strconv.Itoa(int(sv.firstAvailable)),
		"numKeys:" + strconv.Itoa(int(sv.numKeys)),
		"numAvailable:" + strconv.Itoa(int(sv.numAvailable)),
		"disableKV:" + strconv.FormatBool(sv.disableKV),
		"key:" + sv.key,
		"kv:" + fmt.Sprintf("%p", sv.kv),
	}
	return "{" + strings.Join(fields, " ") + "}"
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// stateVectorDisk is used to save the data from a StateVector so that it can be
// JSON marshalled.
type stateVectorDisk struct {
	Vect           []uint64 `json:"vect"`
	FirstAvailable uint32   `json:"firstAvailable"`
	NumKeys        uint32   `json:"numKeys"`
	NumAvailable   uint32   `json:"numAvailable"`
}

// LoadStateVector loads a StateVector with the specified key from the given
// versioned storage.
func LoadStateVector(kv versioned.KV, key string) (*StateVector, error) {
	sv := &StateVector{
		disableKV: false,
		key:       makeStateVectorKey(key),
		kv:        kv,
	}

	// Load StateVector data from storage
	obj, err := kv.Get(sv.key, currentStateVectorVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal data
	err = sv.UnmarshalJSON(obj.Data)
	if err != nil {
		return nil, errors.Errorf(loadUnmarshalErr, err)
	}

	return sv, nil
}

// save stores the StateVector in storage.
func (sv *StateVector) save() error {
	if sv.disableKV {
		return nil
	}

	// Marshal the StateVector
	data, err := sv.MarshalJSON()
	if err != nil {
		return err
	}

	// Create the versioned object
	obj := versioned.Object{
		Version:   currentStateVectorVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return sv.kv.Set(sv.key, &obj)
}

// Delete remove the StateVector from storage.
func (sv *StateVector) Delete() error {
	if sv.disableKV {
		return nil
	}
	return sv.kv.Delete(sv.key, currentStateVectorVersion)
}

// MarshalJSON marshals the StateVector into valid JSON. This function adheres
// to the json.Marshaler interface.
func (sv *StateVector) MarshalJSON() ([]byte, error) {
	svd := stateVectorDisk{
		Vect:           sv.vect,
		FirstAvailable: sv.firstAvailable,
		NumKeys:        sv.numKeys,
		NumAvailable:   sv.numAvailable,
	}

	return json.Marshal(&svd)
}

// UnmarshalJSON unmarshalls the JSON into the StateVector. This function
// adheres to the json.Unmarshaler interface.
func (sv *StateVector) UnmarshalJSON(data []byte) error {
	var svd stateVectorDisk
	err := json.Unmarshal(data, &svd)
	if err != nil {
		return err
	}

	sv.vect = svd.Vect
	sv.firstAvailable = svd.FirstAvailable
	sv.numKeys = svd.NumKeys
	sv.numAvailable = svd.NumAvailable

	return nil
}

// makeStateVectorKey generates the unique key used to save a StateVector to
// storage.
func makeStateVectorKey(key string) string {
	return stateVectorKey + key
}

////////////////////////////////////////////////////////////////////////////////
// Testing Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// SaveTEST saves the StateVector to storage. This should only be used for
// testing.
func (sv *StateVector) SaveTEST(x interface{}) error {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(testInterfaceErr, "SaveTEST")
	}

	sv.mux.Lock()
	defer sv.mux.Unlock()

	return sv.save()
}

// SetFirstAvailableTEST sets the firstAvailable. This should only be used for
// testing.
func (sv *StateVector) SetFirstAvailableTEST(keyNum uint32, x interface{}) {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(testInterfaceErr, "SetFirstAvailableTEST")
	}

	sv.mux.Lock()
	defer sv.mux.Unlock()

	sv.firstAvailable = keyNum
}

// SetNumKeysTEST sets the numKeys. This should only be used for testing.
func (sv *StateVector) SetNumKeysTEST(numKeys uint32, x interface{}) {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(testInterfaceErr, "SetNumKeysTEST")
	}

	sv.mux.Lock()
	defer sv.mux.Unlock()

	sv.numKeys = numKeys
}

// SetNumAvailableTEST sets the numAvailable. This should only be used for
// testing.
func (sv *StateVector) SetNumAvailableTEST(numAvailable uint32, x interface{}) {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(testInterfaceErr, "SetNumAvailableTEST")
	}

	sv.mux.Lock()
	defer sv.mux.Unlock()

	sv.numAvailable = numAvailable
}

// SetKvTEST sets the kv. This should only be used for testing.
func (sv *StateVector) SetKvTEST(kv versioned.KV, x interface{}) {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(testInterfaceErr, "SetKvTEST")
	}

	sv.mux.Lock()
	defer sv.mux.Unlock()

	sv.kv = kv
}
