////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// expectedHeaderJson is the expected result for calling json.Marshal on a
	// header object with example data.
	expectedHeaderJson = `{"version":0,"entries":{"key0":"val0","key1":"val1","key2":"val2","key3":"val3","key4":"val4","key5":"val5","key6":"val6","key7":"val7","key8":"val8","key9":"val9"}}`

	// expectedHeaderJsonNewline is the expected result of calling
	// json.MarshalIndent on a header object with example data. This differs
	// from expectedHeaderJson by having a newline character, `\n`. within
	// header.entries. expectedHeaderJsonNewline is presented with idents to
	// illustrate that the newline character `\n` is parsed as part of the key
	// and not as the escape character. Note that if one were to json.Unmarshal
	// this back into a header object, then json.Marshal that object, the output
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

	// expectedHeaderSerial is the expected result after calling
	// header.serialize with example data.
	expectedHeaderSerial = `WFhES1RYTE9HSERSZXlKMlpYSnphVzl1SWpvd0xDSmxiblJ5YVdWeklqcDdJbXRsZVNJNkluWmhiQ0o5ZlE9PQ==`
)

// Unit test of newHeader.
func TestNewHeader(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	dvcID, err := NewRandomInstanceID(rng)
	require.NoError(t, err)
	receivedHeader := newHeader(dvcID)
	expectedHeader := &header{
		Version:  headerVersion,
		DeviceID: dvcID,
	}
	require.Equal(t, expectedHeader, receivedHeader)
}

// Smoke & unit test for header.MarshalJSON. Checks basic marshaling outputs expected
// data. Further edge checks that when given a key with a newline character, the
// character is parsed as part of a string value and not as an escape character.
func TestHeader_MarshalJSON(t *testing.T) {
	// Initialize header object
	rng := rand.New(rand.NewSource(42))
	dvcID, err := NewRandomInstanceID(rng)
	require.NoError(t, err)
	head := newHeader(dvcID)

	// Marshal header into JSON byte data
	marshaledData, err := json.Marshal(head)
	require.NoError(t, err)

	// Check that marshaled data matches expected JSON
	require.Equal(t, expectedHeaderJson, string(marshaledData))

	// Ensure it outputs a single line, ie the newline character does not
	// create a multi-line JSON file.
	require.Equal(t, expectedHeaderJsonNewline, string(marshaledData))
}

// Smoke & unit test for header.UnmarshalJSON.
func TestHeader_UnmarshalJSON(t *testing.T) {
	// Initialize header object
	rng := rand.New(rand.NewSource(42))
	dvcID, err := NewRandomInstanceID(rng)
	require.NoError(t, err)
	oldHeader := newHeader(dvcID)

	// Marshal header
	oldHeaderData, err := json.Marshal(oldHeader)
	require.NoError(t, err)

	// Construct a new header and unmarshal the old header into it
	newHeader := &header{}
	require.NoError(t, json.Unmarshal(oldHeaderData, newHeader))

	// Ensure that the newHeader.UnmarshalJSON call places oldHeader's data
	// into the new header object.
	require.Equal(t, oldHeader, newHeader)

	// Marshal the newHeader into JSON
	newHeaderData, err := json.Marshal(newHeader)
	require.NoError(t, err)

	// Ensure that newHeader's marshalled data matches the expected JSON output
	// (if no data has been lost, this should be the case)
	require.Equal(t, expectedHeaderJson, string(newHeaderData))

	// Edge check: Testing that entering invalid JSON fails Unmarshal
	require.Error(t, json.Unmarshal([]byte("badJSON"), newHeader))

}

// Smoke test of header.serialize.
func TestHeader_Serialize(t *testing.T) {
	// Initialize header object
	rng := rand.New(rand.NewSource(42))
	dvcID, err := NewRandomInstanceID(rng)
	require.NoError(t, err)
	head := newHeader(dvcID)

	// Serialize header
	hdrSerial, err := head.serialize()
	require.NoError(t, err)

	// Ensure serialization is consistent
	require.Equal(t, expectedHeaderSerial,
		base64.StdEncoding.EncodeToString(hdrSerial))
}

// Unit test of deserializeHeader. Ensures that deserialize will construct
// the same header that was serialized using header.serialize.
func TestHeader_Deserialize(t *testing.T) {
	// Initialize header object
	rng := rand.New(rand.NewSource(42))
	dvcID, err := NewRandomInstanceID(rng)
	require.NoError(t, err)
	head := newHeader(dvcID)

	// Serialize header
	hdrSerial, err := head.serialize()
	require.NoError(t, err)

	// Deserialize header
	hdrDeserialize, err := deserializeHeader(hdrSerial)
	require.NoError(t, err)

	// Ensure deserialized object matches original object
	require.Equal(t, head, hdrDeserialize)
}
