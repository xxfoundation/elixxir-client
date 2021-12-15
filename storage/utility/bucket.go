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
	bucketStoreVersion = 0
)

// BucketStore stores a leaky bucket into storage. The bucket
// is saved in a JSON-able format, where the key is used as an identifier
// of the bucket for ekv purposes.
type BucketStore struct {
	bucket *rateLimiting.Bucket

	// key represents the identity of what the bucket is tracking, and
	// is used as the kv's value on a save()
	key string
	kv  *versioned.KV
	mux sync.Mutex
}

// NewBucketStore creates a new, empty BucketStore and saves it to storage.
// If the primary method of modifying your BucketStore.bucket is via the method
// BucketStore.AddWithExternalParams, then the params argument may be
// default or junk data.
func NewBucketStore(params *rateLimiting.BucketParams, key string,
	kv *versioned.KV) (*BucketStore, error) {
	bs := &BucketStore{
		bucket: rateLimiting.CreateBucketFromParams(params, nil),
		kv:     kv.Prefix(bucketStorePrefix),
		key:    key,
		mux:    sync.Mutex{},
	}

	return bs, bs.save()
}

// Add adds the specified number of tokens to the bucket. If an add is
// unsuccessful, an error is returned. Else the bucket is saved to storage.
func (s *BucketStore) Add(tokens uint32) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	success, _ := s.bucket.Add(tokens)
	if !success {
		return errors.Errorf("Failed to add tokens %d "+
			"to bucket %s", tokens, s.key)
	}

	return s.save()
}

// AddWithExternalParams adds the specified number of tokens to the bucket
// given external bucket parameters rather than the params specified in
// the bucket. If an add is unsuccessful, an error is returned.
// Else the bucket is saved to storage.
func (s *BucketStore) AddWithExternalParams(tokens,
	capacity, leakedTokens uint32,
	duration time.Duration) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	success, _ := s.bucket.
		AddWithExternalParams(tokens, capacity, leakedTokens, duration)
	if !success {
		return errors.Errorf("Failed to AddWithExternalParams "+
			"(tokens %d, capacity %d, leakedTokens %d) to bucket %s",
			tokens, capacity, leakedTokens, s.key)
	}

	return s.save()
}

// LoadBucketStore is a storage operation which loads a bucket from storage
// given the key identifier.
func LoadBucketStore(params *rateLimiting.BucketParams, key string,
	kv *versioned.KV) (*BucketStore, error) {
	bs := &BucketStore{
		bucket: rateLimiting.CreateBucketFromParams(params, nil),
		key:    key,
		kv:     kv.Prefix(bucketStorePrefix),
		mux:    sync.Mutex{},
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

	return s.kv.Set(s.key, bucketStoreVersion, &obj)
}

// load is a helper function which extracts the bucket data from storage
// and loads it back into BucketStore.
func (s *BucketStore) load() error {
	// Load the versioned object
	vo, err := s.kv.Get(s.key, bucketStoreVersion)
	if err != nil {
		return err
	}

	return s.bucket.UnmarshalJSON(vo.Data)

}
