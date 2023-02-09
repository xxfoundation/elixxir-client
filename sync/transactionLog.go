////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"sort"
	"sync"
)

// TransactionLog constants.
const (
	// The prefix for a serialized header.
	xxdkTxLogHeader = "XXDKTXLOGHDR"

	// The delimiter for a serialized transaction.
	xxdkTxLogDelim = ","

	// The header for the jww log print.
	logHeader = "Transaction Log"
)

// Error messages.
const (
	getLastWriteErr           = "failed to get last write operation from remote store: %+v"
	writeToStoreErr           = "failed to write to %s store: %+v"
	loadFromLocalStoreErr     = "failed to deserialize log from local store at path %s: %+v"
	deserializeTransactionErr = "failed to deserialize transaction (%d/%d): %+v"
)

// TransactionLog will log all Transaction's to a storage interface. It will
// contain all Transaction's in an ordered list, and will ensure to retain order
// when Append is called. This will store to a LocalStore and a RemoteStore when
// appending Transaction's.
type TransactionLog struct {
	// path is the filepath that the TransactionLog will be written to on remote
	// and local storage.
	path string

	// local is the store for writing/reading to a local store.
	//
	// EkvLocalStore is provided as an example.
	local LocalStore

	// remote is the store for writing/reading to a remote store.
	//
	// FileSystemRemoteStorage is provided as an example.
	remote RemoteStore

	// Header is the Header of the TransactionLog.
	Header *Header

	// txs is a list of transactions. This list must always be ordered by
	// timestamp.
	txs []Transaction

	// deviceSecret is the secret for the device that the TransactionLog will
	// be stored.
	deviceSecret []byte

	// rng is an io.Reader that will be used for encrypt. This should be a
	// secure random number generator (fastRNG.Stream is recommended).
	rng io.Reader

	lck sync.RWMutex
}

// NewOrLoadTransactionLog constructs a new TransactionLog. If the LocalStore
// has serialized data within Note that by default the
// log's header is empty. To set this field, call TransactionLog.SetHeader.
func NewOrLoadTransactionLog(local LocalStore, remote RemoteStore,
	rng io.Reader, path string, deviceSecret []byte) (*TransactionLog, error) {

	// Construct a new transaction log
	tx := &TransactionLog{
		Header:       NewHeader(),
		path:         path,
		local:        local,
		remote:       remote,
		txs:          make([]Transaction, 0),
		deviceSecret: deviceSecret,
		rng:          rng,
	}

	// Attempt to read stored transaction log
	data, err := tx.local.Read(path)
	if err == nil {
		// If data has been read, attempt to deserialize
		if err = tx.deserialize(data); err != nil {
			return nil, errors.Errorf(loadFromLocalStoreErr, path, err)
		}
	}

	// If failed to read, then there is no state
	jww.DEBUG.Printf("[%s] Failed to read from local when loading: %+v", local, err)
	return tx, nil
}

// Append will add a transaction to the TransactionLog. This will save the
// serialized TransactionLog to local and remote storage.
func (tl *TransactionLog) Append(t Transaction) error {
	tl.lck.Lock()

	// Insert new transaction into list
	jww.INFO.Printf("[%s] Inserting transaction to log", logHeader)
	tl.append(t)

	// Save data to file store
	jww.INFO.Printf("[%s] Saving transaction log", logHeader)

	// Serialize the transaction log
	dataToSave, err := tl.serialize()
	if err != nil {
		return err
	}
	tl.lck.Unlock()
	return tl.save(dataToSave)
}

// append will write the new Transaction to txs. txs must be ordered by
// timestamp, so it will the txs list is sorted after appending the new
// Transaction.
//
// Note that this operation is NOT thread-safe, and the caller should hold the
// lck.
func (tl *TransactionLog) append(newTransaction Transaction) {
	// Lazily insert new transaction
	tl.txs = append(tl.txs, newTransaction)

	// Sort transaction list. This operates in n * log(n) time complexity
	sort.SliceStable(tl.txs, func(i, j int) bool {
		firstTs, secondTs := tl.txs[i].Timestamp, tl.txs[j].Timestamp
		return firstTs.Before(secondTs)
	})

}

// serialize serializes the state of TransactionLog to byte data, so that it can
// be written to a store (remote, local or both).
//
// This is the inverse operation of TransactionLog.deserialize.
func (tl *TransactionLog) serialize() ([]byte, error) {
	buff := new(bytes.Buffer)

	// Serialize header
	headerSerialized, err := tl.Header.serialize()
	if err != nil {
		return nil, err
	}

	// Write the length of the header info into the buffer
	headerInfoLen := len(headerSerialized)
	buff.Write(serializeInt(headerInfoLen))

	// Write serialized header to bufer
	buff.Write(headerSerialized)

	// Retrieve the last written timestamp from remote
	lastRemoteWrite, err := tl.remote.GetLastWrite()
	if err != nil {
		return nil, errors.Errorf(getLastWriteErr, err)
	}

	// Serialize the length of the list
	buff.Write(serializeInt(len(tl.txs)))

	// Serialize all transactions
	for i := 0; i < len(tl.txs); i++ {
		// Timestamp must be updated every write attempt time if new entry
		if tl.txs[i].Timestamp.After(lastRemoteWrite) {
			tl.txs[i].Timestamp = netTime.Now()
		}

		// Serialize transaction
		txSerialized, err := tl.txs[i].serialize(tl.deviceSecret, i, tl.rng)
		if err != nil {
			return nil, err
		}

		// Write the length of the transaction info into the buffer
		txInfoLen := len(txSerialized)
		buff.Write(serializeInt(txInfoLen))

		// Write to buffer
		buff.Write(txSerialized)

	}

	return buff.Bytes(), nil
}

// deserialize will deserialize TransactionLog byte data.
//
// This is the inverse operation of TransactionLog.serialize.
func (tl *TransactionLog) deserialize(data []byte) error {
	// Initialize buffer
	buff := bytes.NewBuffer(data)

	// Extract header length from buffer
	lengthOfHeaderInfo := deserializeInt(buff.Next(8))
	serializedHeader := buff.Next(int(lengthOfHeaderInfo))

	// Deserialize header
	hdr, err := deserializeHeader(serializedHeader)
	if err != nil {
		return err
	}

	// Set the header
	tl.Header = hdr

	// Deserialize length of transactions list
	listLen := binary.LittleEndian.Uint64(buff.Next(8))

	// Construct transactions list
	txs := make([]Transaction, listLen)

	// Iterate over transaction log
	for i := range txs {
		//Read length of transaction from buffer
		txInfoLen := deserializeInt(buff.Next(8))
		txInfo := buff.Next(int(txInfoLen))
		tx, err := deserializeTransaction(txInfo, tl.deviceSecret)
		if err != nil {
			return errors.Errorf(deserializeTransactionErr, i, listLen, err)
		}

		txs[i] = tx
	}

	tl.txs = txs

	return nil
}

// save writes the data passed int to file, both remotely and locally. The data
// created from serialize.
func (tl *TransactionLog) save(dataToSave []byte) error {
	tl.lck.Lock()

	// Save to local storage (if set)
	if tl.local == nil {
		tl.lck.Unlock()
		jww.FATAL.Panicf("[%s] Cannot write to a nil local store", logHeader)
	}

	jww.INFO.Printf("[%s] Writing transaction log to local store", logHeader)
	if err := tl.local.Write(tl.path, dataToSave); err != nil {
		tl.lck.Unlock()
		return errors.Errorf(writeToStoreErr, "local", err)
	}

	// Do not let remote writing block operations
	// fixme: consider making writing a go routine, then locking can
	//  be done by the caller of save
	tl.lck.Unlock()

	// Save to remote storage (if set)
	if tl.remote == nil {
		jww.FATAL.Panicf("[%s] Cannot write to a nil remote store", logHeader)
	}

	jww.INFO.Printf("[%s] Writing transaction log to remote store", logHeader)
	if err := tl.remote.Write(tl.path, dataToSave); err != nil {
		return errors.Errorf(writeToStoreErr, "remote", err)
	}

	return nil
}

// serializeInt is a utility function which serializes an integer into a byte
// slice.
//
// This is the inverse operation of deserializeInt.
func serializeInt(i int) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	return b
}

// deserializeInt is a utility function which deserializes byte data into an
// integer.
//
// This is the inverse operation of serializeInt.
func deserializeInt(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
