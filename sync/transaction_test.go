////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Hard-coded constants for testing purposes.
const (
	expectedTransactionJson         = `{"Timestamp":"2012-12-21T22:08:41Z","Key":"key","Value":"dmFsdWU="}`
	expectedTransactionZeroTimeJson = `{"Timestamp":"0001-01-01T00:00:00Z","Key":"key","Value":"dmFsdWU="}`
	expectedSerializedTransaction   = `kgAAAAAAAAAwLEFRSURCQVVHQndnSkNnc01EUTRQRUJFU0V4UVZGaGNZcERlRHFmOU1jSzJrVUhxamZVbnRIdkhVb053aWdiN3pZMENBb0xnMzIxWDJiVERRQ1JpeU8ySEJYbWFLeEtYSTRMMWItb0dvV24wNzhOTkhhNkxsNk1kczJya0lDYmJ4UTZFOTcwOUgzbkQ1OTdBPQ==`
)

// Smoke test for NewTransaction.
func TestNewTransaction(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct expected Transaction object
	key, val := "key", []byte("value")
	expectedTx := Transaction{
		Timestamp: testTime.UTC(),
		Key:       key,
		Value:     val,
	}

	require.Equal(t, expectedTx, NewTransaction(testTime, key, val))
}

// Smoke & unit test for Transaction.MarshalJSON.
func TestTransaction_MarshalJSON(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Transaction object
	key, val := "key", []byte("value")
	tx := NewTransaction(testTime, key, val)

	// Marshal Transaction into JSON data
	marshalledData, err := json.Marshal(tx)
	require.NoError(t, err)

	// Check that marshaled data matches expected value
	require.Equal(t, expectedTransactionJson, string(marshalledData))

}

// Smoke & unit test for Transaction.UnmarshalJSON.
func TestTransaction_UnmarshalJSON(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Transaction object
	key, val := "key", []byte("value")
	oldTx := NewTransaction(testTime, key, val)

	// Marshal transaction into JSON data
	oldTxData, err := json.Marshal(oldTx)
	require.NoError(t, err)

	// Construct a new transaction and unmarshal the old transaction into it
	newTx := NewTransaction(time.Time{}, "", make([]byte, 0))
	require.NoError(t, json.Unmarshal(oldTxData, &newTx))

	// Ensure that the newTx.UnmarshalJSON call places
	// oldTx's data into the new transaction object.
	require.Equal(t, oldTx, newTx)

	// Marshal the newTx into JSON
	newTxData, err := json.Marshal(newTx)
	require.NoError(t, err)

	// Ensure that newTx's marshalled data matches the expected JSON
	// output (if no data has been lost, this should be the case)
	require.Equal(t, expectedTransactionJson, string(newTxData))

}

// Edge check: check that a zero value time.Time object gets marshalled
// and unmarshalled properly.
func TestTransaction_UnmarshalJSON_ZeroTime(t *testing.T) {
	testTime := time.Time{}

	// Construct a Transaction object
	key, val := "key", []byte("value")
	oldTx := NewTransaction(testTime, key, val)

	// Marshal transaction into JSON data
	oldTxData, err := json.Marshal(oldTx)
	require.NoError(t, err)

	require.Equal(t, expectedTransactionZeroTimeJson, string(oldTxData))

	// Construct a new transaction and unmarshal the old transaction into it
	newTx := NewTransaction(time.Time{}, "", make([]byte, 0))
	require.NoError(t, json.Unmarshal(oldTxData, &newTx))

	require.True(t, newTx.Timestamp.Equal(testTime))
}

// Smoke test of Transaction.serialize.
func TestTransaction_Serialize(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Transaction object
	key, val := "key", []byte("value")
	tx := NewTransaction(testTime, key, val)

	// Serialize transaction
	secret, mockRng := []byte("secret"), &CountingReader{count: 0}
	txSerial, err := tx.serialize(secret, 0, mockRng)
	require.NoError(t, err)

	// Ensure serialization is consistent
	require.Equal(t, expectedSerializedTransaction,
		base64.StdEncoding.EncodeToString(txSerial))
}

func TestTransaction_Deserialize(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Transaction object
	key, val := "key", []byte("value")
	tx := NewTransaction(testTime, key, val)

	// Serialize transaction
	secret, mockRng := []byte("secret"), &CountingReader{count: 0}
	txSerial, err := tx.serialize(secret, 0, mockRng)
	require.NoError(t, err)

	// Find the length of the transaction
	buff := bytes.NewBuffer(txSerial)
	txInfoLen := deserializeInt(buff.Next(8))

	// Extract transaction info from buffer
	txInfo := buff.Next(int(txInfoLen))

	// Deserialize transaction
	txDeserial, err := deserializeTransaction(txInfo, secret)
	require.NoError(t, err)

	require.Equal(t, tx, txDeserial)

}
