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
// for account synchronization. It inherits the private transaction object.
// This prevents recursive calls by json.Marshal on Header.MarshalJSON. Any
// changes to the Header object fields should be done in header.
type Transaction transaction

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
	return json.Marshal(transaction(*t))
}

// UnmarshalJSON unmarshalls JSON into the Transaction. This function adheres to
// the json.Unmarshaler interface.
func (t *Transaction) UnmarshalJSON(data []byte) error {
	headerData := transaction{}
	if err := json.Unmarshal(data, &headerData); err != nil {
		return err
	}
	*t = Transaction(headerData)
	return nil

}
