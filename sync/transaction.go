////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/hash"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	transactionUnexpectedSerialErr = "unexpected data in serialized trnasaction."
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

// serialize serializes a Transaction object. More accurately, since the
// serialization will be stored remotely, serialize will encrypt the Transaction
// and encode the encryption.
//
// Use deserializeTransaction to reverse this operation.
//
// Arguments:
//   - deviceSecret - []byte, the secret used for the device.
//   - index - int, the position of the Transaction within TransactionLog.txs.
//   - rng - An io.Reader, used to encrypt the Transaction.
func (t *Transaction) serialize(deviceSecret []byte, index int,
	rng io.Reader) ([]byte, error) {

	// Marshal the current transaction
	txMarshal, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	// Encrypt the current transaction
	secret := makeTransactionSecret(deviceSecret, index)
	encrypted := encrypt(txMarshal, string(secret), rng)

	// Construct a transaction info
	txInfo := strconv.Itoa(index) + xxdkTxLogDelim +
		base64.URLEncoding.EncodeToString(encrypted)

	return []byte(txInfo), nil
}

// deserializeTransaction will deserialize transaction data. More accurately,
// this will decode and decrypt the transaction byte data.
//
// This is the inverse operation of Transaction.serialize.
//
// Arguments:
//   - txInfo - []byte, the serialized data of a Transaction.
//   - deviceSecret - []byte, the secret used for the device.
func deserializeTransaction(txInfo, deviceSecret []byte) (Transaction, error) {

	// Extract index and encoded transaction
	splitter := strings.Split(string(txInfo), xxdkTxLogDelim)
	if len(splitter) != 2 {
		return Transaction{}, errors.Errorf(transactionUnexpectedSerialErr)
	}
	indexStr, txEncoded := splitter[0], splitter[1]

	// Convert index into integer
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return Transaction{}, err
	}

	// Decode transaction
	txEncrypted, err := base64.URLEncoding.DecodeString(txEncoded)
	if err != nil {
		return Transaction{}, err
	}

	// Construct secret
	txSecret := makeTransactionSecret(deviceSecret, index)

	// Decrypt transaction
	txMarshal, err := decrypt(txEncrypted, string(txSecret))
	if err != nil {
		return Transaction{}, err
	}

	// Unmarshal transaction
	tx := Transaction{}
	if err = json.Unmarshal(txMarshal, &tx); err != nil {
		return Transaction{}, err
	}

	return tx, nil
}

// makeTransactionSecret is a utility function which generates the secret used
// to encrypt or decrypt a transaction.
func makeTransactionSecret(deviceSecret []byte, index int) []byte {
	// Construct cMix hash
	h := hash.CMixHash.New()

	// Construct secret for encryption
	serializedIndex := serializeInt(index)
	h.Write(serializedIndex)
	h.Write(deviceSecret)
	return h.Sum(nil)
}
