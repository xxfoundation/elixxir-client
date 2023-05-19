////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"encoding/json"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/rateLimiting"
)

const (
	bucketStorePrefix  = "bucketStore"
	bucketStoreKey     = "bucketStoreKey"
	bucketStoreVersion = 0
)

// BucketStore stores a leaky bucket into storage. The bucket
// is saved in a JSON-able format.
type BucketStore struct {
	kv versioned.KV
}

// bucketDisk is a JSON-able structure used to store
// a rateLimiting.Bucket parameters.
type bucketDisk struct {
	capacity  uint32
	timestamp int64
}

// NewStoredBucket creates a new, empty Bucket and saves it to storage.
func NewStoredBucket(capacity, leaked uint32, leakDuration time.Duration,
	kv versioned.KV) *rateLimiting.Bucket {
	kv, err := kv.Prefix(bucketStorePrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to prefix KV with %s: %+v", bucketStorePrefix, err)
	}
	bs := &BucketStore{
		kv: kv,
	}

	bs.save(0, netTime.Now().UnixNano())

	return rateLimiting.CreateBucket(capacity, leaked, leakDuration, bs.save)
}

// save stores the buckets values into storage.
func (s *BucketStore) save(inBucket uint32, timestamp int64) {

	// Create
	bd := bucketDisk{
		capacity:  inBucket,
		timestamp: timestamp,
	}

	data, err := json.Marshal(&bd)
	if err != nil {
		jww.ERROR.Printf("Failed to marshal %s bucket data for"+
			" storage: %v", s.kv.GetPrefix(), err)
	}

	obj := versioned.Object{
		Version:   bucketStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = s.kv.Set(bucketStoreKey, &obj)

	if err != nil {
		jww.ERROR.Printf("Failed to store %s bucket data: %v",
			s.kv.GetPrefix(), err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadBucket is a storage operation which loads a bucket from storage.
func LoadBucket(capacity, leaked uint32, leakDuration time.Duration,
	kv versioned.KV) (*rateLimiting.Bucket, error) {
	kv, err := kv.Prefix(bucketStorePrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to prefix KV with %s: %+v", bucketStorePrefix, err)
	}
	bs := &BucketStore{
		kv: kv,
	}
	inBucket, ts, err := bs.load()
	if err != nil {
		return nil, err
	}

	return rateLimiting.CreateBucketFromDB(capacity,
		leaked, leakDuration, inBucket, ts, bs.save), nil
}

// load is a helper function which extracts the bucket data from storage
// and loads it back into BucketStore.
func (s *BucketStore) load() (uint32, int64, error) {
	// Load the versioned object
	vo, err := s.kv.Get(bucketStoreKey, bucketStoreVersion)
	if err != nil {
		return 0, 0, err
	}

	bd := bucketDisk{}

	err = json.Unmarshal(vo.Data, &bd)
	if err != nil {
		return 0, 0, err
	}

	return bd.capacity, bd.timestamp, err

}
