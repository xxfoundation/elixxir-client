////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"time"
)

// Transaction is the object that is uploaded to a remote service responsible
// for account synchronization.
type Transaction struct {
	// Timestamp is the time of the transaction based on a cMix time oracle.
	Timestamp time.Time

	// The key of the transaction (e.g. the device prefix)
	Key string

	// The value of the transaction (e.g. the state change update).
	Value []byte
}

// NewTransaction is the constructor of a Transaction object.
func NewTransaction(ts time.Time, key string, value []byte) Transaction {
	return Transaction{
		Timestamp: ts.UTC(),
		Key:       key,
		Value:     value,
	}
}

// transaction is an object strictly adhering to Transaction. This serves as the
// marshal-able an unmarshal-able object such that transaction may adhere to the
// json.Marshaler and json.Unmarshaler interfaces.
//
// WARNING: If transaction is modified, header should reflect these changes to
// ensure no data is lost when calling the json.Marshaler or json.Unmarshaler.
type transaction struct {
	Timestamp time.Time
	Key       string
	Value     []byte
}

// MarshalJSON marshals the Transaction into valid JSON. This function adheres
// to the json.Marshaler interface.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	marshaller := transaction{
		Timestamp: t.Timestamp,
		Key:       t.Key,
		Value:     t.Value,
	}

	return json.Marshal(marshaller)
}

// UnmarshalJSON unmarshalls JSON into the Transaction. This function adheres to
// the json.Unmarshaler interface.
func (t *Transaction) UnmarshalJSON(data []byte) error {
	transactionData := transaction{}
	err := json.Unmarshal(data, &transactionData)
	if err != nil {
		return err
	}

	*t = Transaction{
		Timestamp: transactionData.Timestamp,
		Key:       transactionData.Key,
		Value:     transactionData.Value,
	}

	return nil
}
