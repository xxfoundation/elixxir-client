///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"reflect"
	"testing"
	"time"
)

// NewBucketStore happy path.
func TestNewBucketStore(t *testing.T) {
	// Create initializers
	params := &rateLimiting.BucketParams{
		Capacity:   10,
		Remaining:  11,
		LeakRate:   12,
		LastUpdate: 13,
	}
	key := "1.2.3.4"
	kv := versioned.NewKV(make(ekv.Memstore))

	// Initialize bucket
	bs, err := NewBucketStore(params, key, kv)
	if err != nil {
		t.Fatalf("Error received creating BucketStore: %v", err)
	}

	// Load new bucket from storage
	vo, err := bs.kv.Get(key, bucketStoreVersion)
	if err != nil {
		t.Fatalf("Failed to get bucket from EKV: %v", err)
	}

	// Unmarshal stored data into separate bucket
	receivedBs := rateLimiting.CreateBucketFromParams(params, nil)
	err = receivedBs.UnmarshalJSON(vo.Data)
	if err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	// Ensure bucket in RAM and bucket in storage are consistent
	if !reflect.DeepEqual(receivedBs, bs.bucket) {
		t.Fatalf("Loaded bucket and created bucket did not match."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v ", bs.bucket, receivedBs)
	}

}

// BucketStore.AddWithExternalParams happy path.
func TestBucketStore_AddWithExternalParams(t *testing.T) {
	// Create initializers
	params := &rateLimiting.BucketParams{
		Capacity:   10,
		Remaining:  11,
		LeakRate:   12,
		LastUpdate: 13,
	}
	key := "1.2.3.4"
	kv := versioned.NewKV(make(ekv.Memstore))

	// Create bucket
	bs, err := NewBucketStore(params, key, kv)
	if err != nil {
		t.Fatalf("Error received creating BucketStore: %v", err)
	}

	// Modify internal state of bucket
	err = bs.AddWithExternalParams(10, 11, 12, 14*time.Second)
	if err != nil {
		t.Fatalf("AddWithExternalParams error: %v", err)
	}

	// Load stored bucket
	vo, err := bs.kv.Get(key, bucketStoreVersion)
	if err != nil {
		t.Fatalf("Failed to get bucket from EKV: %v", err)
	}

	// Unmarshal stored data into separate bucket
	receivedBs := rateLimiting.CreateBucketFromParams(params, nil)
	err = receivedBs.UnmarshalJSON(vo.Data)
	if err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	// Ensure bucket in RAM and bucket in storage are consistent
	if !reflect.DeepEqual(receivedBs, bs.bucket) {
		t.Fatalf("Loaded bucket and created bucket did not match."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v ", bs.bucket, receivedBs)
	}

}

// LoadBucketStore happy path.
func TestLoadBucketStore(t *testing.T) {
	// Create initializers
	params := &rateLimiting.BucketParams{
		Capacity:   10,
		Remaining:  11,
		LeakRate:   12,
		LastUpdate: 13,
	}
	key := "1.2.3.4"
	kv := versioned.NewKV(make(ekv.Memstore))

	// Create bucket
	bs, err := NewBucketStore(params, key, kv)
	if err != nil {
		t.Fatalf("Error received creating BucketStore: %v", err)
	}

	receivedBs, err := LoadBucketStore(params, key, kv)
	if err != nil {
		t.Fatalf("LoadBucketStore error: %v", err)
	}

	// Ensure bucket in RAM and bucket in storage are consistent
	if !reflect.DeepEqual(receivedBs, bs) {
		t.Fatalf("Loaded bucket store and created bucket did not match."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v ", bs, receivedBs)
	}
}

// BucketStore.Add happy path.
func TestBucketStore_Add(t *testing.T) {
	// Create initializers
	params := &rateLimiting.BucketParams{
		Capacity:   10,
		Remaining:  11,
		LeakRate:   12,
		LastUpdate: time.Now().UnixNano(),
	}
	key := "1.2.3.4"
	kv := versioned.NewKV(make(ekv.Memstore))

	// Create bucket
	bs, err := NewBucketStore(params, key, kv)
	if err != nil {
		t.Fatalf("Error received creating BucketStore: %v", err)
	}

	// Modify internal state of bucket
	err = bs.Add(1)
	if err != nil {
		t.Fatalf("AddWithExternalParams error: %v", err)
	}

	// Load stored bucket
	vo, err := bs.kv.Get(key, bucketStoreVersion)
	if err != nil {
		t.Fatalf("Failed to get bucket from EKV: %v", err)
	}

	// Unmarshal stored data into separate bucket
	receivedBs := rateLimiting.CreateBucketFromParams(params, nil)
	err = receivedBs.UnmarshalJSON(vo.Data)
	if err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	// Ensure bucket in RAM and bucket in storage are consistent
	if !reflect.DeepEqual(receivedBs, bs.bucket) {
		t.Fatalf("Loaded bucket and created bucket did not match."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v ", bs.bucket, receivedBs)
	}
}
