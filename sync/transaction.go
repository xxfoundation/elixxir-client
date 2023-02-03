////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
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

// transaction is the object to which Transaction strictly adheres. transaction
// serves as the marshal-able an unmarshal-able object that
// Transaction.MarshalJSON and Transaction.UnmarshalJSON utilizes when calling
// json.Marshal/json.Unmarshal.
//
// WARNING: Modifying transaction will modify Transaction, be mindful of the
// consumers when modifying this structure.
type transaction struct {
	Timestamp time.Time
	Key       string
	Value     []byte
}

// fixme: This is commented out until Transaction needs a Marshaler/Unmarshaler.
//// MarshalJSON marshals the Transaction into valid JSON. This function adheres
//// to the json.Marshaler interface.
//func (t *Transaction) MarshalJSON() ([]byte, error) {
//	return json.Marshal(transaction(*t))
//}
//
//// UnmarshalJSON unmarshalls JSON into the Transaction. This function adheres to
//// the json.Unmarshaler interface.
//func (t *Transaction) UnmarshalJSON(data []byte) error {
//	transData := transaction{}
//	if err := json.Unmarshal(data, &transData); err != nil {
//		return err
//	}
//	*t = Transaction(transData)
//	return nil
//
//}
