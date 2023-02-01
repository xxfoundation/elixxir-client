////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strconv"
	"sync"
	"testing"
)

const expectedHeaderJson = `{"version":0,"entries":{"key0":"val0","key1":"val1","key2":"val2","key3":"val3","key4":"val4","key5":"val5","key6":"val6","key7":"val7","key8":"val8","key9":"val9"}}`

// Unit test of NewHeader.
func TestNewHeader(t *testing.T) {
	receivedHeader := NewHeader()
	expectedHeader := &Header{
		version: headerVersion,
		entries: make(map[string]string, 0),
		mux:     sync.Mutex{},
	}
	require.Equal(t, expectedHeader, receivedHeader)
}

// Unit test of Header.Set.
func TestHeader_Set(t *testing.T) {
	// Initialize header object
	head := NewHeader()

	// Set key-value entry into header
	key, val := "key", "val"
	require.NoError(t, head.Set(key, val))

	// Ensure that key exists in map and is the expected value
	received, exists := head.entries[key]
	require.True(t, exists)
	require.Equal(t, val, received)
}

// Error test of Header.Set where Set is called with a duplicate key.
// Overwriting an entry should not occur.
func TestHeader_Set_Error(t *testing.T) {
	// Initialize header object
	head := NewHeader()

	// Set key-value entry into header
	key, originalVal, newVal := "key", "val", "newValFailure"
	require.NoError(t, head.Set(key, originalVal))

	// Attempt to overwrite key with new value
	require.Error(t, head.Set(key, newVal))

	// Ensure that key exists in map and is the expected value
	received, exists := head.entries[key]
	require.True(t, exists)
	require.Equal(t, originalVal, received)
}

// Smoke test for Header.MarshalJSON.
func TestHeader_MarshalJSON(t *testing.T) {
	// Initialize header object
	head := NewHeader()

	// Create multiple entries for JSON
	const numTests = 10
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), "val"+strconv.Itoa(i)
		require.NoError(t, head.Set(key, val))
	}

	// Marshal header
	marshaledData, err := json.Marshal(head)
	require.NoError(t, err)

	// Check that marshaled data
	require.Equal(t, expectedHeaderJson, string(marshaledData))
}

// Smoke test for Header.UnmarshalJSON. Ensures that
func TestHeader_UnmarshalJSON(t *testing.T) {
	// Initialize header object
	oldHeader := NewHeader()

	// Create multiple entries for JSON
	const numTests = 10
	for i := 0; i < numTests; i++ {
		key, val := "key"+strconv.Itoa(i), "val"+strconv.Itoa(i)
		require.NoError(t, oldHeader.Set(key, val))
	}

	// Marshal header
	oldHeaderData, err := json.Marshal(oldHeader)
	require.NoError(t, err)

	// Construct a new header and unmarshal the old header into it
	newHeader := NewHeader()
	require.NoError(t, newHeader.UnmarshalJSON(oldHeaderData))

	// Ensure that the newHeader.UnmarshalJSON call places oldHeader's data
	// into the new header object.
	require.Equal(t, newHeader, oldHeader)

	// Marshal the newHeader into JSON
	newHeaderData, err := json.Marshal(newHeader)
	require.NoError(t, err)

	// Ensure that newHeader's marshalled data matches the expected JSON output
	// (if no data has been lost, this should be the case)
	require.Equal(t, expectedHeaderJson, string(newHeaderData))

}
