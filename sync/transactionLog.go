////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"sort"
	"strconv"
	"sync"
)

const (
	xxdkTxLogHeader = "XXDKTXLOGHDR"
)

// Error messages.
const (
	writeToBufferErr = "failed to write to buffer (%s): %+v"
	getLastWriteErr  = "failed to get last write operation from remote store: %+v"
	writeToStoreErr  = "failed to write to %s store: %+v"
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

	// hdr is the Header of the TransactionLog.
	hdr *Header

	// txs is a list of transactions. This list must always be ordered by
	// timestamp.
	txs []Transaction

	// curBuf is what is used to serialize the current state of a log so that
	// the state can be written to local and remote store.
	curBuf *bytes.Buffer

	// deviceSecret is the secret for the device that the TransactionLog will
	// be stored.
	deviceSecret []byte

	// rng is an io.Reader that will be used for encrypt. This should be a
	// secure random number generator (fastRNG.Stream is recommended).
	rng io.Reader

	lck sync.RWMutex
}

// NewTransactionLog constructs a new TransactionLog.
func NewTransactionLog(local LocalStore, remote RemoteStore,
	hdr *Header, rng io.Reader, path string, deviceSecret []byte) *TransactionLog {
	// Return a new transaction log
	return &TransactionLog{
		path:         path,
		local:        local,
		remote:       remote,
		txs:          make([]Transaction, 0),
		hdr:          hdr,
		curBuf:       &bytes.Buffer{},
		deviceSecret: deviceSecret,
		rng:          rng,
	}
}

// Append will add a transaction to the TransactionLog. This will save the
// serialized TransactionLog to local and remote storage.
func (tl *TransactionLog) Append(t Transaction) error {
	tl.lck.Lock()

	// Insert new transaction into list
	jww.INFO.Println("[Transaction Log] Inserting transaction to log")
	tl.append(t)

	// Serialize the transaction log
	dataToSave, err := tl.serialize()
	if err != nil {
		return err
	}

	// Release lock now that serialization is complete
	tl.lck.Unlock()

	// Save data to file store
	jww.INFO.Println("[Transaction Log] Saving transaction log")
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

// serialize serializes the state of TransactionLog to byte data that can be
// written to a store (remote, local or both).
func (tl *TransactionLog) serialize() ([]byte, error) {
	// Refresh buffer after returning serialized data
	defer tl.curBuf.Reset()

	// Marshal header into JSON
	headerMarshal, err := json.Marshal(tl.hdr)
	if err != nil {
		return nil, err
	}

	// Write serialized header into buffer
	_, err = tl.curBuf.WriteString(xxdkTxLogHeader +
		base64.URLEncoding.EncodeToString(headerMarshal))
	if err != nil {
		return nil, errors.Errorf(writeToBufferErr,
			xxdkTxLogHeader+base64.URLEncoding.EncodeToString(headerMarshal),
			err)
	}

	lastRemoteWrite, err := tl.remote.GetLastWrite()
	if err != nil {
		return nil, errors.Errorf(getLastWriteErr, err)
	}

	// Serialize all transactions
	for i := 0; i < len(tl.txs); i++ {

		// Construct cMix hash
		h, err := hash.NewCMixHash()
		if err != nil {
			return nil, err
		}

		// Construct secret for encryption
		h.Write(binary.LittleEndian.AppendUint16(make([]byte, 0), uint16(i)))
		h.Write(tl.deviceSecret)
		secret := h.Sum(nil)

		if tl.txs[i].Timestamp.After(lastRemoteWrite) {
			// Timestamp must be updated every write attempt time if new entry
			tl.txs[i].Timestamp = netTime.Now()
		}

		// Marshal the current transaction
		txMarshal, err := json.Marshal(tl.txs[i])
		if err != nil {
			return nil, err
		}

		// Encrypt the current transaction
		encrypted := encrypt(txMarshal, string(secret), tl.rng)

		// Write the encrypted transaction to the buffer
		_, err = tl.curBuf.WriteString(strconv.Itoa(i) + "," +
			base64.URLEncoding.EncodeToString(encrypted))
		if err != nil {
			return nil, errors.Errorf(writeToBufferErr,
				strconv.Itoa(i)+","+
					base64.URLEncoding.EncodeToString(encrypted),
				err)

		}
	}

	return tl.curBuf.Bytes(), nil
}

// save writes the data passed int to file, both remotely and locally. The
// data passed in should be read in from curBuf.
func (tl *TransactionLog) save(dataToSave []byte) error {

	// Save to local storage (if set)
	if tl.local != nil {
		jww.INFO.Println("[Transaction Log] Writing transaction log to local store")
		if err := tl.local.Write(tl.path, dataToSave); err != nil {
			return errors.Errorf(writeToStoreErr, "local", err)
		}
	}

	// Save to remote storage (if set)
	if tl.remote != nil {
		jww.INFO.Println("[Transaction Log] Writing transaction log to remote store")
		if err := tl.remote.Write(tl.path, dataToSave); err != nil {
			return errors.Errorf(writeToStoreErr, "remote", err)
		}
	}

	return nil
}
