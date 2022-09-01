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
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Tests that NewStateVector creates the expected new StateVector and that it is
// saved to storage.
func TestNewStateVector(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := "myTestKey"
	numKeys := uint32(275)
	expected := &StateVector{
		vect:           make([]uint64, (numKeys+63)/64),
		firstAvailable: 0,
		numKeys:        numKeys,
		numAvailable:   numKeys,
		key:            makeStateVectorKey(key),
		kv:             kv,
	}

	received, err := NewStateVector(kv, key, numKeys)
	if err != nil {
		t.Errorf("NewStateVector returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New StateVector does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, received)
	}

	_, err = kv.Get(key, currentStateVectorVersion)
	if err == nil {
		t.Error("New StateVector not saved to storage.")
	}
}

// Tests that StateVector.Use sets the correct keys to used and does not modify
// others keys and that numAvailable is correctly set.
func TestStateVector_Use(t *testing.T) {
	sv := newTestStateVector("StateVectorUse", 138, t)

	// Set some keys to used
	usedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	usedKeysMap := make(map[uint32]bool, len(usedKeys))
	for i, keyNum := range usedKeys {
		sv.Use(keyNum)

		// Check if numAvailable is correct
		if sv.numAvailable != sv.numKeys-uint32(i+1) {
			t.Errorf("numAvailable incorrect (%d).\nexpected: %d\nreceived: %d",
				i, sv.numKeys-uint32(i+1), sv.numAvailable)
		}

		usedKeysMap[keyNum] = true
	}

	// Check all keys for their expected states
	for i := uint32(0); i < sv.numKeys; i++ {
		if usedKeysMap[i] {
			if !sv.Used(i) {
				t.Errorf("Key #%d should have been marked used.", i)
			}
		} else if sv.Used(i) {
			t.Errorf("Key #%d should have been marked unused.", i)
		}
	}

	// Make sure numAvailable is not modified when the key is already used
	sv.Use(usedKeys[0])
	if sv.numAvailable != sv.numKeys-uint32(len(usedKeys)) {
		t.Errorf("numAvailable incorrect.\nexpected: %d\nreceived: %d",
			sv.numKeys-uint32(len(usedKeys)), sv.numAvailable)
	}
}

// Tests that StateVector.UseMany sets the correct keys to used at once and does
// not modify others keys and that numAvailable is correctly set.
func TestStateVector_UseMany(t *testing.T) {
	sv := newTestStateVector("StateVectorUse", 138, t)

	// Set some keys to used
	usedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	usedKeysMap := make(map[uint32]bool, len(usedKeys))
	for _, keyNum := range usedKeys {
		usedKeysMap[keyNum] = true
	}

	// Use all keys
	sv.UseMany(usedKeys...)

	// Check all keys for their expected states
	for i := uint32(0); i < sv.numKeys; i++ {
		if usedKeysMap[i] {
			if !sv.Used(i) {
				t.Errorf("Key #%d should have been marked used.", i)
			}
		} else if sv.Used(i) {
			t.Errorf("Key #%d should have been marked unused.", i)
		}
	}

	// Make sure numAvailable is not modified when the key is already used
	sv.Use(usedKeys[0])
	if sv.numAvailable != sv.numKeys-uint32(len(usedKeys)) {
		t.Errorf("numAvailable incorrect.\nexpected: %d\nreceived: %d",
			sv.numKeys-uint32(len(usedKeys)), sv.numAvailable)
	}

}

// Tests that StateVector.Unuse sets the correct keys to unused and does not
// modify others keys and that numAvailable is correctly set.
func TestStateVector_Unuse(t *testing.T) {
	sv := newTestStateVector("StateVectorUse", 138, t)

	// Set all the keys to used
	for keyNum := uint32(0); keyNum < sv.numKeys; keyNum++ {
		sv.Use(keyNum)
	}

	// Set some keys to unused
	unusedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	unusedKeysMap := make(map[uint32]bool, len(unusedKeys))
	for i, keyNum := range unusedKeys {
		sv.Unuse(keyNum)

		// Check if numAvailable is correct
		if sv.numAvailable != uint32(i)+1 {
			t.Errorf("numAvailable incorrect (%d).\nexpected: %d\nreceived: %d",
				i, uint32(i)+1, sv.numAvailable)
		}

		// Check if firstAvailable is correct
		if sv.firstAvailable != unusedKeys[0] {
			t.Errorf("firstAvailable incorrect (%d).\nexpected: %d\nreceived: %d",
				i, unusedKeys[0], sv.firstAvailable)
		}

		unusedKeysMap[keyNum] = true
	}

	// Check all keys for their expected states
	for i := uint32(0); i < sv.numKeys; i++ {
		if unusedKeysMap[i] {
			if sv.Used(i) {
				t.Errorf("Key #%d should have been marked unused.", i)
			}
		} else if !sv.Used(i) {
			t.Errorf("Key #%d should have been marked used.", i)
		}
	}

	// Make sure numAvailable is not modified when the key is already used
	sv.Unuse(unusedKeys[0])
	if sv.numAvailable != uint32(len(unusedKeys)) {
		t.Errorf("numAvailable incorrect.\nexpected: %d\nreceived: %d",
			uint32(len(unusedKeys)), sv.numAvailable)
	}
}

// Tests that StateVector.Unuse sets the correct keys to unused at the same time
// and does not modify others keys and that numAvailable is correctly set.
func TestStateVector_UnuseMany(t *testing.T) {
	sv := newTestStateVector("StateVectorUse", 138, t)

	// Set all the keys to used
	for keyNum := uint32(0); keyNum < sv.numKeys; keyNum++ {
		sv.Use(keyNum)
	}

	// Set some keys to unused
	unusedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	unusedKeysMap := make(map[uint32]bool, len(unusedKeys))
	for _, keyNum := range unusedKeys {
		unusedKeysMap[keyNum] = true
	}

	sv.UnuseMany(unusedKeys...)

	// Check all keys for their expected states
	for i := uint32(0); i < sv.numKeys; i++ {
		if unusedKeysMap[i] {
			if sv.Used(i) {
				t.Errorf("Key #%d should have been marked unused.", i)
			}
		} else if !sv.Used(i) {
			t.Errorf("Key #%d should have been marked used.", i)
		}
	}

	// Make sure numAvailable is not modified when the key is already used
	sv.Unuse(unusedKeys[0])
	if sv.numAvailable != uint32(len(unusedKeys)) {
		t.Errorf("numAvailable incorrect.\nexpected: %d\nreceived: %d",
			uint32(len(unusedKeys)), sv.numAvailable)
	}
}

// Tests StateVector.Used by creating a vector with known used and unused keys
// and making sure it returns the expected state for all keys in the vector.
func TestStateVector_Used(t *testing.T) {
	numKeys := uint32(128)
	sv := newTestStateVector("StateVectorNext", numKeys, t)

	// Set all keys to used
	for i := range sv.vect {
		sv.vect[i] = 0xFFFFFFFFFFFFFFFF
	}

	// Set some keys to unused
	unusedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	unusedKeysMap := make(map[uint32]bool, len(unusedKeys))
	for _, keyNum := range unusedKeys {
		block, pos := getBlockAndPos(keyNum)
		sv.vect[block] &= ^(1 << pos)
		unusedKeysMap[keyNum] = true
	}

	// Check all keys for their expected states
	for i := uint32(0); i < numKeys; i++ {
		if unusedKeysMap[i] {
			if sv.Used(i) {
				t.Errorf("Key #%d should have been marked unused.", i)
			}
		} else if !sv.Used(i) {
			t.Errorf("Key #%d should have been marked used.", i)
		}
	}
}

// Tests that StateVector.Next returns the expected unused keys and marks them
// as used and that numAvailable is correctly set.
func TestStateVector_Next(t *testing.T) {
	numKeys := uint32(128)
	sv := newTestStateVector("StateVectorNext", numKeys, t)

	// Set all keys to used
	for i := range sv.vect {
		sv.vect[i] = 0xFFFFFFFFFFFFFFFF
	}

	// Set some keys to unused
	unusedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	for _, keyNum := range unusedKeys {
		block, pos := getBlockAndPos(keyNum)
		sv.vect[block] &= ^(1 << pos)
	}

	sv.numAvailable = uint32(len(unusedKeys))
	sv.firstAvailable = unusedKeys[0]

	// Check that each call of Next returns the expected key and that it is
	// marked as used
	for i, expected := range unusedKeys {
		numKey, err := sv.Next()
		if err != nil {
			t.Errorf("Next returned an error (%d): %+v", i, err)
		}

		if expected != numKey {
			t.Errorf("Received key does not match expected."+
				"\nexpected: %d\nreceived: %d", expected, numKey)
		}

		if !sv.used(numKey) {
			t.Errorf("Key #%d not marked as used.", numKey)
		}

		expectedNumAvailable := uint32(len(unusedKeys) - (i + 1))
		if sv.numAvailable != expectedNumAvailable {
			t.Errorf("numAvailable incorrectly set.\nexpected: %d\nreceived: %d",
				expectedNumAvailable, sv.numAvailable)
		}
	}

	// One more call should cause an error
	_, err := sv.Next()
	if err == nil || err.Error() != noKeysErr {
		t.Errorf("Next did not return the expected error when no keys are "+
			"available.\nexpected: %s\nreceived: %+v", noKeysErr, err)
	}

	// firstAvailable should now be beyond the end of the vector
	if sv.firstAvailable < numKeys {
		t.Errorf("firstAvailable should be beyond numKeys."+
			"\nfirstAvailable: %d\nnumKeys:        %d",
			sv.firstAvailable, numKeys)
	}
}

// Tests the StateVector.nextAvailable sets firstAvailable correctly for a known
// key vector.
func TestStateVector_nextAvailable(t *testing.T) {
	numKeys := uint32(128)
	sv := newTestStateVector("StateVectorNext", numKeys, t)

	// Set all keys to used
	for i := range sv.vect {
		sv.vect[i] = 0xFFFFFFFFFFFFFFFF
	}

	// Set some keys to unused
	unusedKeys := []uint32{0, 2, 3, 4, 6, 39, 62, 70, 98, 100}
	for _, keyNum := range unusedKeys {
		block, pos := getBlockAndPos(keyNum)
		sv.vect[block] &= ^(1 << pos)
	}

	for i, keyNum := range unusedKeys {
		if sv.firstAvailable != keyNum {
			t.Errorf("firstAvailable incorrect (%d)."+
				"\nexpected: %d\nreceived: %d", i, keyNum, sv.firstAvailable)
		}
		sv.nextAvailable()
	}
}

// Tests that StateVector.GetNumAvailable returns the expected number of
// available keys after a set number of keys are used.
func TestStateVector_GetNumAvailable(t *testing.T) {
	numKeys := uint32(500)
	sv := newTestStateVector("StateVectorGetNumAvailable", numKeys, t)

	i, n, used := uint32(0), uint32(5), uint32(0)
	for ; i < n; i++ {
		sv.use(i)
	}
	used = n

	if sv.GetNumAvailable() != numKeys-used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", numKeys-used, sv.GetNumAvailable())
	}

	n = uint32(112)
	for ; i < n; i++ {
		sv.use(i)
	}
	used = n

	if sv.GetNumAvailable() != numKeys-used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", numKeys-used, sv.GetNumAvailable())
	}

	i, n = uint32(400), uint32(456)
	for ; i < n; i++ {
		sv.use(i)
	}
	used += n - 400

	if sv.GetNumAvailable() != numKeys-used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", numKeys-used, sv.GetNumAvailable())
	}
}

// Tests that StateVector.GetNumUsed returns the expected number of used keys
// after a set number of keys are used.
func TestStateVector_GetNumUsed(t *testing.T) {
	numKeys := uint32(500)
	sv := newTestStateVector("StateVectorGetNumUsed", numKeys, t)

	i, n, used := uint32(0), uint32(5), uint32(0)
	for ; i < n; i++ {
		sv.use(i)
	}
	used = n

	if sv.GetNumUsed() != used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", used, sv.GetNumUsed())
	}

	n = uint32(112)
	for ; i < n; i++ {
		sv.use(i)
	}
	used = n

	if sv.GetNumUsed() != used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", used, sv.GetNumUsed())
	}

	i, n = uint32(400), uint32(456)
	for ; i < n; i++ {
		sv.use(i)
	}
	used += n - 400

	if sv.GetNumUsed() != used {
		t.Errorf("Got incorrect number of used keys."+
			"\nexpected: %d\nreceived: %d", used, sv.GetNumUsed())
	}
}

// Tests that StateVector.GetNumKeys returns the correct number of keys.
func TestStateVector_GetNumKeys(t *testing.T) {
	numKeys := uint32(32)
	sv := newTestStateVector("StateVectorGetNumKeys", numKeys, t)

	if sv.GetNumKeys() != numKeys {
		t.Errorf("Got incorrect number of keys.\nexpected: %d\nreceived: %d",
			numKeys, sv.GetNumKeys())
	}
}

// Tests that StateVector.GetUnusedKeyNums returns a list of all odd-numbered
// keys when all even-numbered keys are used.
func TestStateVector_GetUnusedKeyNums(t *testing.T) {
	numKeys := uint32(1000)
	sv := newTestStateVector("StateVectorGetUnusedKeyNums", numKeys, t)

	// Use every other key
	for i := uint32(0); i < numKeys; i += 2 {
		sv.use(i)
	}

	sv.firstAvailable = 1

	// Check that every other key is in the list
	for i, keyNum := range sv.GetUnusedKeyNums() {
		if keyNum != uint32(2*i)+1 {
			t.Errorf("Key number #%d incorrect."+
				"\nexpected: %d\nreceived: %d", i, 2*i+1, keyNum)
		}
	}
}

// Tests that StateVector.GetUsedKeyNums returns a list of all even-numbered
// keys when all even-numbered keys are used.
func TestStateVector_GetUsedKeyNums(t *testing.T) {
	numKeys := uint32(1000)
	sv := newTestStateVector("StateVectorGetUsedKeyNums", numKeys, t)

	// Use every other key
	for i := uint32(0); i < numKeys; i += 2 {
		sv.use(i)
	}

	// Check that every other key is in the list
	for i, keyNum := range sv.GetUsedKeyNums() {
		if keyNum != uint32(2*i) {
			t.Errorf("Key number #%d incorrect."+
				"\nexpected: %d\nreceived: %d", i, 2*i, keyNum)
		}
	}
}

// Tests that StateVector.DeepCopy makes a copy of the values and not of the
// pointers.
func TestStateVector_DeepCopy(t *testing.T) {
	sv := newTestStateVector("StateVectorGetUsedKeyNums", 1000, t)

	newSV := sv.DeepCopy()
	sv.kv = nil

	// Check that the values are the same
	if !reflect.DeepEqual(sv, newSV) {
		t.Errorf("Original and copy do not match."+
			"\nexpected: %#v\nreceived: %#v", sv, newSV)
	}

	// Check that the pointers are different
	if sv == newSV {
		t.Errorf("Original and copy do not match."+
			"\nexpected: %p\nreceived: %p", sv, newSV)
	}

	if &sv.vect == &newSV.vect {
		t.Errorf("Original and copy do not match."+
			"\nexpected: %p\nreceived: %p", sv.vect, newSV.vect)
	}
}

// Tests that StateVector.String returns the expected string.
func TestStateVector_String(t *testing.T) {
	key := "StateVectorString"
	expected := "stateVector: " + makeStateVectorKey(key)
	sv := newTestStateVector(key, 500, t)
	// Use every other key
	for i := uint32(0); i < sv.numKeys; i += 2 {
		sv.use(i)
	}

	if expected != sv.String() {
		t.Errorf("String does not match expected.\nexpected: %q\nreceived: %q",
			expected, sv.String())
	}
}

// Tests that StateVector.GoString returns the expected string.
func TestStateVector_GoString(t *testing.T) {
	expected := "{vect:[6148914691236517205 6148914691236517205 " +
		"6148914691236517205 6148914691236517205 6148914691236517205 " +
		"6148914691236517205 6148914691236517205 1501199875790165] " +
		"firstAvailable:1 numKeys:500 numAvailable:250 " +
		"key:stateVectorStateVectorGoString"
	sv := newTestStateVector("StateVectorGoString", 500, t)
	// Use every other key
	for i := uint32(0); i < sv.numKeys; i += 2 {
		sv.use(i)
	}

	received := strings.Split(sv.GoString(), " kv:")[0]
	if expected != received {
		t.Errorf("String does not match expected.\nexpected: %q\nreceived: %q",
			expected, received)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a StateVector loaded from storage via LoadStateVector matches the
// original.
func TestLoadStateVector(t *testing.T) {
	numKeys := uint32(1000)
	key := "StateVectorLoadStateVector"
	sv := newTestStateVector(key, numKeys, t)

	// Use every other key
	for i := uint32(0); i < numKeys; i += 2 {
		sv.Use(i)
	}

	// Attempt to load StateVector from storage
	loadedSV, err := LoadStateVector(sv.kv, key)
	if err != nil {
		t.Fatalf("LoadStateVector returned an error: %+v", err)
	}

	if !reflect.DeepEqual(sv.vect, loadedSV.vect) {
		t.Errorf("Loaded StateVector does not match original saved."+
			"\nexpected: %#v\nreceived: %#v", sv.vect, loadedSV.vect)
	}
}

// Tests that a StateVector loaded from storage via LoadStateVector matches the
// original.
func TestLoadStateVector_GetError(t *testing.T) {
	key := "StateVectorLoadStateVector"
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedErr := "object not found"

	_, err := LoadStateVector(kv, key)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("LoadStateVector did not return the expected error when no "+
			"object exists in storage.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that a StateVector loaded from storage via LoadStateVector matches the
// original.
func TestLoadStateVector_UnmarshalError(t *testing.T) {
	key := "StateVectorLoadStateVector"
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Save invalid StateVector to storage
	obj := versioned.Object{
		Version:   currentStateVectorVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("invalidStateVector"),
	}
	err := kv.Set(makeStateVectorKey(key), currentStateVectorVersion, &obj)
	if err != nil {
		t.Errorf("Failed to save invalid StateVector to storage: %+v", err)
	}

	expectedErr := strings.Split(loadUnmarshalErr, "%")[0]
	_, err = LoadStateVector(kv, key)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("LoadStateVector did not return the expected error when the "+
			"data in storage is invalid.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that StateVector.save saves the correct data to storage and that it can
// be loaded.
func TestStateVector_save(t *testing.T) {
	key := "StateVectorSave"
	sv := &StateVector{
		vect:           make([]uint64, (1000+63)/64),
		firstAvailable: 0,
		numKeys:        1000,
		numAvailable:   1000,
		key:            makeStateVectorKey(key),
		kv:             versioned.NewKV(ekv.MakeMemstore()),
	}
	expectedData, err := sv.marshal()
	if err != nil {
		t.Errorf("Failed to marshal StateVector: %+v", err)
	}

	// Save to storage
	err = sv.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	// Check that the object can be loaded
	loadedData, err := sv.kv.Get(sv.key, currentStateVectorVersion)
	if err != nil {
		t.Errorf("Failed to load StateVector from storage: %+v", err)
	}

	if !bytes.Equal(expectedData, loadedData.Data) {
		t.Errorf("Loaded data does not match expected."+
			"\nexpected: %v\nreceived: %v", expectedData, loadedData)
	}
}

// Tests that StateVector.Delete removes the StateVector from storage.
func TestStateVector_Delete(t *testing.T) {
	sv := newTestStateVector("StateVectorDelete", 1000, t)

	err := sv.Delete()
	if err != nil {
		t.Errorf("Delete returned an error: %+v", err)
	}

	// Check that the object can be loaded
	loadedData, err := sv.kv.Get(sv.key, currentStateVectorVersion)
	if err == nil {
		t.Errorf("Loaded StateVector from storage when it should be deleted: %v",
			loadedData)
	}
}

func TestStateVector_marshal_unmarshal(t *testing.T) {
	// Generate new StateVector and use ever other key
	sv1 := newTestStateVector("StateVectorMarshalUnmarshal", 224, t)
	for i := uint32(0); i < sv1.GetNumKeys(); i += 2 {
		sv1.Use(i)
	}

	// Marshal and unmarshal the StateVector
	marshalledData, err := sv1.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	// Unmarshal into new StateVector
	sv2 := &StateVector{key: sv1.key, kv: sv1.kv}
	err = sv2.unmarshal(marshalledData)
	if err != nil {
		t.Errorf("unmarshal returned an error: %+v", err)
	}

	// Make sure that the unmarshalled StateVector matches the original
	if !reflect.DeepEqual(sv1, sv2) {
		t.Errorf("Marshalled and unmarshalled StateVector does not match "+
			"original.\nexpected: %#v\nreceived: %#v", sv1, sv2)
	}
}

// Consistency test of makeStateVectorKey.
func Test_makeStateVectorKey(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedStrings := []string{
		"stateVectorU4x/lrFkvxuXu59LtHLon1sU",
		"stateVectorhPJSCcnZND6SugndnVLf15tN",
		"stateVectordkKbYXoMn58NO6VbDMDWFEyI",
		"stateVectorhTWEGsvgcJsHWAg/YdN1vAK0",
		"stateVectorHfT5GSnhj9qeb4LlTnSOgeee",
		"stateVectorS71v40zcuoQ+6NY+jE/+HOvq",
		"stateVectorVG2PrBPdGqwEzi6ih3xVec+i",
		"stateVectorx44bC6+uiBuCp1EQikLtPJA8",
		"stateVectorqkNGWnhiBhaXiu0M48bE8657",
		"stateVectorw+BJW1cS/v2+DBAoh+EA2s0t",
	}

	for i, expected := range expectedStrings {
		b := make([]byte, 18)
		prng.Read(b)
		key := makeStateVectorKey(base64.StdEncoding.EncodeToString(b))

		if expected != key {
			t.Errorf("New StateVector key does not match expected (%d)."+
				"\nexpected: %q\nreceived: %q", i, expected, key)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Testing Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that StateVector.SaveTEST saves the correct data to storage and that it
// can be loaded.
func TestStateVector_SaveTEST(t *testing.T) {
	key := "StateVectorSaveTEST"
	sv := &StateVector{
		vect:           make([]uint64, (1000+63)/64),
		firstAvailable: 0,
		numKeys:        1000,
		numAvailable:   1000,
		key:            makeStateVectorKey(key),
		kv:             versioned.NewKV(ekv.MakeMemstore()),
	}
	expectedData, err := sv.marshal()
	if err != nil {
		t.Errorf("Failed to marshal StateVector: %+v", err)
	}

	// Save to storage
	err = sv.SaveTEST(t)
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	// Check that the object can be loaded
	loadedData, err := sv.kv.Get(sv.key, currentStateVectorVersion)
	if err != nil {
		t.Errorf("Failed to load StateVector from storage: %+v", err)
	}

	if !bytes.Equal(expectedData, loadedData.Data) {
		t.Errorf("Loaded data does not match expected."+
			"\nexpected: %v\nreceived: %v", expectedData, loadedData)
	}
}

// Panic path: tests that StateVector.SaveTEST panics when provided a non-
// testing interface.
func TestStateVector_SaveTEST_InvalidInterfaceError(t *testing.T) {
	sv := &StateVector{}
	expectedErr := fmt.Sprintf(testInterfaceErr, "SaveTEST")

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("Failed to panic with expected error when provided a "+
				"non-testing interface.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	_ = sv.SaveTEST(struct{}{})
}

// Tests that StateVector.SetFirstAvailableTEST correctly sets firstAvailable.
func TestStateVector_SetFirstAvailableTEST(t *testing.T) {
	sv := newTestStateVector("StateVectorSetFirstAvailableTEST", 1000, t)

	firstAvailable := uint32(78)
	sv.SetFirstAvailableTEST(firstAvailable, t)

	if sv.firstAvailable != firstAvailable {
		t.Errorf("Failed to set firstAvailable.\nexpected: %d\nreceived: %d",
			firstAvailable, sv.firstAvailable)
	}
}

// Panic path: tests that StateVector.SetFirstAvailableTEST panics when provided
// a non-testing interface.
func TestStateVector_SetFirstAvailableTEST_InvalidInterfaceError(t *testing.T) {
	sv := &StateVector{}
	expectedErr := fmt.Sprintf(testInterfaceErr, "SetFirstAvailableTEST")

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("Failed to panic with expected error when provided a "+
				"non-testing interface.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	sv.SetFirstAvailableTEST(0, struct{}{})
}

// Tests that StateVector.SetNumKeysTEST correctly sets numKeys.
func TestStateVector_SetNumKeysTEST(t *testing.T) {
	sv := newTestStateVector("StateVectorSetNumKeysTEST", 1000, t)

	numKeys := uint32(78)
	sv.SetNumKeysTEST(numKeys, t)

	if sv.numKeys != numKeys {
		t.Errorf("Failed to set numKeys.\nexpected: %d\nreceived: %d",
			numKeys, sv.numKeys)
	}
}

// Panic path: tests that StateVector.SetNumKeysTEST panics when provided a non-
// testing interface.
func TestStateVector_SetNumKeysTEST_InvalidInterfaceError(t *testing.T) {
	sv := &StateVector{}
	expectedErr := fmt.Sprintf(testInterfaceErr, "SetNumKeysTEST")

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("Failed to panic with expected error when provided a "+
				"non-testing interface.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	sv.SetNumKeysTEST(0, struct{}{})
}

// Tests that StateVector.SetNumAvailableTEST correctly sets numKeys.
func TestStateVector_SetNumAvailableTEST(t *testing.T) {
	sv := newTestStateVector("StateVectorSetNumAvailableTEST", 1000, t)

	numAvailable := uint32(78)
	sv.SetNumAvailableTEST(numAvailable, t)

	if sv.numAvailable != numAvailable {
		t.Errorf("Failed to set numAvailable.\nexpected: %d\nreceived: %d",
			numAvailable, sv.numAvailable)
	}
}

// Panic path: tests that StateVector.SetNumAvailableTEST panics when provided a
// non-testing interface.
func TestStateVector_SetNumAvailableTEST_InvalidInterfaceError(t *testing.T) {
	sv := &StateVector{}
	expectedErr := fmt.Sprintf(testInterfaceErr, "SetNumAvailableTEST")

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("Failed to panic with expected error when provided a "+
				"non-testing interface.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	sv.SetNumAvailableTEST(0, struct{}{})
}

// Tests that StateVector.SetKvTEST correctly sets the versioned.KV.
func TestStateVector_SetKvTEST(t *testing.T) {
	sv := newTestStateVector("SetKvTEST", 1000, t)

	kv := versioned.NewKV(ekv.MakeMemstore()).Prefix("NewKV")
	sv.SetKvTEST(kv, t)

	if sv.kv != kv {
		t.Errorf("Failed to set the KV.\nexpected: %v\nreceived: %v", kv, sv.kv)
	}
}

// Panic path: tests that StateVector.SetKvTEST panics when provided a non-
// testing interface.
func TestStateVector_SetKvTEST_InvalidInterfaceError(t *testing.T) {
	sv := &StateVector{}
	expectedErr := fmt.Sprintf(testInterfaceErr, "SetKvTEST")

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("Failed to panic with expected error when provided a "+
				"non-testing interface.\nexpected: %s\nreceived: %+v",
				expectedErr, r)
		}
	}()

	sv.SetKvTEST(nil, struct{}{})
}

// newTestStateVector produces a new StateVector using the specified number of
// keys and key string for testing.
func newTestStateVector(key string, numKeys uint32, t *testing.T) *StateVector {
	kv := versioned.NewKV(ekv.MakeMemstore())

	sv, err := NewStateVector(kv, key, numKeys)
	if err != nil {
		t.Fatalf("Failed to create new StateVector: %+v", err)
	}

	return sv
}
