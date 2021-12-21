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
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Tests that newPartStore produces the expected new partStore and that an empty
// parts list is saved to storage.
func Test_newPartStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)

	expectedPS := &partStore{
		parts:    make(map[uint16][]byte, numParts),
		numParts: numParts,
		kv:       kv,
	}

	ps, err := newPartStore(kv, numParts)
	if err != nil {
		t.Errorf("newPartStore returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedPS, ps) {
		t.Errorf("Returned incorrect partStore.\nexpected: %+v\nreceived: %+v",
			expectedPS, ps)
	}

	_, err = kv.Get(partsListKey, partsListVersion)
	if err != nil {
		t.Errorf("Failed to load part list from storage: %+v", err)
	}
}

// Tests that newPartStoreFromParts produces the expected partStore filled with
// the given parts.
func Test_newPartStoreFromParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)

	// Generate part slice and part map filled with the same data
	partSlice := make([][]byte, numParts)
	partMap := make(map[uint16][]byte, numParts)
	for i := uint16(0); i < numParts; i++ {
		b := make([]byte, 32)
		prng.Read(b)

		partSlice[i] = b
		partMap[i] = b
	}

	expectedPS := &partStore{
		parts:    partMap,
		numParts: uint16(len(partMap)),
		kv:       kv,
	}

	ps, err := newPartStoreFromParts(kv, partSlice...)
	if err != nil {
		t.Errorf("newPartStoreFromParts returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedPS, ps) {
		t.Errorf("Returned incorrect partStore.\nexpected: %+v\nreceived: %+v",
			expectedPS, ps)
	}

	loadedPS, err := loadPartStore(kv)
	if err != nil {
		t.Errorf("Failed to load partStore from storage: %+v", err)
	}

	if !reflect.DeepEqual(expectedPS, loadedPS) {
		t.Errorf("Loaded partStore does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedPS, loadedPS)
	}
}

// Tests that a part added via partStore.addPart can be loaded from memory and
// storage.
func Test_partStore_addPart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	expectedPart := []byte("part data")
	partNum := uint16(17)

	err := ps.addPart(expectedPart, partNum)
	if err != nil {
		t.Errorf("addPart returned an error: %+v", err)
	}

	// Check if part is in memory
	part, exists := ps.parts[partNum]
	if !exists || !bytes.Equal(expectedPart, part) {
		t.Errorf("Failed to get part #%d from memory."+
			"\nexpected: %+v\nreceived: %+v", partNum, expectedPart, part)
	}

	// Load part from storage
	vo, err := kv.Get(makePartsKey(partNum), partsStoreVersion)
	if err != nil {
		t.Errorf("Failed to load part from storage: %+v", err)
	}

	// Check that the part loaded from storage is correct
	if !bytes.Equal(expectedPart, vo.Data) {
		t.Errorf("Part saved to storage unexpected"+
			"\nexpected: %+v\nreceived: %+v", expectedPart, vo.Data)
	}
}

// Tests that partStore.getPart returns the expected part.
func Test_partStore_getPart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	expectedPart := []byte("part data")
	partNum := uint16(17)

	err := ps.addPart(expectedPart, partNum)
	if err != nil {
		t.Errorf("addPart returned an error: %+v", err)
	}

	// Check if part is in memory
	part, exists := ps.getPart(partNum)
	if !exists || !bytes.Equal(expectedPart, part) {
		t.Errorf("Failed to get part #%d from memory."+
			"\nexpected: %+v\nreceived: %+v", partNum, expectedPart, part)
	}
}

// Tests that partStore.getFile returns all parts concatenated in order.
func Test_partStore_getFile(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	ps, expectedFile := newRandomPartStore(16, kv, prng, t)

	// Pull data from file
	receivedFile, missingParts := ps.getFile()
	if missingParts != 0 {
		t.Errorf("File has missing parts.\nexpected: %d\nreceived: %d",
			0, missingParts)
	}

	// Check correctness of reconstructions
	if !bytes.Equal(receivedFile, expectedFile) {
		t.Fatalf("Full reconstructed file does not match expected."+
			"\nexpected: %v\nreceived: %v", expectedFile, receivedFile)
	}
}

// Tests that partStore.getFile returns all parts concatenated in order.
func Test_partStore_getFile_MissingPartsError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	// Delete half the parts
	for partNum := uint16(0); partNum < numParts; partNum++ {
		if partNum%2 == 0 {
			delete(ps.parts, partNum)
		}
	}

	// Pull data from file
	_, missingParts := ps.getFile()
	if missingParts != 8 {
		t.Errorf("Missing incorrect number of parts."+
			"\nexpected: %d\nreceived: %d", 0, missingParts)
	}
}

// Tests that partStore.len returns 0 for a new map and the correct value when
// parts are added
func Test_partStore_len(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newPartStore(kv, numParts)

	if ps.len() != 0 {
		t.Errorf("Length of new partStore is not 0."+
			"\nexpected: %d\nreceived: %d", 0, ps.len())
	}

	addedParts := 5
	for i := 0; i < addedParts; i++ {
		_ = ps.addPart([]byte("test"), uint16(i))
	}

	if ps.len() != addedParts {
		t.Errorf("Length of new partStore incorrect."+
			"\nexpected: %d\nreceived: %d", addedParts, ps.len())
	}
}

// Tests that loadPartStore gets the expected partStore from storage.
func Test_loadPartStore(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedPS, _ := newRandomPartStore(16, kv, prng, t)

	err := expectedPS.save()
	if err != nil {
		t.Errorf("Failed to save parts to storage: %+v", err)
	}

	ps, err := loadPartStore(kv)
	if err != nil {
		t.Errorf("loadPartStore returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedPS, ps) {
		t.Errorf("Loaded partStore does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedPS, ps)
	}
}

// Error path: tests that loadPartStore returns the expected error when no part
// list is saved to storage.
func Test_loadPartStore_NoSavedListErr(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadPartListErr, ":")[0]
	_, err := loadPartStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadPartStore failed to return the expected error when no "+
			"object is saved in storage.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that loadPartStore returns the expected error when no parts
// are saved to storage.
func Test_loadPartStore_NoPartErr(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadPartsErr, "#")[0]
	ps, _ := newRandomPartStore(16, kv, prng, t)

	err := ps.saveList()
	if err != nil {
		t.Errorf("Failed to save part list to storage: %+v", err)
	}

	_, err = loadPartStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadPartStore failed to return the expected error when no "+
			"object is saved in storage.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that partStore.save correctly stores the part list to storage by
// reading it from storage and ensuring all part numbers are present. It then
// makes sure that all the parts are stored in storage.
func Test_partStore_save(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))

	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	err := ps.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	vo, err := kv.Get(partsListKey, partsListVersion)
	if err != nil {
		t.Errorf("Failed to load part list from storage: %+v", err)
	}

	numList := make(map[uint16]bool, numParts)
	for i := uint16(0); i < numParts; i++ {
		numList[i] = true
	}

	buff := bytes.NewBuffer(vo.Data[2:])
	for next := buff.Next(2); len(next) == 2; next = buff.Next(2) {
		partNum := binary.LittleEndian.Uint16(next)
		if !numList[partNum] {
			t.Errorf("Part number %d is not a correct part number.", partNum)
		} else {
			delete(numList, partNum)
		}
	}

	if len(numList) != 0 {
		t.Errorf("File part numbers missing from list: %+v", numList)
	}

	loadedNumParts, list := unmarshalPartList(vo.Data)

	if numParts != loadedNumParts {
		t.Errorf("Loaded numParts does not match expected."+
			"\nexpected: %d\nrecieved: %d", numParts, loadedNumParts)
	}

	for _, partNum := range list {
		vo, err := kv.Get(makePartsKey(partNum), partsStoreVersion)
		if err != nil {
			t.Errorf("Failed to load part #%d from storage: %+v", partNum, err)
		}

		if !bytes.Equal(ps.parts[partNum], vo.Data) {
			t.Errorf("Part data #%d loaded from storage unexpected."+
				"\nexpected: %+v\nreceived: %+v",
				partNum, ps.parts[partNum], vo.Data)
		}
	}
}

// Tests that partStore.saveList saves the expected list to storage.
func Test_partStore_saveList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	expectedPS, _ := newRandomPartStore(numParts, kv, prng, t)

	err := expectedPS.saveList()
	if err != nil {
		t.Errorf("saveList returned an error: %+v", err)
	}

	vo, err := kv.Get(partsListKey, partsListVersion)
	if err != nil {
		t.Errorf("Failed to load part list from storage: %+v", err)
	}

	numList := make(map[uint16]bool, numParts)
	for i := uint16(0); i < numParts; i++ {
		numList[i] = true
	}

	buff := bytes.NewBuffer(vo.Data[2:])
	for next := buff.Next(2); len(next) == 2; next = buff.Next(2) {
		partNum := binary.LittleEndian.Uint16(next)
		if !numList[partNum] {
			t.Errorf("Part number %d is not a correct part number.", partNum)
		} else {
			delete(numList, partNum)
		}
	}

	if len(numList) != 0 {
		t.Errorf("File part numbers missing from list: %+v", numList)
	}
}

// Tests that a part saved via partStore.savePart can be loaded from storage.
func Test_partStore_savePart(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	ps, err := newPartStore(kv, 16)
	if err != nil {
		t.Fatalf("Failed to create new partStore: %+v", err)
	}

	expectedPart := []byte("part data")
	partNum := uint16(5)
	ps.parts[partNum] = expectedPart

	err = ps.savePart(partNum)
	if err != nil {
		t.Errorf("savePart returned an error: %+v", err)
	}

	// Load part from storage
	vo, err := kv.Get(makePartsKey(partNum), partsStoreVersion)
	if err != nil {
		t.Errorf("Failed to load part from storage: %+v", err)
	}

	// Check that the part loaded from storage is correct
	if !bytes.Equal(expectedPart, vo.Data) {
		t.Errorf("Part saved to storage unexpected"+
			"\nexpected: %+v\nreceived: %+v", expectedPart, vo.Data)
	}
}

// Tests that partStore.delete deletes all stored items.
func Test_partStore_delete(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	err := ps.delete()
	if err != nil {
		t.Errorf("delete returned an error: %+v", err)
	}

	_, err = kv.Get(partsListKey, partsListVersion)
	if err == nil {
		t.Error("Able to load part list from storage when it should have been " +
			"deleted.")
	}

	for partNum := range ps.parts {
		_, err := kv.Get(makePartsKey(partNum), partsStoreVersion)
		if err == nil {
			t.Errorf("Loaded part #%d from storage when it should have been "+
				"deleted.", partNum)
		}
	}
}

// Tests that a list marshalled via partStore.marshalList can be unmarshalled
// with unmarshalPartList to get the original list.
func Test_partStore_marshalList_unmarshalPartList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(make(ekv.Memstore))
	numParts := uint16(16)
	ps, _ := newRandomPartStore(numParts, kv, prng, t)

	expected := make([]uint16, 0, 16)
	for partNum := range ps.parts {
		expected = append(expected, partNum)
	}

	byteList := ps.marshalList()

	loadedNumParts, list := unmarshalPartList(byteList)

	if numParts != loadedNumParts {
		t.Errorf("Loaded numParts does not match expected."+
			"\nexpected: %d\nrecieved: %d", numParts, loadedNumParts)
	}

	sort.SliceStable(list, func(i, j int) bool { return list[i] < list[j] })
	sort.SliceStable(expected,
		func(i, j int) bool { return expected[i] < expected[j] })

	if !reflect.DeepEqual(expected, list) {
		t.Errorf("Failed to marshal and unmarshal part list."+
			"\nexpected: %+v\nreceived: %+v", expected, list)
	}
}

// Tests that partSliceToMap correctly maps all the parts in a slice of parts
// to a map of parts keyed on their part number.
func Test_partSliceToMap(t *testing.T) {
	prng := rand.New(rand.NewSource(42))

	// Create list of file parts with random data
	partSlice := make([][]byte, 8)
	for i := range partSlice {
		partSlice[i] = make([]byte, 16)
		prng.Read(partSlice[i])
	}

	// Convert the slice of parts to a map
	partMap := partSliceToMap(partSlice...)

	// Check that each part in the map matches a part in the slice
	for partNum, slicePart := range partSlice {
		// Check that the part exists in the map
		mapPart, exists := partMap[uint16(partNum)]
		if !exists {
			t.Errorf("Part number %d does not exist in the map.", partNum)
		}

		// Check that the part in the map is correct
		if !bytes.Equal(slicePart, mapPart) {
			t.Errorf("Part found in map does not match expected part in slice."+
				"\nexpected: %+v\nreceived: %+v", slicePart, mapPart)
		}

		delete(partMap, uint16(partNum))
	}

	// Make sure there are no extra parts in the map
	if len(partMap) != 0 {
		t.Errorf("Part map contains %d extra parts not in the slice."+
			"\nparts: %+v", len(partMap), partMap)
	}
}

// Tests the consistency of makePartsKey.
func Test_makePartsKey_consistency(t *testing.T) {
	expectedStrings := []string{
		partsStoreKey + "0",
		partsStoreKey + "1",
		partsStoreKey + "2",
		partsStoreKey + "3",
	}

	for i, expected := range expectedStrings {
		key := makePartsKey(uint16(i))
		if key != expected {
			t.Errorf("Key #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, key)
		}
	}
}

// newRandomPartStore creates a new partStore filled with random data. Returns
// the random partStore and a slice of the original file.
func newRandomPartStore(numParts uint16, kv *versioned.KV, prng io.Reader,
	t *testing.T) (*partStore, []byte) {

	partSize := 64

	ps, err := newPartStore(kv, numParts)
	if err != nil {
		t.Fatalf("Failed to create new partStore: %+v", err)
	}

	fileBuff := bytes.NewBuffer(nil)
	fileBuff.Grow(int(numParts) * partSize)

	for partNum := uint16(0); partNum < numParts; partNum++ {
		ps.parts[partNum] = make([]byte, partSize)
		_, err := prng.Read(ps.parts[partNum])
		if err != nil {
			t.Errorf("Failed to generate random part (%d): %+v", partNum, err)
		}
		fileBuff.Write(ps.parts[partNum])
	}

	return ps, fileBuff.Bytes()
}

// newRandomPartSlice returns a list of file parts and the file in one piece.
func newRandomPartSlice(numParts uint16, prng io.Reader, t *testing.T) (
	[][]byte, []byte) {
	partSize := 64
	fileBuff := bytes.NewBuffer(make([]byte, 0, int(numParts)*partSize))
	partList := make([][]byte, numParts)
	for partNum := range partList {
		partList[partNum] = make([]byte, partSize)
		_, err := prng.Read(partList[partNum])
		if err != nil {
			t.Errorf("Failed to generate random part (%d): %+v", partNum, err)
		}
		fileBuff.Write(partList[partNum])
	}

	return partList, fileBuff.Bytes()
}
