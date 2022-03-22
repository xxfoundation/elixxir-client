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
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Tests that newTransferredBundle returns the expected transferredBundle and
// that it can be loaded from storage.
func Test_newTransferredBundle(t *testing.T) {
	expectedTB := &transferredBundle{
		list: make(map[id.Round][]uint16),
		key:  "testKey1",
		kv:   versioned.NewKV(make(ekv.Memstore)),
	}

	tb, err := newTransferredBundle(expectedTB.key, expectedTB.kv)
	if err != nil {
		t.Errorf("newTransferredBundle produced an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedTB, tb) {
		t.Errorf("New transferredBundle does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedTB, tb)
	}

	_, err = expectedTB.kv.Get(
		makeTransferredBundleKey(expectedTB.key), transferredBundleVersion)
	if err != nil {
		t.Errorf("Failed to load transferredBundle from storage: %+v", err)
	}
}

// Tests that transferredBundle.addPartNums adds the part numbers for the round
// ID to the map correctly.
func Test_transferredBundle_addPartNums(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	key := "testKey"
	tb, err := newTransferredBundle(key, kv)
	if err != nil {
		t.Errorf("Failed to create new transferredBundle: %+v", err)
	}

	rid := id.Round(10)
	expectedPartNums := []uint16{5, 128, 23, 1}

	err = tb.addPartNums(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("addPartNums returned an error: %+v", err)
	}

	partNums, exists := tb.list[rid]
	if !exists || !reflect.DeepEqual(expectedPartNums, partNums) {
		t.Errorf("Part numbers in memory does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedPartNums, partNums)
	}
}

// Tests that transferredBundle.getPartNums returns the expected part numbers
func Test_transferredBundle_getPartNums(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	key := "testKey"
	tb, err := newTransferredBundle(key, kv)
	if err != nil {
		t.Errorf("Failed to create new transferredBundle: %+v", err)
	}

	rid := id.Round(10)
	expectedPartNums := []uint16{5, 128, 23, 1}

	err = tb.addPartNums(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("failed to add part numbers: %+v", err)
	}

	partNums, exists := tb.getPartNums(rid)
	if !exists || !reflect.DeepEqual(expectedPartNums, partNums) {
		t.Errorf("Part numbers in memory does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedPartNums, partNums)
	}
}

// Tests that transferredBundle.getNumParts returns the correct number of parts
// after parts are added and removed from the list.
func Test_transferredBundle_getNumParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	tb, err := newTransferredBundle("testKey", versioned.NewKV(make(ekv.Memstore)))
	if err != nil {
		t.Errorf("Failed to create new transferredBundle: %+v", err)
	}

	// Add 10 random lists of part numbers to the map
	var expectedNumParts uint16
	for i := 0; i < 10; i++ {
		partNums := make([]uint16, prng.Intn(16))
		for j := range partNums {
			partNums[j] = uint16(prng.Uint32())
		}

		// Add number of parts for odd numbered rounds
		if i%2 == 1 {
			expectedNumParts += uint16(len(partNums))
		}

		err = tb.addPartNums(id.Round(i), partNums...)
		if err != nil {
			t.Errorf("Failed to add part #%d: %+v", i, err)
		}
	}

	// Delete num parts for even numbered rounds
	for i := 0; i < 10; i += 2 {
		err = tb.deletePartNums(id.Round(i))
		if err != nil {
			t.Errorf("Failed to delete part #%d: %+v", i, err)
		}
	}

	// get number of parts
	receivedNumParts := tb.getNumParts()

	if expectedNumParts != receivedNumParts {
		t.Errorf("Failed to get expected number of parts."+
			"\nexpected: %d\nreceived: %d", expectedNumParts, receivedNumParts)
	}
}

// Tests that transferredBundle.deletePartNums deletes the part number from
// memory.
func Test_transferredBundle_deletePartNums(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	key := "testKey"
	tb, err := newTransferredBundle(key, kv)
	if err != nil {
		t.Errorf("Failed to create new transferredBundle: %+v", err)
	}

	rid := id.Round(10)
	expectedPartNums := []uint16{5, 128, 23, 1}

	err = tb.addPartNums(rid, expectedPartNums...)
	if err != nil {
		t.Errorf("failed to add part numbers: %+v", err)
	}

	err = tb.deletePartNums(rid)
	if err != nil {
		t.Errorf("deletePartNums returned an error: %+v", err)
	}

	_, exists := tb.list[rid]
	if exists {
		t.Error("Found part numbers that should have been deleted.")
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that loadTransferredBundle returns a transferredBundle from storage
// that matches the original in memory.
func Test_loadTransferredBundle(t *testing.T) {
	expectedTB := &transferredBundle{
		list: map[id.Round][]uint16{
			1: {1, 2, 3},
			2: {4, 5, 6},
			3: {7, 8, 9},
		},
		numParts: 9,
		key:      "testKey2",
		kv:       versioned.NewKV(make(ekv.Memstore)),
	}

	err := expectedTB.save()
	if err != nil {
		t.Errorf("Failed to save transferredBundle to storage: %+v", err)
	}

	tb, err := loadTransferredBundle(expectedTB.key, expectedTB.kv)
	if err != nil {
		t.Errorf("loadTransferredBundle returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedTB, tb) {
		t.Errorf("transferredBundle loaded from storage does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedTB, tb)
	}
}

// Error path: tests that loadTransferredBundle returns the expected error when
// there is no transferredBundle in storage.
func Test_loadTransferredBundle_LoadError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadTransferredBundleErr, "%")[0]

	_, err := loadTransferredBundle("", kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadTransferredBundle did not returned the expected error "+
			"when no transferredBundle exists in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that transferredBundle.save saves the correct data to storage.
func Test_transferredBundle_save(t *testing.T) {
	tb := &transferredBundle{
		list: map[id.Round][]uint16{
			1: {1, 2, 3},
			2: {4, 5, 6},
			3: {7, 8, 9},
		},
		key: "testKey3",
		kv:  versioned.NewKV(make(ekv.Memstore)),
	}
	expectedData := tb.marshal()

	err := tb.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	vo, err := tb.kv.Get(
		makeTransferredBundleKey(tb.key), transferredBundleVersion)
	if err != nil {
		t.Errorf("Failed to load transferredBundle from storage: %+v", err)
	}

	sort.SliceStable(expectedData, func(i, j int) bool {
		return expectedData[i] > expectedData[j]
	})

	sort.SliceStable(vo.Data, func(i, j int) bool {
		return vo.Data[i] > vo.Data[j]
	})

	if !bytes.Equal(expectedData, vo.Data) {
		t.Errorf("Loaded transferredBundle does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedData, vo.Data)
	}
}

// Tests that transferredBundle.delete removes a saved transferredBundle from
// storage.
func Test_transferredBundle_delete(t *testing.T) {
	tb := &transferredBundle{
		list: map[id.Round][]uint16{
			1: {1, 2, 3},
			2: {4, 5, 6},
			3: {7, 8, 9},
		},
		key: "testKey4",
		kv:  versioned.NewKV(make(ekv.Memstore)),
	}

	err := tb.save()
	if err != nil {
		t.Errorf("Failed to save transferredBundle to storage: %+v", err)
	}

	err = tb.delete()
	if err != nil {
		t.Errorf("delete returned an error: %+v", err)
	}

	_, err = tb.kv.Get(makeTransferredBundleKey(tb.key), transferredBundleVersion)
	if err == nil {
		t.Error("Read transferredBundleVersion from storage when it should" +
			"have been deleted.")
	}
}

// Tests that a transferredBundle that is marshalled via
// transferredBundle.marshal and unmarshalled via transferredBundle.unmarshal
// matches the original.
func Test_transferredBundle_marshal_unmarshal(t *testing.T) {
	expectedTB := &transferredBundle{
		list: map[id.Round][]uint16{
			1: {1, 2, 3},
			2: {4, 5, 6},
			3: {7, 8, 9},
		},
		numParts: 9,
	}

	b := expectedTB.marshal()

	tb := &transferredBundle{list: make(map[id.Round][]uint16)}

	tb.unmarshal(b)

	if !reflect.DeepEqual(expectedTB, tb) {
		t.Errorf("Failed to marshal and unmarshal transferredBundle into "+
			"original.\nexpected: %+v\nreceived: %+v", expectedTB, tb)
	}
}
