////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	expectedTransactionJson         = `{"Timestamp":"2012-12-21T22:08:41Z","Key":"key","Value":"dmFsdWU="}`
	expectedTransactionZeroTimeJson = `{"Timestamp":"0001-01-01T00:00:00Z","Key":"key","Value":"dmFsdWU="}`
)

// Smoke test for NewTransaction.
func TestNewTransaction(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct expected Transaction object
	key, val := "key", []byte("value")
	expectedTransaction := Transaction{
		Timestamp: testTime.UTC(),
		Key:       key,
		Value:     val,
	}

	require.Equal(t, expectedTransaction, NewTransaction(testTime, key, val))
}

// Smoke & unit test for Transaction.MarshalJSON.
func TestTransaction_MarshalJSON(t *testing.T) {
	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	// Construct a Transaction object
	key, val := "key", []byte("value")
	transObj := NewTransaction(testTime, key, val)

	// Marshal Transaction into JSON data
	marshalledData, err := json.Marshal(transObj)
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
	oldTransaction := NewTransaction(testTime, key, val)

	// Marshal transaction into JSON data
	oldTransactionData, err := json.Marshal(oldTransaction)
	require.NoError(t, err)

	// Construct a new transaction and unmarshal the old transaction into it
	newTransaction := NewTransaction(time.Time{}, "", make([]byte, 0))
	require.NoError(t, json.Unmarshal(oldTransactionData, &newTransaction))

	// Ensure that the newTransaction.UnmarshalJSON call places
	// oldTransaction's data into the new transaction object.
	require.Equal(t, oldTransaction, newTransaction)

	// Marshal the newTransaction into JSON
	newTransactionData, err := json.Marshal(newTransaction)
	require.NoError(t, err)

	// Ensure that newTransaction's marshalled data matches the expected JSON
	// output (if no data has been lost, this should be the case)
	require.Equal(t, expectedTransactionJson, string(newTransactionData))

}

// Edge check: check that a zero value time.Time object gets marshalled
// and unmarshalled properly.
func TestTransaction_UnmarshalJSON_ZeroTime(t *testing.T) {
	testTime := time.Time{}

	// Construct a Transaction object
	key, val := "key", []byte("value")
	oldTransaction := NewTransaction(testTime, key, val)

	// Marshal transaction into JSON data
	oldTransactionData, err := json.Marshal(oldTransaction)
	require.NoError(t, err)

	require.Equal(t, expectedTransactionZeroTimeJson, string(oldTransactionData))

	// Construct a new transaction and unmarshal the old transaction into it
	newTransaction := NewTransaction(time.Time{}, "", make([]byte, 0))
	require.NoError(t, json.Unmarshal(oldTransactionData, &newTransaction))

	require.True(t, newTransaction.Timestamp.Equal(testTime))
}
