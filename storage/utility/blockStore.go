package utility

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
)

// Sizes in bytes
const (
	int64Size      = 8
	marshalledSize = 4 * int64Size
)

// Error messages
const (
	bsBuffLengthErr   = "length of buffer %d != %d expected"
	bsKvSaveErr       = "failed to save blockStore to KV: %+v"
	bsKvInitSaveErr   = "failed to save initial block: %+v"
	bsKvLoadErr       = "failed to get BlockStore from storage: %+v"
	bsKvUnmarshalErr  = "failed to unmarshal BlockStore loaded from storage: %+v"
	bJsonMarshalErr   = "failed to JSON marshal block %d: %+v"
	bKvSaveErr        = "failed to save block %d to KV: %+v"
	bKvDeleteErr      = "failed to delete block %d from KV: %+v"
	bKvLoadErr        = "failed to get block %d from KV: %+v"
	bJsonUnmarshalErr = "failed to JSON marshal block %d: %+v"
)

// Storage keys and parts
const (
	delimiter         = "/"
	blockStoreKey     = "blockStore"
	blockStoreVersion = 0
	blockKey          = "block"
	blockVersion      = 0
)

type Iterator interface {
	Next() ([]byte, bool)
}

type BlockStore struct {
	block      [][]byte
	numBlocks  int // The maximum number of blocks saved to the kv
	blockSize  int // The maximum number of items allowed in a block
	firstSaved int // The index of the oldest block in the list
	lastSaved  int // The index of the newest block in the list
	kv         *versioned.KV
}

// NewBlockStore returns a new BlockStore and saves it to storage.
func NewBlockStore(numBlocks, blockSize int, kv *versioned.KV) (*BlockStore, error) {
	bs := &BlockStore{
		block:      make([][]byte, 0, blockSize),
		numBlocks:  numBlocks,
		blockSize:  blockSize,
		firstSaved: 0,
		lastSaved:  0,
		kv:         kv,
	}

	return bs, bs.save()
}

// LoadBlockStore returns the BlockStore from storage and a concatenation of all
// blocks in storage.
func LoadBlockStore(kv *versioned.KV) (*BlockStore, [][]byte, error) {
	bs := &BlockStore{kv: kv}

	// get BlockStore parameters from storage
	err := bs.load()
	if err != nil {
		return nil, nil, err
	}

	// LoadBlockStore each block from storage and join together into single slice
	var data, block [][]byte
	for i := bs.firstSaved; i <= bs.lastSaved; i++ {
		// get the block from storage
		block, err = bs.loadBlock(i)
		if err != nil {
			return nil, nil, err
		}

		// Append block to the rest of the data
		data = append(data, block...)
	}

	// Save the last block into memory
	bs.block = block

	return bs, data, nil
}

// Store stores all items in the Iterator to storage in blocks.
func (bs *BlockStore) Store(iter Iterator) error {
	// Iterate through all items in the Iterator and add each to the block in
	// memory. When the block is full, it is saved to storage and a new block is
	// added to until the iterator returns false.
	for item, exists := iter.Next(); exists; item, exists = iter.Next() {
		// If the block is full, save it to storage and start a new block
		if len(bs.block) >= bs.blockSize {
			if err := bs.saveBlock(); err != nil {
				return err
			}

			bs.lastSaved++
			bs.block = make([][]byte, 0, bs.blockSize)
		}

		// Append the item to the block in memory
		bs.block = append(bs.block, item)
	}

	// Save the current partially filled block to storage
	if err := bs.saveBlock(); err != nil {
		return err
	}

	// Calculate the new first saved index
	oldFirstSaved := bs.firstSaved
	if (bs.lastSaved+1)-bs.firstSaved > bs.numBlocks {
		bs.firstSaved = (bs.lastSaved + 1) - bs.numBlocks
	}

	// Save the BlockStorage parameters to storage
	if err := bs.save(); err != nil {
		return err
	}

	// Delete all old blocks
	bs.pruneBlocks(oldFirstSaved)

	return nil
}

// saveBlock saves the block to an indexed storage.
func (bs *BlockStore) saveBlock() error {
	// JSON marshal block
	data, err := json.Marshal(bs.block)
	if err != nil {
		return errors.Errorf(bJsonMarshalErr, bs.lastSaved, err)
	}

	// Construct versioning object
	obj := versioned.Object{
		Version:   blockVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = bs.kv.Set(bs.getKey(bs.lastSaved), &obj)
	if err != nil {
		return errors.Errorf(bKvSaveErr, bs.lastSaved, err)
	}

	return nil
}

// loadBlock loads the block with the index from storage.
func (bs *BlockStore) loadBlock(i int) ([][]byte, error) {
	// get the data from the kv
	obj, err := bs.kv.Get(bs.getKey(i), blockVersion)
	if err != nil {
		return nil, errors.Errorf(bKvLoadErr, i, err)
	}

	// Unmarshal the block
	var block [][]byte
	err = json.Unmarshal(obj.Data, &block)
	if err != nil {
		return nil, errors.Errorf(bJsonUnmarshalErr, i, err)
	}

	return block, nil
}

// pruneBlocks reduces the number of saved blocks to numBlocks by removing the
// oldest blocks.
func (bs *BlockStore) pruneBlocks(firstSaved int) {
	// Exit if no blocks need to be pruned
	if (bs.lastSaved+1)-firstSaved < bs.numBlocks {
		return
	}

	// Delete all blocks before the firstSaved index
	for ; firstSaved < bs.firstSaved; firstSaved++ {
		err := bs.kv.Delete(bs.getKey(firstSaved), blockVersion)
		if err != nil {
			jww.WARN.Printf(bKvDeleteErr, bs.firstSaved, err)
		}
	}
}

// getKey produces a block storage key for the given index.
func (bs *BlockStore) getKey(i int) string {
	return blockKey + delimiter + strconv.Itoa(i)
}

// save saves the parameters in BlockStore to storage. It does not save any
// block data.
func (bs *BlockStore) save() error {
	// Construct versioning object
	obj := versioned.Object{
		Version:   blockStoreVersion,
		Timestamp: netTime.Now(),
		Data:      bs.marshal(),
	}

	// Save to storage
	err := bs.kv.Set(blockStoreKey, &obj)
	if err != nil {
		return errors.Errorf(bsKvSaveErr, err)
	}

	// Save initial block
	err = bs.saveBlock()
	if err != nil {
		return errors.Errorf(bsKvInitSaveErr, err)
	}

	return nil
}

// load loads BlockStore parameters from storage.
func (bs *BlockStore) load() error {
	// get the data from the kv
	obj, err := bs.kv.Get(blockStoreKey, blockStoreVersion)
	if err != nil {
		return errors.Errorf(bsKvLoadErr, err)
	}

	// Unmarshal the data into a BlockStore
	err = bs.unmarshal(obj.Data)
	if err != nil {
		return errors.Errorf(bsKvUnmarshalErr, err)
	}

	return nil
}

// marshal marshals the BlockStore integer values to a byte slice.
func (bs *BlockStore) marshal() []byte {
	// Build list of values to store
	values := []int{bs.numBlocks, bs.blockSize, bs.firstSaved, bs.lastSaved}

	// Convert each value to a byte slice and store
	var buff bytes.Buffer
	for _, val := range values {
		b := make([]byte, int64Size)
		binary.LittleEndian.PutUint64(b, uint64(val))
		buff.Write(b)
	}

	// Return the bytes
	return buff.Bytes()
}

// unmarshal unmarshalls the BlockStore int values from the buffer. An error is
// returned if the length of the bytes is incorrect.
func (bs *BlockStore) unmarshal(b []byte) error {
	// Return an error if the buffer is not the expected length
	if len(b) != marshalledSize {
		return errors.Errorf(bsBuffLengthErr, len(b), marshalledSize)
	}

	// Convert the byte slices to ints and store
	buff := bytes.NewBuffer(b)
	bs.numBlocks = int(binary.LittleEndian.Uint64(buff.Next(int64Size)))
	bs.blockSize = int(binary.LittleEndian.Uint64(buff.Next(int64Size)))
	bs.firstSaved = int(binary.LittleEndian.Uint64(buff.Next(int64Size)))
	bs.lastSaved = int(binary.LittleEndian.Uint64(buff.Next(int64Size)))

	return nil
}
