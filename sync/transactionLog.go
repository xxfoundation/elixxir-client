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
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/netTime"
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
	writeToStoreErr           = "failed to write to local store: %+v"
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

	// local is the local file system (or anything implementing
	// the FileIO interface).
	local FileIO

	// remote is the store for writing/reading to a remote store. All writes
	// should be asynchronous.
	//
	// FileSystemRemoteStorage is provided as an example.
	remote RemoteStore

	// Header is the Header of the TransactionLog.
	Header *Header

	// txs is a list of transactions. This list must always be ordered by
	// timestamp.
	txs []Transaction

	// offsets is the last index a certain device ID has read.
	offsets deviceOffset

	// deviceSecret is the secret for the device that the TransactionLog will
	// be stored.
	deviceSecret []byte

	// rng is an io.Reader that will be used for encrypt. This should be a
	// secure random number generator (fastRNG.Stream is recommended).
	rng io.Reader

	lck        sync.RWMutex
	openWrites int32
}

// NewOrLoadTransactionLog constructs a new TransactionLog. If the LocalStore
// has serialized data within Note that by default the
// log's header is empty. To set this field, call TransactionLog.SetHeader.
//
// Parameters:
//   - path - the file path that will be used to write transactions, both locally
//     and remotely.
//   - localFS - a filesystem [FileIO] adhering object which will be
//     used to write the transaction log to file.
//   - remote - the RemoteStore adhering object which will be used to write the
//     transaction log to file.
//   - appendCallback - the callback used to report the status of writing to
//     remote.
//   - deviceSecret - the randomly generated secret for this individual device.
//     This should be unique for every device.
//   - rng - An io.Reader used for random generation when encrypting data.
func NewOrLoadTransactionLog(path string, localFS FileIO, remote RemoteStore,
	deviceSecret []byte, rng io.Reader) (*TransactionLog, error) {

	// Construct a new transaction log
	tx := &TransactionLog{
		Header:       NewHeader(),
		path:         path,
		local:        localFS,
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
			return nil, errors.Errorf(loadFromLocalStoreErr, path,
				err)
		}
	}

	// If failed to read, then there is no state, which may or may not be
	// expected. For example, calling this the first time on a device, there
	// will naturally be no state to read.
	jww.DEBUG.Printf("[%s] Failed to read from local when loading: %+v",
		localFS, err)
	return tx, nil
}

// Append will add a transaction to the TransactionLog. This will save the
// serialized TransactionLog to local and remote storage. The callback for
// remote storage will be NewOrLoadTransactionLog or SetRemoteCallback.
func (tl *TransactionLog) Append(newTx Transaction,
	remoteCb RemoteStoreCallback) error {
	atomic.AddInt32(&tl.openWrites, 1)
	defer atomic.AddInt32(&tl.openWrites, -1)
	tl.lck.Lock()
	defer tl.lck.Unlock()
	// Insert new transaction into list
	jww.INFO.Printf("[%s] Inserting transaction to log", logHeader)

	// Use insertion sort as it has been benchmarked to be more performant
	tl.appendUsingInsertion(newTx)

	// Save data to file store
	jww.INFO.Printf("[%s] Saving transaction log", logHeader)

	// Serialize the transaction log
	dataToSave, err := tl.serialize()
	if err != nil {
		return err
	}
	return tl.save(newTx, dataToSave, remoteCb)
}

// WaitForRemote blocks until writes complete or the timeout
// occurs. It returns true if writes completed or false if not.
func (tl *TransactionLog) WaitForRemote(timeout time.Duration) bool {
	t := time.NewTimer(timeout)
	for {
		select {
		case <-time.After(time.Millisecond * 100):
			x := atomic.LoadInt32(&tl.openWrites)
			if x == 0 {
				return true
			}
		case <-t.C:
			return false
		}
	}
}

// appendUsingInsertion will write the new Transaction to txs. txs must be
// ordered by timestamp, so it will the txs list is sorted after appending the
// new Transaction. Sorting is achieved using a custom insertion sort.
//
// Note that this operation is NOT thread-safe, and the caller should hold the
// lck.
func (tl *TransactionLog) appendUsingInsertion(newTransaction Transaction) {
	// If list is empty, just append
	if tl.txs == nil || len(tl.txs) == 0 {
		tl.txs = []Transaction{newTransaction}
		return
	}

	for i := len(tl.txs); i != 0; i-- {
		curidx := i - 1
		if tl.txs[curidx].Timestamp.After(newTransaction.Timestamp) {
			// If we are the start of the list, place at the beginning
			if curidx == 0 {
				tl.txs = append([]Transaction{newTransaction}, tl.txs...)
				return
			}
			continue
		}
		// If the current index is Before, insert just after this index
		insertionIndex := i
		// Just append when we are at the end already
		if insertionIndex == len(tl.txs) {
			tl.txs = append(tl.txs, newTransaction)
		} else {
			tl.txs = append(tl.txs[:insertionIndex+1], tl.txs[insertionIndex:]...)
			tl.txs[insertionIndex] = newTransaction

		}
		return
	}
	return
}

// appendUsingInsertion will write the new Transaction to txs. txs must be
// ordered by timestamp, so it will the txs list is sorted after appending the
// new Transaction. Sorting is achieved using a quick sort as defined by Go's
// native sort package (specifically sort.Slice).
//
// This is not used for production. It is left here for benchmarking purposes to
// compare against appendUsingInsertion.
//
// Note that this operation is NOT thread-safe, and the caller should hold the
// lck.
func (tl *TransactionLog) appendUsingQuickSort(newTransaction Transaction) {
	// Lazily insert new transaction
	tl.txs = append(tl.txs, newTransaction)

	//Sort transaction list. This operates in n * log(n) time complexity
	sort.Slice(tl.txs, func(i, j int) bool {
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

	// Write serialized header to buffer
	buff.Write(headerSerialized)

	// Serialize the device offset
	offsetSerialized, err := tl.offsets.serialize()
	if err != nil {
		return nil, err
	}

	// Write the length of the serialized offset
	offsetLen := len(offsetSerialized)
	buff.Write(serializeInt(offsetLen))

	// Write the serialized offsets
	buff.Write(offsetSerialized)

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

	// Extract offset length from buffer
	offsetLen := deserializeInt(buff.Next(8))
	serializedOffset := buff.Next(int(offsetLen))

	// Deserialize the offset
	offset, err := deserializeDeviceOffset(serializedOffset)
	if err != nil {
		return err
	}

	// Set the offset
	tl.offsets = offset

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
func (tl *TransactionLog) save(newTx Transaction,
	dataToSave []byte, remoteCb RemoteStoreCallback) error {
	// Save to local storage (if set)
	if tl.local == nil {
		jww.FATAL.Panicf("[%s] Cannot write to a nil local store", logHeader)
	}

	jww.INFO.Printf("[%s] Writing transaction log to local store", logHeader)
	if err := tl.local.Write(tl.path, dataToSave); err != nil {
		return errors.Errorf(writeToStoreErr, err)
	}

	// Do not let remote writing block operations
	go tl.saveToRemote(newTx, dataToSave, remoteCb)
	return nil
}

// saveToRemote will write the data to save to remote. It will panic if remote
// or remoteStoreCb are nil.
func (tl *TransactionLog) saveToRemote(newTx Transaction, dataToSave []byte,
	remoteCb RemoteStoreCallback) {
	// Check if remote is set
	if tl.remote == nil {
		jww.FATAL.Panicf("[%s] Cannot write to a nil remote store", logHeader)
	}

	if remoteCb == nil {
		jww.FATAL.Panicf(
			"[%s] Cannot report status of remote storage write for transaction %s to a nil callback",
			logHeader, newTx)
	}

	jww.INFO.Printf("[%s] Writing transaction log to remote store", logHeader)

	// Use callback to report status of remote write.
	remoteCb(newTx, tl.remote.Write(tl.path, dataToSave))
}
