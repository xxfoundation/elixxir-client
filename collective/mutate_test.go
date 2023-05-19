////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Smoke test for NewMutate.
func TestNewTransaction(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct expected Mutate object
	_, val := "key", []byte("value")
	expectedTx := Mutate{
		Timestamp: testTime.UTC().Unix(),
		Value:     val,
	}

	require.Equal(t, expectedTx, NewMutate(testTime, val, false))
}

// Smoke & unit test for Mutate.MarshalJSON.
func TestTransaction_MarshalJSON(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Mutate object
	_, val := "key", []byte("value")
	tx := NewMutate(testTime, val, false)

	// Marshal Mutate into JSON data
	marshalledData, err := json.Marshal(tx)
	require.NoError(t, err)

	expectedTransactionJson := `{"Timestamp":1356127721,"Value":"dmFsdWU=","Deletion":false}`

	// Check that marshaled data matches expected value
	require.Equal(t, expectedTransactionJson, string(marshalledData))

}

// Smoke & unit test for Mutate.UnmarshalJSON.
func TestTransaction_UnmarshalJSON(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Mutate object
	_, val := "key", []byte("value")
	oldTx := NewMutate(testTime, val, false)

	// Marshal mutate into JSON data
	oldTxData, err := json.Marshal(oldTx)
	require.NoError(t, err)

	// Construct a new mutate and unmarshal the old mutate into it
	newTx := NewMutate(time.Time{}, make([]byte, 0), false)
	require.NoError(t, json.Unmarshal(oldTxData, &newTx))

	// Ensure that the newTx.UnmarshalJSON call places
	// oldTx's data into the new mutate object.
	require.Equal(t, oldTx, newTx)

	// Marshal the newTx into JSON
	newTxData, err := json.Marshal(newTx)
	require.NoError(t, err)

	expectedTransactionJson := `{"Timestamp":1356127721,"Value":"dmFsdWU=","Deletion":false}`

	// Ensure that newTx's marshalled data matches the expected JSON
	// output (if no data has been lost, this should be the case)
	require.Equal(t, expectedTransactionJson, string(newTxData))

}

// Edge check: check that a zero value time.Time object gets marshalled
// and unmarshalled properly.
func TestTransaction_UnmarshalJSON_ZeroTime(t *testing.T) {
	testTime := time.Unix(0, 0)

	// Construct a Mutate object
	_, val := "key", []byte("value")
	oldTx := NewMutate(testTime, val, false)

	// Marshal mutate into JSON data
	oldTxData, err := json.Marshal(oldTx)
	require.NoError(t, err)

	expectedTransactionZeroTimeJson := `{"Timestamp":0,"Value":"dmFsdWU=","Deletion":false}`

	require.Equal(t, expectedTransactionZeroTimeJson, string(oldTxData))

	// Construct a new mutate and unmarshal the old mutate into it
	newTx := NewMutate(time.Unix(0, 0), make([]byte, 0), false)
	require.NoError(t, json.Unmarshal(oldTxData, &newTx))

	require.Equal(t, newTx.Timestamp, testTime.Unix())
}

// Smoke test of Mutate.serialize.
func TestTransaction_Serialize(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Mutate object
	_, val := "key", []byte("value")
	tx := NewMutate(testTime, val, false)

	// Serialize mutate
	txSerial, err := json.Marshal(tx)
	require.NoError(t, err)

	expectedSerializedTransaction := "eyJUaW1lc3RhbXAiOjEzNTYxMjc3MjEsIlZhbHVlIjoiZG1Gc2RXVT0iLCJEZWxldGlvbiI6ZmFsc2V9"

	// Ensure serialization is consistent
	require.Equal(t, expectedSerializedTransaction,
		base64.StdEncoding.EncodeToString(txSerial))
}

// Unit test of DeserializeTransaction. Ensures that deserialize will construct
// the same Mutate that was serialized using Mutate.serialize.
func TestTransaction_Deserialize(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Mutate object
	_, val := "key", []byte("value")
	tx := NewMutate(testTime, val, false)

	// Serialize mutate
	txSerial, err := json.Marshal(tx)
	require.NoError(t, err)

	// Deserialize mutate
	txDeserialize := &Mutate{}
	err = json.Unmarshal(txSerial, txDeserialize)
	require.NoError(t, err)

	// Ensure deserialized object matches original object
	require.Equal(t, tx, *txDeserialize)
}
