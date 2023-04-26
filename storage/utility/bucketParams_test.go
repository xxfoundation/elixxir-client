////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"reflect"
	"testing"
	"time"
)

// todo: write tests

func TestNewBucketParamsStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	capacity, leakedTokens, leakDuration := uint32(10), uint32(11), time.Duration(12)
	bps, err := NewBucketParamsStore(capacity, leakedTokens, leakDuration, kv)
	if err != nil {
		t.Fatalf("NewBucketParamsStore error: %v", err)
	}

	newParams := bps.params
	if newParams.Capacity != capacity || newParams.LeakedTokens != leakedTokens ||
		newParams.LeakDuration != leakDuration {
		t.Fatalf("Unexpected values in BucketParamStore!"+
			"\n\tExpected params {capacity: %d, leakedToken %d, leakDuration: %d}"+
			"\n\tReceived params {capacity: %d, leakedToken %d, leakDuration: %d}",
			capacity, leakedTokens, leakDuration,
			newParams.Capacity, newParams.LeakedTokens, newParams.LeakDuration)
	}

	kv, err = kv.Prefix(bucketParamsPrefix)
	require.NoError(t, err)

	vo, err := kv.Get(bucketParamsKey, bucketParamsVersion)
	if err != nil {
		t.Fatalf("Failed to load from KV: %v", err)
	}

	loadedParams := unmarshalBucketParams(vo.Data)

	if !reflect.DeepEqual(newParams, loadedParams) {
		t.Fatalf("Loaded params from store does not match initialized values."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", newParams, loadedParams)
	}
}

func TestLoadBucketParamsStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	capacity, leakedTokens, leakDuration := uint32(10), uint32(11), time.Duration(12)
	bps, err := NewBucketParamsStore(capacity, leakedTokens, leakDuration, kv)
	if err != nil {
		t.Fatalf("NewBucketParamsStore error: %v", err)
	}

	loadedBps, err := LoadBucketParamsStore(kv)
	if err != nil {
		t.Fatalf("LoadBucketParamsStore error: %v", err)
	}

	if !reflect.DeepEqual(loadedBps, bps) {
		t.Fatalf("Loaded params from store does not match initialized values."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", bps, loadedBps)
	}
}
