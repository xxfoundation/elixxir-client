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

	expectedHeaderJson := `{"version":0,"device":"U4x_lrFkvxs"}`

	// Check that marshaled data matches expected JSON
	require.Equal(t, expectedHeaderJson, string(marshaledData))
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

	expectedHeaderJson := `{"version":0,"device":"U4x_lrFkvxs"}`

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

	expectedHeaderSerial := "WFhES1RYTE9HSERSZXlKMlpYSnphVzl1SWpvd0xDSmtaWFpwWTJVaU9pSlZOSGhmYkhKR2EzWjRjeUo5"

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
