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

const expectedJsonOutput = `{"version":0,"entries":{"key0":"val0","key1":"val1","key2":"val2","key3":"val3","key4":"val4","key5":"val5","key6":"val6","key7":"val7","key8":"val8","key9":"val9"}}`
const (
	// expectedHeaderJsonNewline is the result of calling json.MarshalIndent
	// on a Header object. expectedHeaderJsonNewline is presented with idents to
	// illustrate that the newline character `\n` is parsed as part of the key
	// and not as the escape character. Note that if one were to json.Unmarshal
	// this back into a Header object, then json.Marshal that object, the output
	// would be a single line of text.
	expectedHeaderJsonNewline = `{
	"version": 0,
	"entries": {
		"edgeCheckKey\n": "edgeCheckVal",
		"key0": "val0",
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
		"key5": "val5",
		"key6": "val6",
		"key7": "val7",
		"key8": "val8",
		"key9": "val9"
	}
}`
)

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

// Smoke test for Header.MarshalJSON. Checks basic marshaling outputs expected
// data. Further edge checks that when given a key with a newline character, the
// character is parsed as part of a string value and not as an escape character.
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
	require.Equal(t, expectedJsonOutput, string(marshaledData))

	// Edge check: Add a key with a newline character
	key, val := "edgeCheckKey\n", "edgeCheckVal"
	require.NoError(t, head.Set(key, val))

	marshaledData, err = json.MarshalIndent(head, "", "\t")
	require.NoError(t, err)

	t.Logf("%s", marshaledData)

	// Ensure it outputs a single line, ie the newline character does not
	// create a multi-line JSON file.
	require.Equal(t, expectedHeaderJsonNewline, string(marshaledData))
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
	require.Equal(t, expectedJsonOutput, string(newHeaderData))

}
