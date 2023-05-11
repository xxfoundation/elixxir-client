////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// remoteWriter constants.
const (
	// The prefix for a serialized header.
	xxdkTxLogHeader = "XXDKTXLOGHDR"

	// The delimiter for a serialized mutate.
	xxdkTxLogDelim = ","

	// The header for the jww log print.
	logHeader = "Mutate Log"

	toDiskKeyName = "TransactionLog_"

	defaultUploadPeriod = synchronizationEpoch
)

// Error messages.
const (
	getLastWriteErr           = "failed to get last Write operation from remote store: %+v"
	writeToStoreErr           = "failed to Write to local store: %+v"
	loadFromLocalStoreErr     = "failed to deserialize log from local store at path %s: %+v"
	deserializeTransactionErr = "failed to deserialize mutate (%d/%d): %+v"
)

// remoteWriter will log all Mutate's to a storage interface. It will
// contain all Mutate's in an ordered list, and will ensure to retain order
// when Append is called. This will store to a LocalStore and a RemoteStore when
// appending Mutate's.
type remoteWriter struct {
	// path is the filepath that the remoteWriter will be written to on remote
	// and local storage.
	path string

	// header is the header of the remoteWriter.
	header *header

	// txs is a list of transactions. This list must always be ordered by
	// timestamp.
	state Patch

	//channel over which writes started localy are processed
	adds chan transaction

	// call to Write to remote
	io FileIO

	// interface to encrypt and decrypt patch files
	encrypt encryptor

	// kv store
	kv            ekv.KeyValue
	localWriteKey string

	// exclusion mutex which ensures writes and deletes do not occur
	// while the collector is running
	syncLock sync.RWMutex

	// tracks if as of the last interaction, we are connected to the
	// remote
	remoteUpToDate *uint32
	*notifier
}

type transaction struct {
	Mutate
	Key string
}

// newRemoteWriter constructs a remoteWriter object that does
// not Write to a remote.
//
// Parameters:
//   - path - the file path that will be used to Write transactions, both locally
//     and remotely.
//   - localFS - a filesystem [FileIO] adhering object which will be
//     used to Write the mutate log to file.
//   - appendCallback - the callback used to report the status of writing to
//     remote.
//   - deviceSecret - the secret for this device to communicate with the others.
//     Note: In the future this will be unique per device.
//   - rng - An io.Reader used for random generation when encrypting data.
func newRemoteWriter(path string, deviceID InstanceID,
	io FileIO, encrypt encryptor, kv ekv.KeyValue) (*remoteWriter, error) {

	connected := uint32(0)
	// Construct a new mutate log
	tx := &remoteWriter{
		path:           path,
		header:         newHeader(deviceID),
		state:          Patch{},
		adds:           make(chan transaction, 1000),
		io:             io,
		encrypt:        encrypt,
		kv:             kv,
		localWriteKey:  makeLocalWriteKey(path),
		remoteUpToDate: &connected,
	}

	tx.notifier = &notifier{}

	// Attempt to Read stored mutate log
	data, err := tx.kv.GetBytes(tx.localWriteKey)
	if err == nil {
		// If data has been Read, attempt to deserialize
		if err = tx.state.Deserialize(data); err != nil {
			return nil, errors.Errorf(loadFromLocalStoreErr, path,
				err)
		}
	} else {
		jww.WARN.Printf("No transaction log found, creating a new one")
	}

	return tx, nil
}

// Runner pushes updates to the patch file to the remote
func (rw *remoteWriter) Runner(s *stoppable.Single) {
	//always Write to remote when we start in order to ensure that any
	//dropped updates are propogated
	timer := time.NewTimer(time.Nanosecond)
	serial, err := rw.state.Serialize()
	if err != nil {
		jww.FATAL.Panicf("Failed to serialize transaction: %+v", err)
	}
	running := true
	var ts time.Time
	uploadPeriod := defaultUploadPeriod
	for {
		select {
		case t := <-rw.adds:
			rw.state.AddUnsafe(t.Key, &t.Mutate)

			// batch writes
			counter := 5 * time.Millisecond
			timer2 := time.NewTimer(counter)
			quit := false
		batch:
			for {
				select {
				case t = <-rw.adds:
					rw.state.AddUnsafe(t.Key, &t.Mutate)
					rw.syncLock.RUnlock()
					counter -= 100 * time.Microsecond
					if counter == 0 {
						break batch
					}
					timer2.Reset(counter)
				case <-timer2.C:
					break batch
				case <-s.Quit():
					quit = true
				}
			}

			// once all have been added, unlock allowing the collector
			// to continue
			rw.syncLock.RUnlock()

			// Write to disk and queue the remote Write
			serial, err = rw.state.Serialize()
			if err != nil {
				jww.FATAL.Panicf("failed to serialize transaction "+
					"log: %+v", err)
			}

			if err = rw.kv.SetBytes(rw.localWriteKey, serial); err != nil {
				jww.FATAL.Panicf("failed to Write transaction "+
					"log to disk: %+v", err)
			}

			if quit == true {
				s.ToStopped()
				return
			}
			if running == false {
				timer.Reset(defaultUploadPeriod)
				running = true
			}

		case <-timer.C:
			running = false
			encrypted := rw.encrypt.Encrypt(serial)
			file := buildFile(rw.header, encrypted)

			if err = rw.io.Write(rw.path, file); err != nil {
				rw.notify(true)
				uploadPeriod = expBackoff(uploadPeriod)
				jww.ERROR.Printf("Failed to update collective state, "+
					"last update %s, will auto retry in %s: %+v", ts,
					uploadPeriod, err)
				timer.Reset(uploadPeriod)
				running = true
			} else {
				rw.notify(false)
				uploadPeriod = defaultUploadPeriod
				ts = netTime.Now()
				timer.Stop()
				running = false
			}
		case <-s.Quit():
			s.ToStopped()
		}

	}

}

// Write will add a mutate to the remoteWriter to Write the
// key remotely and Write it to disk. This will saveLastMutationTime the
// serialized remoteWriter to local and remote storage. The callback for
// remote storage will be NewOrLoadTransactionLog or SetRemoteCallback.
// this blocks so it cannot be run conncurently with the collector
func (rw *remoteWriter) Write(key string, value []byte) error {
	jww.INFO.Printf("[%s] Inserting upsert to remote at %s", logHeader, key)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk
	rw.syncLock.RLock()

	ts := netTime.Now()

	//Write to KV
	err := rw.kv.SetBytes(key, value)
	if err != nil {
		rw.syncLock.RUnlock()
		return err
	}

	rw.adds <- transaction{
		Mutate{
			Timestamp: ts.UTC().UnixNano(),
			Value:     value,
			Deletion:  false,
		},
		key,
	}
	return nil
}

// WriteMap
func (rw *remoteWriter) WriteMap(mapName string,
	elements map[string][]byte, toDelete map[string]struct{}) error {
	jww.INFO.Printf("[%s] Inserting upsert to remote for map %s",
		logHeader, mapName)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk

	ts := netTime.Now()
	tsInt := ts.UTC().UnixNano()

	mapKey := versioned.MakeMapKey(mapName)
	keys := make([]string, 0, len(elements)+1)
	updates := make(map[string]ekv.Value, len(elements)+len(toDelete)+1)
	mutates := make(map[string]Mutate, len(elements)+len(toDelete))
	for element := range elements {
		key := versioned.MakeElementKey(mapName, element)
		rw.syncLock.RLock()
		keys = append(keys, key)
		v := elements[element]
		updates[key] = ekv.Value{
			Data:   v,
			Exists: true,
		}
		mutates[key] = Mutate{
			Timestamp: tsInt,
			Value:     v,
			Deletion:  false,
		}

	}
	for element := range toDelete {
		key := versioned.MakeElementKey(mapName, element)
		rw.syncLock.RLock()
		keys = append(keys, key)
		updates[key] = ekv.Value{
			Exists: false,
		}
		mutates[key] = Mutate{
			Timestamp: tsInt,
			Value:     nil,
			Deletion:  true,
		}
	}
	keys = append(keys, mapKey)

	op := func(old map[string]ekv.Value) (map[string]ekv.Value, error) {

		// process key map, will always be the last value due to it being
		mapFile, err := getMapFile(old[mapKey], len(old)-1)
		if err != nil {
			return nil, err
		}

		// ensure all elements are in the map file
		for key := range elements {
			mapFile.Add(key)
		}

		//remove all deletions from the map file
		for key := range toDelete {
			mapFile.Delete(key)
		}

		// add the updated map file to updates
		mapFileValue, err := json.Marshal(mapFile)
		if err != nil {
			return nil, err
		}

		updates[mapKey] = ekv.Value{
			Data:   mapFileValue,
			Exists: true,
		}

		return updates, nil
	}

	//Write to KV
	_, _, err := rw.kv.MutualTransaction(keys, op)
	if err != nil && !strings.Contains(err.Error(), ekv.ErrDeletesFailed) {
		for i := 0; i < len(elements)+len(toDelete); i++ {
			rw.syncLock.RUnlock()
		}
		return err
	}

	// send signals to collective all transactions
	for key, m := range mutates {
		rw.adds <- transaction{m, key}
	}

	return nil
}

// Delete will add a mutate to the remoteWriter to Delete the
// key remotely and Delete it on disk. This will saveLastMutationTime the
// serialized remoteWriter to local and remote storage. The callback for
// remote storage will be NewOrLoadTransactionLog or SetRemoteCallback.
// this blocks so it cannot be run conncurently with the collector
func (rw *remoteWriter) Delete(key string) error {
	jww.INFO.Printf("[%s] Inserting Delete to remote at %s", logHeader, key)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk
	rw.syncLock.RLock()

	ts := netTime.Now()

	//Write to KV
	err := rw.kv.Delete(key)
	if err != nil {
		rw.syncLock.RUnlock()
		return err
	}

	rw.adds <- transaction{
		Mutate{
			Timestamp: ts.UTC().UnixNano(),
			Value:     nil,
			Deletion:  true,
		},
		key,
	}
	return nil
}

func (rw *remoteWriter) Read() (patch *Patch, unlock func()) {
	rw.syncLock.Lock()
	unlock = func() {
		rw.syncLock.Unlock()
	}
	patch = &rw.state
	return patch, unlock
}

func (rw *remoteWriter) RemoteUpToDate() bool {
	return atomic.LoadUint32(rw.remoteUpToDate) == 1
}

func (rw *remoteWriter) notify(state bool) {
	var toWrite uint32
	if state {
		toWrite = 1
	} else {
		toWrite = 0
	}
	old := atomic.SwapUint32(rw.remoteUpToDate, toWrite)
	if old != toWrite {
		rw.Notify(state)
	}
}

func makeLocalWriteKey(path string) string {
	return toDiskKeyName + path
}

func expBackoff(timeout time.Duration) time.Duration {
	timeout = (timeout * 3) / 2
	if timeout > 5*time.Minute {
		return 5 * time.Minute
	}
	return timeout
}
