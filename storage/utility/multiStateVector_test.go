////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Tests that NewMultiStateVector returns a new MultiStateVector with the
// expected values.
func TestNewMultiStateVector(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := "testKey"
	expected := &MultiStateVector{
		numKeys:         189,
		numStates:       5,
		bitSize:         3,
		numKeysPerBlock: 21,
		wordSize:        63,
		vect:            make([]uint64, 9),
		stateMap: [][]bool{
			{true, true, true, true, true},
			{true, false, true, true, true},
			{false, false, false, true, true},
			{false, false, false, false, true},
			{false, false, false, false, false},
		},
		stateUseCount: []uint16{189, 0, 0, 0, 0},
		key:           makeMultiStateVectorKey(key),
		kv:            kv,
	}

	newMSV, err := NewMultiStateVector(expected.numKeys, expected.numStates, expected.stateMap, key, kv)
	if err != nil {
		t.Errorf("NewMultiStateVector returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, newMSV) {
		t.Errorf("New MultiStateVector does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newMSV)
	}
}

// Error path: tests that NewMultiStateVector returns the expected error when
// the number of states is too small.
func TestNewMultiStateVector_NumStatesMinError(t *testing.T) {
	expectedErr := fmt.Sprintf(numStateMinErr, minNumStates-1, minNumStates)
	_, err := NewMultiStateVector(5, minNumStates-1, nil, "", nil)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("NewMultiStateVector did not return the expected error when "+
			"the number of states is too small.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: tests that NewMultiStateVector returns the expected error when
// the number of states is too large.
func TestNewMultiStateVector_NumStatesMaxError(t *testing.T) {
	expectedErr := fmt.Sprintf(numStateMaxErr, maxNumStates+1, maxNumStates)
	_, err := NewMultiStateVector(5, maxNumStates+1, nil, "", nil)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("NewMultiStateVector did not return the expected error when "+
			"the number of states is too large.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: tests that NewMultiStateVector returns the expected error when
// the state map is of the wrong size.
func TestNewMultiStateVector_StateMapError(t *testing.T) {
	expectedErr := fmt.Sprintf(stateMapLenErr, 1, 5)
	_, err := NewMultiStateVector(5, 5, [][]bool{{true}}, "", nil)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("NewMultiStateVector did not return the expected error when "+
			"the state map is too small.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.Get returns the expected states for all keys in
// two situations. First, it tests that all states are zero on creation. Second,
// random states are generated and manually inserted into the vector and then
// each key is checked for the expected vector
func TestMultiStateVector_Get(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	// Check that all states are zero
	for keyNum := uint16(0); keyNum < msv.numKeys; keyNum++ {
		state, err := msv.Get(keyNum)
		if err != nil {
			t.Errorf("get returned an error for key %d: %+v", keyNum, err)
		}
		if state != 0 {
			t.Errorf("Key %d has unexpected state.\nexpected: %d\nreceived: %d",
				keyNum, 0, state)
		}
	}

	// Generate slice of expected states and set vector to expected values. Each
	// state is generated randomly. A binary string is built for each block
	// using a lookup table (note the LUT can only be used for 3-bit states).
	// When the string is 64 characters long, it is converted from a binary
	// string to an uint64 and inserted into the vector.
	expectedStates := make([]uint8, msv.numKeys)
	prng := rand.New(rand.NewSource(42))
	stateLUT := map[uint8]string{0: "000", 1: "001", 2: "010", 3: "011",
		4: "100", 5: "101", 6: "110", 7: "111"}
	block, blockString := 0, ""
	keysInVect := uint16(int(msv.numKeysPerBlock) * len(msv.vect))
	for keyNum := uint16(0); keyNum < keysInVect; keyNum++ {
		if keyNum < msv.numKeys {
			state := uint8(prng.Intn(int(msv.numStates)))
			blockString += stateLUT[state]
			expectedStates[keyNum] = state
		} else {
			blockString += stateLUT[0]
		}

		if (keyNum+1)%uint16(msv.numKeysPerBlock) == 0 {
			val, err := strconv.ParseUint(blockString+"0", 2, 64)
			if err != nil {
				t.Errorf("Failed to parse vector string %d: %+v", block, err)
			}
			msv.vect[block] = val
			block++
			blockString = ""
		}
	}

	// Check that each key has the expected state
	for keyNum, expectedState := range expectedStates {
		state, err := msv.Get(uint16(keyNum))
		if err != nil {
			t.Errorf("get returned an error for key %d: %+v", keyNum, err)
		}

		if expectedState != state {
			t.Errorf("Key %d has unexpected state.\nexpected: %d\nreceived: %d",
				keyNum, expectedState, state)
		}
	}
}

// Error path: tests that MultiStateVector.get returns the expected error when
// the key number is greater than the max key number.
func TestMultiStateVector_get_KeyNumMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	keyNum := msv.numKeys
	expectedErr := fmt.Sprintf(keyNumMaxErr, keyNum, msv.numKeys-1)
	_, err = msv.get(keyNum)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("get did not return the expected error when the key number "+
			"is larger than the max key number.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.Set sets the correct key to the correct status
// and that the state use counter is set correctly.
func TestMultiStateVector_Set(t *testing.T) {
	stateMap := [][]bool{
		{false, true, false, false, false},
		{true, false, true, false, false},
		{false, true, false, true, false},
		{false, false, true, false, true},
		{false, false, false, false, false},
	}
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, stateMap, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	testValues := []struct {
		keyNum             uint16
		state              uint8
		newCount, oldCount uint16
	}{
		{22, 1, 1, msv.numKeys - 1},
		{22, 2, 1, 0},
		{2, 1, 1, msv.numKeys - 2},
		{2, 2, 2, 0},
		{22, 3, 1, 1},
		{16, 1, 1, msv.numKeys - 3},
		{22, 4, 1, 0},
		{16, 2, 2, 0},
		{2, 3, 1, 1},
		{16, 1, 1, 0},
		{2, 4, 2, 0},
		{16, 2, 1, 0},
		{16, 3, 1, 0},
		{16, 4, 3, 0},
	}

	for i, v := range testValues {
		oldStatus, _ := msv.Get(v.keyNum)

		if err = msv.Set(v.keyNum, v.state); err != nil {
			t.Errorf("Failed to move key %d to state %d (%d): %+v",
				v.keyNum, v.state, i, err)
		} else if received, _ := msv.Get(v.keyNum); received != v.state {
			t.Errorf("Key %d has state %d instead of %d (%d).",
				v.keyNum, received, v.state, i)
		}

		if msv.stateUseCount[v.state] != v.newCount {
			t.Errorf("Count for new state %d is %d instead of %d (%d).",
				v.state, msv.stateUseCount[v.state], v.newCount, i)
		}

		if msv.stateUseCount[oldStatus] != v.oldCount {
			t.Errorf("Count for old state %d is %d instead of %d (%d).",
				oldStatus, msv.stateUseCount[oldStatus], v.oldCount, i)
		}
	}
}

// Error path: tests that MultiStateVector.Set returns the expected error when
// the key number is greater than the last key number.
func TestMultiStateVector_Set_KeyNumMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	keyNum := msv.numKeys
	expectedErr := fmt.Sprintf(
		setStateErr, keyNum, fmt.Sprintf(keyNumMaxErr, keyNum, msv.numKeys-1))
	err = msv.Set(keyNum, 1)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Set did not return the expected error when the key number "+
			"is larger than the max key number.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.SetMany
func TestMultiStateVector_SetMany(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	testValues := []struct {
		keyNums  []uint16
		state    uint8
		newCount uint16
	}{
		{[]uint16{2, 22, 12, 19}, 1, 4},
		{[]uint16{1, 5, 154, 7, 6}, 4, 5},
	}

	for i, v := range testValues {

		if err = msv.SetMany(v.keyNums, v.state); err != nil {
			t.Errorf("Failed to move keys %d to state %d (%d): %+v",
				v.keyNums, v.state, i, err)
		} else {
			for _, keyNum := range v.keyNums {
				if received, _ := msv.Get(keyNum); received != v.state {
					t.Errorf("Key %d has state %d instead of %d (%d).",
						keyNum, received, v.state, i)
				}
			}
		}

		if msv.stateUseCount[v.state] != v.newCount {
			t.Errorf("Count for new state %d is %d instead of %d (%d).",
				v.state, msv.stateUseCount[v.state], v.newCount, i)
		}
	}
}

// Error path: tests that MultiStateVector.SetMany returns the expected error
// when one of the keys is greater than the last key number.
func TestMultiStateVector_SetMany_KeyNumMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	keyNum := msv.numKeys
	expectedErr := fmt.Sprintf(setManyStateErr, keyNum, 1, 1,
		fmt.Sprintf(keyNumMaxErr, keyNum, msv.numKeys-1))
	err = msv.SetMany([]uint16{keyNum}, 1)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("SetMany did not return the expected error when the key "+
			"number is larger than the max key number."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}

// Error path: tests that MultiStateVector.set returns the expected error when
// the key number is greater than the last key number.
func TestMultiStateVector_set_KeyNumMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	keyNum := msv.numKeys
	expectedErr := fmt.Sprintf(keyNumMaxErr, keyNum, msv.numKeys-1)
	err = msv.set(keyNum, 1)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("set did not return the expected error when the key number "+
			"is larger than the max key number.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: tests that MultiStateVector.set returns the expected error when
// the given state is greater than the last state.
func TestMultiStateVector_set_NewStateMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	state := msv.numStates
	expectedErr := fmt.Sprintf(stateMaxErr, state, msv.numStates-1)
	err = msv.set(0, state)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("set did not return the expected error when the state is "+
			"larger than the max number of states.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: tests that MultiStateVector.set returns the expected error when
// the state read from the vector is greater than the last state.
func TestMultiStateVector_set_OldStateMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	msv.vect[0] = 0b1110000000000000000000000000000000000000000000000000000000000000

	expectedErr := fmt.Sprintf(oldStateMaxErr, 7, msv.numStates-1)

	err = msv.set(0, 1)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("set did not return the expected error when the state is "+
			"larger than the max number of states.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: tests that MultiStateVector.set returns the expected error when
// the state change is not allowed by the state map.
func TestMultiStateVector_set_StateChangeError(t *testing.T) {
	stateMap := [][]bool{{true, false}, {true, true}}
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 2, stateMap, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	keyNum, state := uint16(0), uint8(1)
	expectedErr := fmt.Sprintf(stateChangeErr, keyNum, 0, state)

	err = msv.set(keyNum, state)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("set did not return the expected error when the state change "+
			"should not be allowed.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.GetNumKeys returns the expected number of keys.
func TestMultiStateVector_GetNumKeys(t *testing.T) {
	numKeys := uint16(155)
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(numKeys, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	if numKeys != msv.GetNumKeys() {
		t.Errorf("Got unexpected number of keys.\nexpected: %d\nreceived: %d",
			numKeys, msv.GetNumKeys())
	}
}

// Tests that MultiStateVector.GetCount returns the correct count for each state
// after each key has been set to a random state.
func TestMultiStateVector_GetCount(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(
		156, 5, nil, "TestMultiStateVector_GetCount", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	expectedCounts := make([]uint16, msv.numStates)

	prng := rand.New(rand.NewSource(42))
	for keyNum := uint16(0); keyNum < msv.numKeys; keyNum++ {
		state := uint8(prng.Intn(int(msv.numStates)))

		err = msv.Set(keyNum, state)
		if err != nil {
			t.Errorf("Failed to set key %d to state %d.", keyNum, state)
		}

		expectedCounts[state]++
	}

	for state, expected := range expectedCounts {
		count, err := msv.GetCount(uint8(state))
		if err != nil {
			t.Errorf("GetCount returned an error for state %d: %+v", state, err)
		}
		if expected != count {
			t.Errorf("Incorrect count for state %d.\nexpected: %d\nreceived: %d",
				state, expected, count)
		}
	}
}

// Error path: tests that MultiStateVector.GetCount returns the expected error
// when the given state is greater than the last state.
func TestMultiStateVector_GetCount_NewStateMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	state := msv.numStates
	expectedErr := fmt.Sprintf(stateMaxErr, state, msv.numStates-1)
	_, err = msv.GetCount(state)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("GetCount did not return the expected error when the state is "+
			"larger than the max number of states.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.GetKeys returns the correct list of keys for each
// state after each key has been set to a random state.
func TestMultiStateVector_GetKeys(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(
		156, 5, nil, "TestMultiStateVector_GetKeys", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	expectedKeys := make([][]uint16, msv.numStates)

	prng := rand.New(rand.NewSource(42))
	for keyNum := uint16(0); keyNum < msv.numKeys; keyNum++ {
		state := uint8(prng.Intn(int(msv.numStates)))

		err = msv.Set(keyNum, state)
		if err != nil {
			t.Errorf("Failed to set key %d to state %d.", keyNum, state)
		}

		expectedKeys[state] = append(expectedKeys[state], keyNum)
	}

	for state, expected := range expectedKeys {
		keys, err := msv.GetKeys(uint8(state))
		if err != nil {
			t.Errorf("GetNodeKeys returned an error: %+v", err)
		}
		if !reflect.DeepEqual(expected, keys) {
			t.Errorf("Incorrect keys for state %d.\nexpected: %d\nreceived: %d",
				state, expected, keys)
		}
	}
}

// Error path: tests that MultiStateVector.GetKeys returns the expected error
// when the given state is greater than the last state.
func TestMultiStateVector_GetKeys_NewStateMaxError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	state := msv.numStates
	expectedErr := fmt.Sprintf(stateMaxErr, state, msv.numStates-1)
	_, err = msv.GetKeys(state)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("GetNodeKeys did not return the expected error when the state is "+
			"larger than the max number of states.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that MultiStateVector.DeepCopy makes a copy of the values and not of
// the pointers.
func TestMultiStateVector_DeepCopy(t *testing.T) {
	stateMap := [][]bool{
		{false, true, false, false, false},
		{true, false, true, false, false},
		{false, true, false, true, false},
		{false, false, true, false, true},
		{false, false, false, false, false},
	}
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, stateMap, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	newMSV := msv.DeepCopy()
	msv.kv = nil

	// Check that the values are the same
	if !reflect.DeepEqual(msv, newMSV) {
		t.Errorf("Original and copy do not match."+
			"\nexpected: %+v\nreceived: %+v", msv, newMSV)
	}

	// Check that the pointers are different
	if msv == newMSV {
		t.Errorf("Pointers for original and copy match."+
			"\nexpected: %p\nreceived: %p", msv, newMSV)
	}

	if &msv.vect == &newMSV.vect {
		t.Errorf("Pointers for original and copy vect match."+
			"\nexpected: %p\nreceived: %p", msv.vect, newMSV.vect)
	}

	if &msv.vect[0] == &newMSV.vect[0] {
		t.Errorf("Pointers for original and copy vect[0] match."+
			"\nexpected: %p\nreceived: %p", &msv.vect[0], &newMSV.vect[0])
	}

	if &msv.stateMap == &newMSV.stateMap {
		t.Errorf("Pointers for original and copy stateMap match."+
			"\nexpected: %p\nreceived: %p", msv.stateMap, newMSV.stateMap)
	}

	if &msv.stateMap[0] == &newMSV.stateMap[0] {
		t.Errorf("Pointers for original and copy stateMap[0] match."+
			"\nexpected: %p\nreceived: %p",
			&msv.stateMap[0], &newMSV.stateMap[0])
	}

	if &msv.stateMap[0][0] == &newMSV.stateMap[0][0] {
		t.Errorf("Pointers for original and copy stateMap[0][0] match."+
			"\nexpected: %p\nreceived: %p",
			&msv.stateMap[0][0], &newMSV.stateMap[0][0])
	}

	if &msv.stateUseCount == &newMSV.stateUseCount {
		t.Errorf("Pointers for original and copy stateUseCount match."+
			"\nexpected: %p\nreceived: %p",
			msv.stateUseCount, newMSV.stateUseCount)
	}

	if &msv.stateUseCount[0] == &newMSV.stateUseCount[0] {
		t.Errorf("Pointers for original and copy stateUseCount[0] match."+
			"\nexpected: %p\nreceived: %p",
			&msv.stateUseCount[0], &newMSV.stateUseCount[0])
	}
}

// Tests that MultiStateVector.DeepCopy is able to make the expected copy when
// the state map is nil.
func TestMultiStateVector_DeepCopy_NilStateMap(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	msv, err := NewMultiStateVector(155, 5, nil, "", kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	newMSV := msv.DeepCopy()
	msv.kv = nil

	// Check that the values are the same
	if !reflect.DeepEqual(msv, newMSV) {
		t.Errorf("Original and copy do not match."+
			"\nexpected: %+v\nreceived: %+v", msv, newMSV)
	}
}

// Tests that getMultiBlockAndPos returns the expected block and position for
// various key numbers, bit sizes, and word sizes.
func Test_getMultiBlockAndPos(t *testing.T) {
	testValues := []struct {
		keyNum, block, pos uint16
		msv                *MultiStateVector
	}{
		{0, 0, 0, getTestMSV(150, 2)},
		{1, 0, 1, getTestMSV(150, 2)},
		{24, 0, 24, getTestMSV(150, 2)},
		{64, 1, 0, getTestMSV(150, 2)},
		{0, 0, 0, getTestMSV(150, 5)},
		{1, 0, 3, getTestMSV(150, 5)},
		{9, 0, 27, getTestMSV(150, 5)},
		{21, 1, 0, getTestMSV(150, 5)},
		{0, 0, 0, getTestMSV(150, 255)},
		{1, 0, 8, getTestMSV(150, 255)},
		{2, 0, 16, getTestMSV(150, 255)},
		{8, 1, 0, getTestMSV(150, 255)},
		{0, 0, 0, getTestMSV(150, 127)},
		{8, 0, 56, getTestMSV(150, 127)},
		{9, 1, 0, getTestMSV(150, 127)},
	}

	for i, v := range testValues {
		block, pos := v.msv.getMultiBlockAndPos(v.keyNum)
		if v.block != block || v.pos != pos {
			t.Errorf("Incorrect block/pos for key %2d (%d bits, %2d word size): "+
				"expected %d/%-2d, got %d/%-2d (%d).", v.keyNum, v.msv.bitSize,
				v.msv.wordSize, v.block, v.pos, block, pos, i)
		}
	}
}

func getTestMSV(numKeys uint16, numStates uint8) *MultiStateVector {
	numBits := uint8(math.Ceil(math.Log2(float64(numStates))))
	return &MultiStateVector{
		numKeys:         numKeys,
		numStates:       numStates,
		bitSize:         numBits,
		numKeysPerBlock: 64 / numBits,
		wordSize:        (64 / numBits) * numBits,
	}
}

// Tests that checkStateMap does not return an error for various state map
// sizes.
func Test_checkStateMap(tt *testing.T) {
	const (
		t = true
		f = false
	)
	testValues := []struct {
		numStates uint8
		stateMap  [][]bool
	}{
		{9, nil},
		{0, [][]bool{}},
		{1, [][]bool{{t}}},
		{2, [][]bool{{t, t}, {f, f}}},
		{3, [][]bool{{t, t, f}, {f, f, t}, {t, t, t}}},
		{4, [][]bool{{t, t, f, f}, {f, f, t, f}, {t, t, t, f}, {f, f, t, t}}},
	}

	for i, v := range testValues {
		err := checkStateMap(v.numStates, v.stateMap)
		if err != nil {
			tt.Errorf("Could not verify state map #%d with %d states: %+v",
				i, v.numStates, err)
		}
	}
}

// Error path: tests that checkStateMap returns an error for various state map
// sizes and number of states mismatches.
func Test_checkStateMap_Error(tt *testing.T) {
	const (
		t = true
		f = false
	)
	testValues := []struct {
		numStates uint8
		stateMap  [][]bool
		err       string
	}{
		{1, [][]bool{},
			fmt.Sprintf(stateMapLenErr, 0, 1)},
		{0, [][]bool{{t}},
			fmt.Sprintf(stateMapLenErr, 1, 0)},
		{2, [][]bool{{t, t}, {f, f}, {f, f}},
			fmt.Sprintf(stateMapLenErr, 3, 2)},
		{3, [][]bool{{t, t, f}, {f, f, f, t}, {t, t, t}},
			fmt.Sprintf(stateMapStateLenErr, 1, 4, 3)},
		{4, [][]bool{{t, t, f, f}, {f, f, t, f}, {t, t, t, f}, {f, t, t}},
			fmt.Sprintf(stateMapStateLenErr, 3, 3, 4)},
	}

	for i, v := range testValues {
		err := checkStateMap(v.numStates, v.stateMap)
		if err == nil || !strings.Contains(err.Error(), v.err) {
			tt.Errorf("Verified invalid state map #%d with %d states."+
				"\nexpected: %s\nreceived: %v",
				i, v.numStates, v.err, err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Selection Bit Mask                                                         //
////////////////////////////////////////////////////////////////////////////////

// Tests that getSelectionMask returns the expected bit mask for various key
// numbers and bit sizes.
func Test_getSelectionMask(t *testing.T) {
	testValues := []struct {
		keyNum  uint16
		bitSize uint8
		mask    uint64
	}{
		{0, 0, 0b0000000000000000000000000000000000000000000000000000000000000000},
		{0, 1, 0b1000000000000000000000000000000000000000000000000000000000000000},
		{1, 0, 0b0000000000000000000000000000000000000000000000000000000000000000},
		{1, 1, 0b0100000000000000000000000000000000000000000000000000000000000000},
		{63, 1, 0b0000000000000000000000000000000000000000000000000000000000000001},
		{64, 1, 0b1000000000000000000000000000000000000000000000000000000000000000},
		{0, 3, 0b1110000000000000000000000000000000000000000000000000000000000000},
		{1, 3, 0b0001110000000000000000000000000000000000000000000000000000000000},
		{20, 3, 0b0000000000000000000000000000000000000000000000000000000000001110},
		{21, 3, 0b1110000000000000000000000000000000000000000000000000000000000000},
		{0, 17, 0b1111111111111111100000000000000000000000000000000000000000000000},
		{1, 17, 0b0000000000000000011111111111111111000000000000000000000000000000},
		{2, 17, 0b0000000000000000000000000000000000111111111111111110000000000000},
		{3, 17, 0b1111111111111111100000000000000000000000000000000000000000000000},
		{0, 33, 0b1111111111111111111111111111111110000000000000000000000000000000},
		{1, 33, 0b1111111111111111111111111111111110000000000000000000000000000000},
		{2, 33, 0b1111111111111111111111111111111110000000000000000000000000000000},
		{3, 33, 0b1111111111111111111111111111111110000000000000000000000000000000},
		{0, 64, 0b1111111111111111111111111111111111111111111111111111111111111111},
		{1, 64, 0b1111111111111111111111111111111111111111111111111111111111111111},
		{2, 64, 0b1111111111111111111111111111111111111111111111111111111111111111},
		{3, 64, 0b1111111111111111111111111111111111111111111111111111111111111111},
	}

	for i, v := range testValues {
		mask := getSelectionMask(v.keyNum, v.bitSize)

		if v.mask != mask {
			t.Errorf("Unexpected bit mask for key %d of size %d (%d)."+
				"\nexpected: %064b\nreceived: %064b",
				v.keyNum, v.bitSize, i, v.mask, mask)
		}
	}
}

// Tests that shiftBitsOut returns the expected shifted bits for various key
// numbers and bit sizes.
func Test_shiftBitsOut(t *testing.T) {
	testValues := []struct {
		keyNum         uint16
		bitSize        uint8
		bits, expected uint64
	}{
		{0, 0,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000001},
		{0, 1,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000000},
		{1, 0,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000001},
		{1, 2,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b0001000000000000000000000000000000000000000000000000000000000000},
		{1, 32,
			0b0111111111111111111111111111111111111111111111111111111111111110,
			0b0000000000000000000000000000000011111111111111111111111111111110},
		{13, 7,
			0b0111111111111111111111111111111111111111111111111111111111111110,
			0b0000000000000000000000000000111111000000000000000000000000000000},
	}

	for i, v := range testValues {
		bits := shiftBitsOut(v.bits, v.keyNum, v.bitSize)

		if v.expected != bits {
			t.Errorf("Unexpected shifted mask for key %d of size %d (%d)."+
				"\nexpected: %064b\nreceived: %064b",
				v.keyNum, v.bitSize, i, v.expected, bits)
		}
	}
}

// Tests that shiftBitsIn returns the expected shifted bits for various key
// numbers and bit sizes.
func Test_shiftBitsIn(t *testing.T) {
	testValues := []struct {
		keyNum         uint16
		bitSize        uint8
		bits, expected uint64
	}{
		{0, 0,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000001},
		{0, 1,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000001},
		{1, 0,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b1000000000000000000000000000000000000000000000000000000000000001},
		{1, 2,
			0b1000000000000000000000000000000000000000000000000000000000000001,
			0b0010000000000000000000000000000000000000000000000000000000000000},
		{1, 32,
			0b0111111111111111111111111111111111111111111111111111111111111110,
			0b0000000000000000000000000000000001111111111111111111111111111111},
		{13, 7,
			0b0111111111111111111111111111111111111111111111111111111111111110,
			0b0000000000000000000000000000011111111111111111111111111111111111},
	}

	for i, v := range testValues {
		bits := shiftBitsIn(v.bits, v.keyNum, v.bitSize)

		if v.expected != bits {
			t.Errorf("Unexpected shifted mask for key %d of size %d (%d)."+
				"\nexpected: %064b\nreceived: %064b",
				v.keyNum, v.bitSize, i, v.expected, bits)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a MultiStateVector loaded from storage via LoadMultiStateVector
// matches the original.
func TestLoadMultiStateVector(t *testing.T) {
	key := "TestLoadMultiStateVector"
	msv, kv := newTestFilledMSV(156, 5, nil, "TestLoadMultiStateVector", t)

	// Attempt to load MultiStateVector from storage
	loadedSV, err := LoadMultiStateVector(nil, key, kv)
	if err != nil {
		t.Fatalf("LoadMultiStateVector returned an error: %+v", err)
	}

	if !reflect.DeepEqual(msv, loadedSV) {
		t.Errorf("Loaded MultiStateVector does not match original saved."+
			"\nexpected: %+v\nreceived: %+v", msv, loadedSV)
	}
}

// Error path: tests that LoadMultiStateVector returns the expected error when
// no object is saved in storage.
func TestLoadMultiStateVector_GetFromStorageError(t *testing.T) {
	key := "TestLoadMultiStateVector_GetFromStorageError"
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedErr := strings.Split(loadGetMsvErr, "%")[0]

	_, err := LoadMultiStateVector(nil, key, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("LoadMultiStateVector did not return the expected error "+
			"when no object exists in storage.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that LoadMultiStateVector returns the expected error when
// the data in storage cannot be unmarshalled.
func TestLoadMultiStateVector_UnmarshalError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := "TestLoadMultiStateVector_MarshalError"
	expectedErr := strings.Split(loadUnmarshalMsvErr, "%")[0]

	// Save invalid data to storage
	err := kv.Set(makeMultiStateVectorKey(key),
		&versioned.Object{
			Version:   multiStateVectorVersion,
			Timestamp: netTime.Now(),
			Data:      []byte("?"),
		})
	if err != nil {
		t.Errorf("Failed to save data to storage: %+v", err)
	}

	_, err = LoadMultiStateVector(nil, key, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("LoadMultiStateVector did not return the expected error "+
			"when the object in storage should be invalid."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the data saved via MultiStateVector.save to storage matches the
// expected data.
func TestMultiStateVector_save(t *testing.T) {
	msv := &MultiStateVector{
		numKeys:         3 * 64,
		numStates:       5,
		bitSize:         3,
		numKeysPerBlock: 21,
		wordSize:        63,
		vect:            []uint64{0, 1, 2},
		stateUseCount:   []uint16{5, 12, 104, 0, 4000},
		key:             makeStateVectorKey("TestMultiStateVector_save"),
		kv:              versioned.NewKV(ekv.MakeMemstore()),
	}

	expectedData, err := msv.marshal()
	if err != nil {
		t.Errorf("Failed to marshal MultiStateVector: %+v", err)
	}

	// Save to storage
	err = msv.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	// Check that the object can be loaded
	loadedData, err := msv.kv.Get(msv.key, multiStateVectorVersion)
	if err != nil {
		t.Errorf("Failed to load MultiStateVector from storage: %+v", err)
	}

	if !bytes.Equal(expectedData, loadedData.Data) {
		t.Errorf("Loaded data does not match expected."+
			"\nexpected: %v\nreceived: %v", expectedData, loadedData)
	}
}

// Tests that MultiStateVector.Delete removes the MultiStateVector from storage.
func TestMultiStateVector_Delete(t *testing.T) {
	msv, _ := newTestFilledMSV(156, 5, nil, "TestMultiStateVector_Delete", t)

	err := msv.Delete()
	if err != nil {
		t.Errorf("Delete returned an error: %+v", err)
	}

	// Check that the object can be loaded
	loadedData, err := msv.kv.Get(msv.key, multiStateVectorVersion)
	if err == nil {
		t.Errorf("Loaded MultiStateVector from storage when it should be "+
			"deleted: %v", loadedData)
	}
}

// Tests that a MultiStateVector marshalled with MultiStateVector.marshal and
// unmarshalled with MultiStateVector.unmarshal matches the original.
func TestMultiStateVector_marshal_unmarshal(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	msv := &MultiStateVector{
		numKeys:         3 * 64,
		numStates:       5,
		bitSize:         3,
		numKeysPerBlock: 21,
		wordSize:        63,
		vect:            []uint64{prng.Uint64(), prng.Uint64(), prng.Uint64()},
		stateUseCount:   []uint16{5, 12, 104, 0, 4000},
	}

	marshalledBytes, err := msv.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	newMsv := &MultiStateVector{}
	err = newMsv.unmarshal(marshalledBytes)
	if err != nil {
		t.Errorf("unmarshal returned an error: %+v", err)
	}

	if !reflect.DeepEqual(&msv, &newMsv) {
		t.Errorf("Marshalled and unmarsalled MultiStateVector does not match "+
			"the original.\nexpected: %+v\nreceived: %+v", msv, newMsv)
	}
}

// Consistency test of makeMultiStateVectorKey.
func Test_makeMultiStateVectorKey(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedStrings := []string{
		multiStateVectorKey + "U4x/lrFkvxuXu59LtHLon1sU",
		multiStateVectorKey + "hPJSCcnZND6SugndnVLf15tN",
		multiStateVectorKey + "dkKbYXoMn58NO6VbDMDWFEyI",
		multiStateVectorKey + "hTWEGsvgcJsHWAg/YdN1vAK0",
		multiStateVectorKey + "HfT5GSnhj9qeb4LlTnSOgeee",
		multiStateVectorKey + "S71v40zcuoQ+6NY+jE/+HOvq",
		multiStateVectorKey + "VG2PrBPdGqwEzi6ih3xVec+i",
		multiStateVectorKey + "x44bC6+uiBuCp1EQikLtPJA8",
		multiStateVectorKey + "qkNGWnhiBhaXiu0M48bE8657",
		multiStateVectorKey + "w+BJW1cS/v2+DBAoh+EA2s0t",
	}

	for i, expected := range expectedStrings {
		b := make([]byte, 18)
		prng.Read(b)
		key := makeMultiStateVectorKey(base64.StdEncoding.EncodeToString(b))

		if expected != key {
			t.Errorf("New MultiStateVector key does not match expected (%d)."+
				"\nexpected: %q\nreceived: %q", i, expected, key)
		}
	}
}

// newTestFilledMSV produces a new MultiStateVector and sets each key to a
// random state.
func newTestFilledMSV(numKeys uint16, numStates uint8, stateMap [][]bool,
	key string, t *testing.T) (*MultiStateVector, *versioned.KV) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	msv, err := NewMultiStateVector(numKeys, numStates, stateMap, key, kv)
	if err != nil {
		t.Errorf("Failed to create new MultiStateVector: %+v", err)
	}

	prng := rand.New(rand.NewSource(42))
	for keyNum := uint16(0); keyNum < msv.numKeys; keyNum++ {
		state := uint8(prng.Intn(int(msv.numStates)))

		err = msv.Set(keyNum, state)
		if err != nil {
			t.Errorf("Failed to set key %d to state %d.", keyNum, state)
		}
	}

	return msv, kv
}
