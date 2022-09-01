package utility

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

type iter [][]byte

func (it *iter) Next() ([]byte, bool) {
	if len(*it) > 0 {
		item := (*it)[0]
		*it = (*it)[1:]
		return item, true
	}

	return nil, false
}

// Happy path.
func TestNewBlockStore(t *testing.T) {
	expected := &BlockStore{
		block:      make([][]byte, 0, 20),
		numBlocks:  50,
		blockSize:  20,
		firstSaved: 0,
		lastSaved:  0,
		kv:         versioned.NewKV(ekv.MakeMemstore()),
	}

	bs, err := NewBlockStore(expected.numBlocks, expected.blockSize, expected.kv)
	if err != nil {
		t.Errorf("NewBlockStore() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, bs) {
		t.Errorf("NewBlockStore() did not return the expected BlockStore."+
			"\nexpected: %+v\nreceived: %+v", expected, bs)
	}
}

// Tests BlockStore storing and loading data in multiple situations.
func TestBlockStore_Store_LoadBlockStore(t *testing.T) {
	values := []struct {
		blockSize, numBlocks                  int
		expectedFirstSaved, expectedLastSaved int
		dataCutIndex                          int
	}{
		{3, 5, 0, 3, 0},  // Multiple blocks, last block partial, no pruning
		{10, 5, 0, 0, 0}, // Single block, last block full, no pruning
		{15, 5, 0, 0, 0}, // Single block, last block partial, no pruning

		{2, 3, 2, 4, 4}, // Multiple blocks, last block partial, pruned
		{5, 1, 1, 1, 5}, // Single block, last block full, pruned
		{4, 1, 2, 2, 8}, // Single block, last block partial, pruned
	}

	for i, v := range values {
		// Create the initial data to store
		iter := make(iter, 10)
		for i := uint64(0); i < 10; i++ {
			iter[i] = make([]byte, 8)
			binary.LittleEndian.PutUint64(iter[i], i)
		}

		// Calculate the expected data
		expected := make([][]byte, len(iter[v.dataCutIndex:]))
		copy(expected, iter[v.dataCutIndex:])

		bs, err := NewBlockStore(v.numBlocks, v.blockSize, versioned.NewKV(ekv.MakeMemstore()))
		if err != nil {
			t.Errorf("Failed to create new BlockStore (%d): %+v", i, err)
		}

		// Attempt to store the data
		err = bs.Store(&iter)
		if err != nil {
			t.Errorf("Store() returned an error (%d): %+v", i, err)
		}

		if bs.firstSaved != v.expectedFirstSaved {
			t.Errorf("Store() did not return the expected firstSaved (%d)."+
				"\nexpected: %d\nreceived: %d", i, v.expectedFirstSaved, bs.firstSaved)
		}

		if bs.lastSaved != v.expectedLastSaved {
			t.Errorf("Store() did not return the expected lastSaved (%d)."+
				"\nexpected: %d\nreceived: %d", i, v.expectedLastSaved, bs.lastSaved)
		}

		// Attempt to load the data
		loadBS, data, err := LoadBlockStore(bs.kv)
		if err != nil {
			t.Errorf("LoadBlockStore() returned an error (%d): %+v", i, err)
		}

		// Check if the loaded BlockStore is correct
		if !reflect.DeepEqual(bs, loadBS) {
			t.Errorf("Loading wrong BlockStore from storage (%d)."+
				"\nexpected: %+v\nreceived: %+v", i, bs, loadBS)
		}

		// Check if the loaded data is correct
		if !reflect.DeepEqual(expected, data) {
			t.Errorf("Loading wrong data from storage (%d)."+
				"\nexpected: %+v\nreceived: %+v", i, expected, data)
		}
	}
}

// Tests that a block is successfully saved and loaded from storage.
func TestBlockStore_saveBlock_loadBlock(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	bs := &BlockStore{
		block:      make([][]byte, 0, 20),
		numBlocks:  50,
		blockSize:  20,
		firstSaved: 0,
		lastSaved:  0,
		kv:         versioned.NewKV(ekv.MakeMemstore()),
	}

	for i := range bs.block {
		bs.block[i] = make([]byte, 32)
		prng.Read(bs.block[i])
	}

	err := bs.saveBlock()
	if err != nil {
		t.Errorf("saveBlock() returned an error: %+v", err)
	}

	newBS := &BlockStore{kv: bs.kv}
	block, err := newBS.loadBlock(0)
	if err != nil {
		t.Errorf("loadBlock() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(bs.block, block) {
		t.Errorf("Failed to save and load block to storage."+
			"\nexpected: %+v\nreceived: %+v", bs.block, block)
	}
}

// Error path: failed to save to KV.
func TestBlockStore_saveBlock_SaveError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	bs := &BlockStore{
		block:      make([][]byte, 0, 20),
		numBlocks:  50,
		blockSize:  20,
		firstSaved: 0,
		lastSaved:  0,
		kv:         versioned.NewKV(ekv.MakeMemstore()),
	}

	for i := range bs.block {
		bs.block[i] = make([]byte, 32)
		prng.Read(bs.block[i])
	}

	err := bs.saveBlock()
	if err != nil {
		t.Errorf("saveBlock() returned an error: %+v", err)
	}

	newBS := &BlockStore{kv: bs.kv}
	block, err := newBS.loadBlock(0)
	if err != nil {
		t.Errorf("loadBlock() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(bs.block, block) {
		t.Errorf("Failed to save and load block to storage."+
			"\nexpected: %+v\nreceived: %+v", bs.block, block)
	}
}

// Error path: loading of nonexistent key returns an error.
func TestBlockStore_loadBlock_LoadStorageError(t *testing.T) {
	expectedErr := strings.SplitN(bKvLoadErr, "%", 2)[0]
	bs := &BlockStore{kv: versioned.NewKV(ekv.MakeMemstore())}
	_, err := bs.loadBlock(0)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadBlock() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: unmarshalling of invalid data fails.
func TestBlockStore_loadBlock_UnmarshalError(t *testing.T) {
	bs := &BlockStore{kv: versioned.NewKV(ekv.MakeMemstore())}
	expectedErr := strings.SplitN(bJsonUnmarshalErr, "%", 2)[0]

	// Construct object with invalid data
	obj := versioned.Object{
		Version:   blockVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("invalid JSON"),
	}

	// Save to storage
	err := bs.kv.Set(bs.getKey(bs.lastSaved), &obj)
	if err != nil {
		t.Errorf("Failed to save data to KV: %+v", err)
	}

	_, err = bs.loadBlock(0)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadBlock() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Happy path.
func TestBlockStore_pruneBlocks(t *testing.T) {
	bs := &BlockStore{
		block:      make([][]byte, 0, 32),
		numBlocks:  5,
		blockSize:  32,
		firstSaved: 0,
		lastSaved:  0,
		kv:         versioned.NewKV(ekv.MakeMemstore()),
	}

	// Save blocks to storage
	for ; bs.lastSaved < 15; bs.lastSaved++ {
		if err := bs.saveBlock(); err != nil {
			t.Errorf("Failed to save block %d: %+v", bs.lastSaved, err)
		}
	}

	// Calculate the new first saved index
	oldFirstSaved := bs.firstSaved
	bs.firstSaved = bs.lastSaved - bs.numBlocks

	// Prune blocks
	bs.pruneBlocks(oldFirstSaved)

	// Check that the old blocks were deleted
	for i := 0; i < bs.lastSaved-bs.numBlocks; i++ {
		if _, err := bs.kv.Get(bs.getKey(i), blockVersion); err == nil {
			t.Errorf("pruneBlocks() failed to delete old block %d: %+v", i, err)
		}
	}

	// Check that the new blocks were not deleted
	for i := bs.firstSaved; i < bs.lastSaved; i++ {
		if _, err := bs.kv.Get(bs.getKey(i), blockVersion); err != nil {
			t.Errorf("pruneBlocks() deleted block %d: %+v", i, err)
		}
	}

	// Call pruneBlocks when there are no blocks to prune
	oldFirstSaved = bs.firstSaved
	bs.firstSaved = bs.lastSaved - bs.numBlocks
	bs.pruneBlocks(oldFirstSaved)

	// Check that the new blocks were not deleted
	for i := bs.firstSaved; i < bs.lastSaved; i++ {
		if _, err := bs.kv.Get(bs.getKey(i), blockVersion); err != nil {
			t.Errorf("pruneBlocks() deleted block %d: %+v", i, err)
		}
	}
}

// Consistency test.
func TestBlockStore_getKey_Consistency(t *testing.T) {
	expectedKeys := []string{
		"block/0", "block/1", "block/2", "block/3", "block/4",
		"block/5", "block/6", "block/7", "block/8", "block/9",
	}
	var bs BlockStore

	for i, expected := range expectedKeys {
		key := bs.getKey(i)
		if key != expected {
			t.Errorf("getKey did not return the correct key for the index %d."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

// Tests that a BlockStore can be saved and loaded from the KV correctly.
func TestBlockStore_save_load(t *testing.T) {
	bs := &BlockStore{
		numBlocks: 5, blockSize: 6, firstSaved: 7, lastSaved: 8,
		kv: versioned.NewKV(ekv.MakeMemstore()),
	}

	err := bs.save()
	if err != nil {
		t.Errorf("save() returned an error: %+v", err)
	}

	testBS := &BlockStore{kv: bs.kv}
	err = testBS.load()
	if err != nil {
		t.Errorf("load() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(bs, testBS) {
		t.Errorf("Failed to save and load BlockStore to KV."+
			"\nexpected: %+v\nreceived: %+v", bs, testBS)
	}
}

// Error path: loading of unsaved BlockStore fails.
func TestBlockStore_load_KvGetError(t *testing.T) {
	expectedErr := strings.SplitN(bsKvLoadErr, "%", 2)[0]

	testBS := &BlockStore{kv: versioned.NewKV(ekv.MakeMemstore())}
	err := testBS.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("load() did not return an error for a nonexistent item in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: unmarshalling of invalid data fails.
func TestBlockStore_load_UnmarshalError(t *testing.T) {
	expectedErr := strings.SplitN(bsKvUnmarshalErr, "%", 2)[0]
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Construct invalid versioning object
	obj := versioned.Object{
		Version:   blockStoreVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("invalid data"),
	}

	// Save to storage
	err := kv.Set(blockStoreKey, &obj)
	if err != nil {
		t.Fatalf("failed to save object to storage: %+v", err)
	}

	// Try to retrieve invalid object
	testBS := &BlockStore{kv: kv}
	err = testBS.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("load() did not return an error for a nonexistent item in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Consistency test.
func TestBlockStore_unmarshal(t *testing.T) {
	buff := []byte{5, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0,
		0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0}
	expected := &BlockStore{numBlocks: 5, blockSize: 6, firstSaved: 7, lastSaved: 8}

	bs := &BlockStore{}
	err := bs.unmarshal(buff)
	if err != nil {
		t.Errorf("unmarshal() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, bs) {
		t.Errorf("unmarshal() did not return the expected BlockStore."+
			"\nexpected: %+v\nreceived: %+v", expected, bs)
	}
}

// Error path: length of buffer incorrect.
func TestBlockStore_unmarshal_BuffLengthError(t *testing.T) {
	expectedErr := fmt.Sprintf(bsBuffLengthErr, 0, marshalledSize)
	bs := BlockStore{}
	err := bs.unmarshal([]byte{})
	if err == nil || err.Error() != expectedErr {
		t.Errorf("unmarshal() did not return the expected error for a buffer "+
			"of the wrong size.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Consistency test.
func TestBlockStore_marshal(t *testing.T) {
	expected := []byte{5, 0, 0, 0, 0, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0,
		0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0}
	bs := &BlockStore{numBlocks: 5, blockSize: 6, firstSaved: 7, lastSaved: 8}

	buff := bs.marshal()

	if !bytes.Equal(expected, buff) {
		t.Errorf("marshal() did not return the expected bytes."+
			"\nexpected: %+v\nreceived: %+v", expected, buff)
	}
}

// Tests that marshal and unmarshal work together.
func TestBlockStore_marshal_unmarshal(t *testing.T) {
	bs := &BlockStore{numBlocks: 5, blockSize: 6, firstSaved: 7, lastSaved: 8}

	buff := bs.marshal()

	testBS := &BlockStore{}
	err := testBS.unmarshal(buff)
	if err != nil {
		t.Errorf("unmarshal() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(bs, testBS) {
		t.Errorf("failed to marshal and unmarshal BlockStore."+
			"\nexpected: %+v\nreceived: %+v", bs, testBS)
	}
}
