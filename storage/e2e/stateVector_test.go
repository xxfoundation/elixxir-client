package e2e

import (
	"fmt"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"math/bits"
	"reflect"
	"testing"
)

// GetNumAvailable gets the number of slots left in the state vector
func TestStateVector_GetNumAvailable(t *testing.T) {
	const numAvailable = 23
	sv := &stateVector{
		numAvailable: numAvailable,
	}
	// At the start, NumAvailable should be the same as numKeys
	// as none of the keys have been used
	if sv.GetNumAvailable() != numAvailable {
		t.Errorf("expected %v available, actually %v available", numAvailable, sv.GetNumAvailable())
	}
}

// Shows that GetNumUsed returns the number of slots used in the state vector
func TestStateVector_GetNumUsed(t *testing.T) {
	const numAvailable = 23
	const numKeys = 50
	sv := &stateVector{
		numkeys:      numKeys,
		numAvailable: numAvailable,
	}

	if sv.GetNumUsed() != numKeys-numAvailable {
		t.Errorf("Expected %v used, got %v", numKeys-numAvailable, sv.GetNumUsed())
	}
}

func TestStateVector_GetNumKeys(t *testing.T) {
	const numKeys = 32
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}

	// GetNumKeys should always be the same as numKeys
	if sv.GetNumKeys() != numKeys {
		t.Errorf("expected %v available, actually %v available", numKeys, sv.GetNumAvailable())
	}
}

// Shows that Next mutates vector state as expected
// Shows that Next can find key indexes all throughout the bitfield
func TestStateVector_Next(t *testing.T) {
	// Expected results: all keynums, and beyond the last key
	expectedFirstAvail := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}

	const numKeys = 1000
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}

	// Set all bits to dirty to start
	for i := range sv.vect {
		sv.vect[i] = 0xffffffffffffffff
	}

	// Set a few clean bits randomly
	const numBitsSet = 10
	for i := 0; i < numBitsSet; i++ {
		keyNum := expectedFirstAvail[i]
		// Set a bit clean in the state vector
		vectIndex := keyNum / 64
		bitIndex := keyNum % 64
		sv.vect[vectIndex] &= ^bits.RotateLeft64(uint64(1), int(bitIndex))
	}

	sv.numAvailable = numBitsSet
	sv.nextAvailable()

	// Calling Next ten times should give all of the keyNums we set
	// It should change firstAvailable, but doesn't mutate the bit field itself
	//  (that should be done with Use)
	for numCalls := 0; numCalls < numBitsSet; numCalls++ {
		keyNum, err := sv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if keyNum != expectedFirstAvail[numCalls] {
			t.Errorf("keynum %v didn't match expected %v at index %v", keyNum, expectedFirstAvail[numCalls], numCalls)
		}
	}

	// One more call should cause an error
	_, err = sv.Next()
	if err == nil {
		t.Error("Calling Next() after all keys have been found should result in error, as firstAvailable is more than numKeys")
	}
	// firstAvailable should now be beyond the end of the bitfield
	if sv.firstAvailable < numKeys {
		t.Error("Last Next() call should have set firstAvailable beyond numKeys")
	}
}

// Shows that Use() mutates the state vector itself
func TestStateVector_Use(t *testing.T) {
	// These keyNums will be set to dirty with Use
	keyNums := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}

	const numKeys = 1000
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}

	// Expected vector states as bits are set
	var expectedVect [][]uint64
	expectedVect = append(expectedVect, []uint64{0, 0, 0x800, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x1000000000, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x81000000000, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x2000081000000000, 0, 0})
	expectedVect = append(expectedVect, []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x2000081000000000, 0, 0x800000000})

	for numCalls := range keyNums {
		// These calls to Use won't set nextAvailable, because the first keyNum set
		sv.Use(keyNums[numCalls])
		if !reflect.DeepEqual(expectedVect[numCalls], sv.vect) {
			t.Errorf("sv.vect differed from expected at index %v", numCalls)
			fmt.Println(sv.vect)
		}
	}
}

func TestStateVector_Used(t *testing.T) {
	// These keyNums should be used
	keyNums := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}

	const numKeys = 1000
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}
	sv.vect = []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x2000081000000000, 0, 0x800000000}

	for i := uint32(0); i < numKeys; i++ {
		// if i is in keyNums, Used should be true
		// otherwise, it should be false
		found := false
		for j := range keyNums {
			if i == keyNums[j] {
				found = true
				break
			}
		}
		if sv.Used(i) != found {
			t.Errorf("at keynum %v Used should have been %v but was %v", i, found, sv.Used(i))
		}
	}
}

// Shows that the GetUsedKeyNums method returns the correct keynums
func TestStateVector_GetUsedKeyNums(t *testing.T) {
	// These keyNums should be used
	keyNums := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}

	const numKeys = 1000
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}
	sv.vect = []uint64{0, 0, 0x20800, 0, 0x100000000000, 0x10000000000, 0x1000000000, 0, 0, 0, 0, 0x200000000000000, 0, 0x2000081000000000, 0, 0x800000000}
	sv.numAvailable = uint32(numKeys - len(keyNums))

	usedKeyNums := sv.GetUsedKeyNums()
	for i := range keyNums {
		if usedKeyNums[i] != keyNums[i] {
			t.Errorf("used keynums at %v: expected %v, got %v", i, keyNums[i], usedKeyNums[i])
		}
	}
}

// Shows that GetUnusedKeyNums gets all clean keynums
func TestStateVector_GetUnusedKeyNums(t *testing.T) {
	// These keyNums should not be used
	keyNums := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}

	const numKeys = 1000
	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}
	sv.vect = []uint64{0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffdf7ff, 0xffffffffffffffff, 0xffffefffffffffff, 0xfffffeffffffffff, 0xffffffefffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfdffffffffffffff, 0xffffffffffffffff, 0xdffff7efffffffff, 0xffffffffffffffff, 0xfffffff7ffffffff}
	sv.numAvailable = uint32(len(keyNums))
	sv.firstAvailable = keyNums[0]

	unusedKeyNums := sv.GetUnusedKeyNums()
	for i := range keyNums {
		if unusedKeyNums[i] != keyNums[i] {
			t.Errorf("unused keynums at %v: expected %v, got %v", i, keyNums[i], unusedKeyNums[i])
		}
	}
	if len(keyNums) != len(unusedKeyNums) {
		t.Error("array lengths differed, so arrays must be different")
	}
}

// Serializing and deserializing should result in the same state vector
func TestLoadStateVector(t *testing.T) {
	keyNums := []uint32{139, 145, 300, 360, 420, 761, 868, 875, 893, 995}
	const numKeys = 1000

	sv, err := newStateVector(versioned.NewKV(make(ekv.Memstore)), "key", numKeys)
	if err != nil {
		t.Fatal(err)
	}
	sv.vect = []uint64{0xffffffffffffffff, 0xffffffffffffffff, 0xfffffffffffdf7ff, 0xffffffffffffffff, 0xffffefffffffffff, 0xfffffeffffffffff, 0xffffffefffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xfdffffffffffffff, 0xffffffffffffffff, 0xdffff7efffffffff, 0xffffffffffffffff, 0xfffffff7ffffffff}
	sv.numAvailable = uint32(len(keyNums))
	sv.firstAvailable = keyNums[0]

	err = sv.save()
	if err != nil {
		t.Fatal(err)
	}
	sv2, err := loadStateVector(versioned.NewKV(make(ekv.Memstore)), "key")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sv.vect, sv2.vect) {
		t.Error("state vectors different after deserialization")
	}
}
