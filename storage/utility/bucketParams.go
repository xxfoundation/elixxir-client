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
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"sync"
	"time"
)

const (
	bucketParamsPrefix  = "bucketParamPrefix"
	bucketParamsKey     = "bucketParamKey"
	bucketParamsVersion = 0
)

// BucketParamStore is the storage object for bucket params. Updated via the
// network follower.
type BucketParamStore struct {
	params *rateLimiting.MapParams
	mux    sync.RWMutex
	kv     *versioned.KV
}

// NewBucketParamsStore is the constructor for a BucketParamStore.
func NewBucketParamsStore(capacity, leakedTokens uint32,
	leakDuration time.Duration, kv *versioned.KV) (*BucketParamStore, error) {

	kv, err := kv.Prefix(bucketParamsPrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to prefix KV with %s: %+v", bucketParamsPrefix, err)
	}

	bps := &BucketParamStore{
		params: &rateLimiting.MapParams{
			Capacity:     capacity,
			LeakedTokens: leakedTokens,
			LeakDuration: leakDuration,
		},
		mux: sync.RWMutex{},
		kv:  kv,
	}

	return bps, bps.save()
}

// Get reads and returns te bucket params.
func (s *BucketParamStore) Get() *rateLimiting.MapParams {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.params
}

// UpdateParams updates the parameters to store.
func (s *BucketParamStore) UpdateParams(capacity, leakedTokens uint32,
	leakDuration time.Duration) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.params = &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
	}

	return s.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadBucketParamsStore loads the bucket params data from storage and constructs
// a BucketParamStore.
func LoadBucketParamsStore(kv *versioned.KV) (*BucketParamStore, error) {

	kv, err := kv.Prefix(bucketParamsPrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to prefix KV with %s: %+v", bucketParamsPrefix, err)
	}
	bps := &BucketParamStore{
		params: &rateLimiting.MapParams{},
		mux:    sync.RWMutex{},
		kv:     kv,
	}

	return bps, bps.load()

}

// save stores the bucket params into storage.
func (s *BucketParamStore) save() error {

	// Initiate stored object
	object := &versioned.Object{
		Version:   bucketParamsVersion,
		Timestamp: netTime.Now(),
		Data:      s.marshal(),
	}

	// Store object into storage
	return s.kv.Set(bucketParamsKey, object)
}

// load extracts the bucket params from store and loads it into the
// BucketParamStore.
func (s *BucketParamStore) load() error {
	// Load params from KV
	vo, err := s.kv.Get(bucketParamsKey, bucketParamsVersion)
	if err != nil {
		return errors.Errorf("Failed to load from KV: %s", err.Error())
	}

	// Unmarshal bucket params
	loadedParams := unmarshalBucketParams(vo.Data)

	// Place params into RAM object
	s.params = loadedParams

	return nil

}

// marshal serializes the rateLimiting.MapParams into byte data.
func (s *BucketParamStore) marshal() []byte {
	buff := bytes.NewBuffer(nil)

	// Write capacity to buffer
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, s.params.Capacity)
	buff.Write(b)

	// Write leakedTokens to buffer
	b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, s.params.LeakedTokens)
	buff.Write(b)

	// Write leakDuration to buffer
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(s.params.LeakDuration.Nanoseconds()))
	buff.Write(b)

	return buff.Bytes()
}

// unmarshalBucketParams deserializes the bucket params
// into a rateLimiting.MapParams.
func unmarshalBucketParams(b []byte) *rateLimiting.MapParams {
	buff := bytes.NewBuffer(b)

	// Load capacity
	capacity := binary.LittleEndian.Uint32(buff.Next(4))

	// Load leakedTokens
	leakedTokents := binary.LittleEndian.Uint32(buff.Next(4))

	// Load leakDuration
	leakDuration := time.Duration(binary.LittleEndian.Uint32(buff.Next(8)))

	return &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokents,
		LeakDuration: leakDuration,
	}

}
