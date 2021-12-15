///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"sync"
	"time"
)

const (
	bucketStorePrefix  = "bucketStore"
	bucketStoreKey     = "bucketStoreKey"
	bucketStoreVersion = 0
)

// BucketStore stores a leaky bucket into storage. The bucket
// is saved in a JSON-able format.
type BucketStore struct {
	bucket *rateLimiting.Bucket
	params *rateLimiting.MapParams
	kv     *versioned.KV
	mux    sync.RWMutex
}

// NewBucketStore creates a new, empty BucketStore and saves it to storage.
// If the primary method of modifying your BucketStore.bucket is via the method
// BucketStore.AddWithExternalParams, then the params argument may be
// default or junk data.
func NewBucketStore(params *rateLimiting.BucketParams, kv *versioned.KV) (*BucketStore, error) {
	bs := &BucketStore{
		bucket: rateLimiting.CreateBucketFromParams(params, nil),
		kv:     kv.Prefix(bucketStorePrefix),
		mux:    sync.RWMutex{},
	}

	return bs, bs.save()
}

// AddWithExternalParams adds the specified number of tokens to the bucket
// given external bucket parameters rather than the params specified in
// the bucket. If an add is unsuccessful, an error is returned.
// Else the bucket is saved to storage.
func (s *BucketStore) AddWithExternalParams(tokens uint32) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	success, _ := s.bucket.AddWithExternalParams(tokens,
		s.params.Capacity, s.params.LeakedTokens,
		s.params.LeakDuration)
	if err := s.save(); err != nil {
		return errors.WithMessagef(err, "Failed to save")
	}

	if !success {
		return errors.Errorf("Failed to add to bucket")
	}

	return nil
}

func (s *BucketStore) UpdateParams(capacity, leakedTokens uint32,
	leakDuration time.Duration) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.params = &rateLimiting.MapParams{
		Capacity:     capacity,
		LeakedTokens: leakedTokens,
		LeakDuration: leakDuration,
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadBucketStore is a storage operation which loads a bucket from storage.
func LoadBucketStore(params *rateLimiting.BucketParams,
	kv *versioned.KV) (*BucketStore, error) {
	bs := &BucketStore{
		bucket: rateLimiting.CreateBucketFromParams(params, nil),
		kv:     kv.Prefix(bucketStorePrefix),
		mux:    sync.RWMutex{},
	}

	return bs, bs.load()

}

// save is a non-thread-safe method of saving the bucket to storage. It is
// the responsibility of the caller to hold the lock for BucketStore.
func (s *BucketStore) save() error {
	data, err := s.bucket.MarshalJSON()
	if err != nil {
		return errors.Errorf("Failed to marshal bucket: %v", err)
	}

	obj := versioned.Object{
		Version:   bucketStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return s.kv.Set(bucketStoreKey, bucketStoreVersion, &obj)
}

// load is a helper function which extracts the bucket data from storage
// and loads it back into BucketStore.
func (s *BucketStore) load() error {
	// Load the versioned object
	vo, err := s.kv.Get(bucketStoreKey, bucketStoreVersion)
	if err != nil {
		return err
	}

	return s.bucket.UnmarshalJSON(vo.Data)

}
