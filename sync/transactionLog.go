////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"gitlab.com/elixxir/crypto/hash"
	"io"
	"sort"
	"strconv"
	"sync"
)

const (
	xxdkTxLogHeader = "XXDKTXLOGHDR"
)

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
	curBuf *bufio.ReadWriter

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
	// Construct reader/writer for the buffer
	var (
		writer = bufio.NewWriter(&bytes.Buffer{})
		reader = bufio.NewReader(bytes.NewReader([]byte{}))
	)

	// Return a new transaction log
	return &TransactionLog{
		path:         path,
		local:        local,
		remote:       remote,
		txs:          make([]Transaction, 0),
		hdr:          hdr,
		curBuf:       bufio.NewReadWriter(reader, writer),
		deviceSecret: deviceSecret,
		rng:          rng,
	}
}

// Append will add a transaction to the TransactionLog. This will save the
// serialized TransactionLog to local and remote storage.
func (tl *TransactionLog) Append(t Transaction) error {
	tl.lck.Lock()

	// Insert new transaction into list
	tl.append(t)

	// Serialize the transaction log
	dataToSave, err := tl.serialize()
	if err != nil {
		return err
	}

	// Release lock now that serialization is complete
	tl.lck.Unlock()

	// Save data to file store
	return tl.save(dataToSave)
}

// append will write the new Transaction to txs. txs must be ordered by
// timestamp, so it will ensure to place the new Transaction in the proper slot
// of the list.
//
// Note that this operation is NOT thread-safe, and the caller should hold the
// lck.
func (tl *TransactionLog) append(newTransaction Transaction) {
	tl.txs = append(tl.txs, newTransaction)
	sort.SliceStable(tl.txs, func(i, j int) bool {
		firstTs, secondTs := tl.txs[i].Timestamp, tl.txs[j].Timestamp
		return firstTs.Before(secondTs)
	})

}

// write writes the state of the TransactionLog to a buffer and saves the
func (tl *TransactionLog) serialize() ([]byte, error) {
	defer tl.curBuf.Flush()

	// Marshal header into JSON
	headerMarshal, err := json.Marshal(tl.hdr)
	if err != nil {
		// todo: better err
		return nil, err
	}

	// Write serialized header into buffer
	_, err = tl.curBuf.WriteString(xxdkTxLogHeader +
		base64.URLEncoding.EncodeToString(headerMarshal))
	if err != nil {
		// todo: better err
		return nil, err
	}

	// Serialize all transactions
	for i := 0; i < len(tl.txs); i++ {

		// Construct cMix hash
		h, err := hash.NewCMixHash()
		if err != nil {
			// todo: better err, possibly panic
			return nil, err
		}

		// Construct secret for encryption
		h.Write(binary.LittleEndian.AppendUint16(make([]byte, 0), uint16(i)))
		h.Write(tl.deviceSecret)
		secret := h.Sum(nil)

		// Marshal the current transaction
		txMarshal, err := json.Marshal(tl.txs[i])
		if err != nil {
			// todo: better err
			return nil, err
		}

		// Encrypt the current transaction
		encrypted := encrypt(txMarshal, string(secret), tl.rng)

		// Write the encrypted transaction to the buffer
		_, err = tl.curBuf.WriteString(strconv.Itoa(i) + "," +
			base64.URLEncoding.EncodeToString(encrypted))
		if err != nil {
			// todo: better err
			return nil, err
		}

	}

	// Read from the buffer
	dataToSave := make([]byte, 0)
	if _, err := io.ReadFull(tl.curBuf, dataToSave); err != nil {
		// todo: better err
		return nil, err
	}

	return dataToSave, nil
}

// save writes the data within TransactionLog.curBuf to file, both remotely and
// locally.
func (tl *TransactionLog) save(dataToSave []byte) error {

	// Save to local storage (if set)
	if tl.local != nil {
		if err := tl.local.Write(tl.path, dataToSave); err != nil {
			// todo: better err
			return err
		}
	}

	// Save to remote storage (if set)
	if tl.remote != nil {
		return tl.remote.Write(tl.path, dataToSave)
	}

	return nil
}
