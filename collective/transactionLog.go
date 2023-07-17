////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/stoppable"
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

	// FIXME: It should be: [name]-[deviceid]/[keyid]/txlog
	// but we don't have access to a name, so: [deviceid]/[keyid]/txlog
	txLogPathFmt = "%s/%s/state.xx"
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
	state *Patch

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

	uploadPeriod time.Duration

	// tracks if as of the last interaction, we are connected to the
	// remote
	remoteUpToDate *uint32
	*notifier

	mb *mutateBuffer
}

type transaction struct {
	Mutate    map[string]Mutate
	BuffIndex int
}

// newRemoteWriter constructs a remoteWriter object that does
// not Write to a remote.
//
// Parameters:
//   - path - the remote storage prefix path that will be used to
//     write transactions
//   - localFS - a filesystem [FileIO] adhering object which will be
//     used to Write the mutate log to file.
//   - appendCallback - the callback used to report the status of writing to
//     remote.
//   - deviceSecret - the secret for this device to communicate with the others.
//     Note: In the future this will be unique per device.
//   - rng - An io.Reader used for random generation when encrypting data.
func newRemoteWriter(path string, deviceID InstanceID,
	io FileIO, encrypt encryptor, kv ekv.KeyValue) (*remoteWriter, error) {

	// Per spec, the path is: [path] + /[deviceID]/[txlog]
	// we don't use path.join because we aren't relying on OS pathSep.
	logKeyID := encrypt.KeyID(deviceID)
	myPath := getTxLogPath(path, logKeyID, deviceID)

	connected := uint32(0)
	// Construct a new mutate log
	tx := &remoteWriter{
		path:           myPath,
		header:         newHeader(deviceID),
		state:          newPatch(deviceID),
		adds:           make(chan transaction, bufferSize),
		io:             io,
		encrypt:        encrypt,
		kv:             kv,
		localWriteKey:  makeLocalWriteKey(path),
		remoteUpToDate: &connected,
		notifier:       &notifier{},
		uploadPeriod:   defaultUploadPeriod,
	}

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

	//attempt to load stored mutateBuffer and handle any extant data
	mb, remainingMutations := loadBuffer(kv)
	tx.mb = mb
	if len(remainingMutations) > 0 {
		for index := range remainingMutations {
			tx.syncLock.RLock()
			mutations := remainingMutations[index]
			tx.adds <- transaction{
				Mutate:    mutations,
				BuffIndex: index,
			}
		}
	}
	jww.INFO.Printf("[COL] Transaction Log Writer initialized (%s)", myPath)
	return tx, nil
}

// Runner pushes updates to the patch file to the remote
func (rw *remoteWriter) Runner(s *stoppable.Single) {
	jww.INFO.Printf("[SYNC] started transaction log (remoteWriter) thread")

	//always Write to remote when we start in order to ensure that any
	//dropped updates are propogated
	timer := time.NewTimer(time.Nanosecond)
	serial, err := rw.state.Serialize()
	if err != nil {
		jww.FATAL.Panicf("Failed to serialize transaction: %+v", err)
	}
	running := true
	var ts time.Time
	uploadPeriod := rw.uploadPeriod
	for {
		select {
		case t := <-rw.adds:

			for key, mutate := range t.Mutate {
				jww.INFO.Printf("Adding change for %s", key)
				rw.state.AddUnsafe(key, mutate)
			}

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
			rw.syncLock.RUnlock()
			rw.mb.DeleteBufferElement(t.BuffIndex)

			if !running {
				timer = time.NewTimer(rw.uploadPeriod)
				running = true
			}

		case <-timer.C:
			running = false
			encrypted := rw.encrypt.Encrypt(serial)
			file := buildFile(rw.header, encrypted)

			if err = rw.io.Write(rw.path, file); err != nil {
				rw.notify(false)
				uploadPeriod = expBackoff(uploadPeriod)
				jww.ERROR.Printf("Failed to update collective state, "+
					"last update %s, will auto retry in %s: %+v", ts,
					uploadPeriod, err)
				timer = time.NewTimer(rw.uploadPeriod)
				running = true
			} else {
				jww.DEBUG.Printf("Wrote patch %s: %d",
					rw.header.DeviceID, len(rw.state.keys))
				rw.notify(true)
				uploadPeriod = defaultUploadPeriod
				ts = netTime.Now()
				timer.Stop()
				running = false
			}
		case <-s.Quit():
			s.ToStopped()
			return
		}

	}

}

// Write will add a mutate to the remoteWriter to Write the
// key remotely and Write it to disk. This will saveLastMutationTime the
// serialized remoteWriter to local and remote storage. The callback for
// remote storage will be NewOrLoadTransactionLog or SetRemoteCallback.
// this blocks so it cannot be run conncurently with the collector
func (rw *remoteWriter) Write(key string, value []byte) (
	oldData []byte, existed bool, err error) {
	jww.INFO.Printf("[%s] Inserting upsert to remote at %s", logHeader, key)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk
	rw.syncLock.RLock()

	ts := netTime.Now()

	m := Mutate{
		Timestamp: ts.UTC().UnixNano(),
		Value:     value,
		Deletion:  false,
	}

	op := func(files map[string]ekv.Operable, ext ekv.Extender) error {
		wFile := files[key]
		oldData, existed = wFile.Get()
		wFile.Set(value)
		return nil
	}

	mutateMap := msmm(key, m)

	buffIndex, err := rw.mb.DoTransactionAndWriteToBuffer(op, []string{key}, mutateMap)
	if err != nil {
		rw.syncLock.RUnlock()
		return oldData, existed, err
	}

	jww.INFO.Printf("Sending transaction for %s", key)
	rw.adds <- transaction{
		Mutate:    mutateMap,
		BuffIndex: buffIndex,
	}
	return oldData, existed, nil
}

// WriteMap writes to a map, adding the passed in elements and deleting the
// elements designated for deletion.  It will return the old values for all
// inserted and deleted elements
func (rw *remoteWriter) WriteMap(mapName string,
	elements map[string][]byte, toDelete map[string]struct{}) (
	map[string][]byte, error) {
	jww.INFO.Printf("[%s] Inserting upsert to remote for map %s",
		logHeader, mapName)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk
	rw.syncLock.RLock()

	ts := netTime.Now()
	tsInt := ts.UTC().UnixNano()

	//build handling data
	keys := make([]string, 0, len(elements)+1)
	mutates := make(map[string]Mutate, len(elements)+len(toDelete))
	old := make(map[string][]byte, len(elements)+len(toDelete))
	keyConversions := make(map[string]string, len(elements))

	// construct mutates for upserts
	for element := range elements {
		key := versioned.MakeElementKey(mapName, element)
		keys = append(keys, key)
		mutates[key] = Mutate{
			Timestamp: tsInt,
			Value:     elements[element],
			Deletion:  false,
		}
		keyConversions[key] = element
	}

	// construct mutates for deletions
	for element := range toDelete {
		key := versioned.MakeElementKey(mapName, element)
		keys = append(keys, key)
		mutates[key] = Mutate{
			Timestamp: tsInt,
			Value:     nil,
			Deletion:  true,
		}
		keyConversions[key] = element
	}

	//add the map file to the end of the keys list
	mapKey := versioned.MakeMapKey(mapName)
	keys = append(keys, mapKey)

	op := func(files map[string]ekv.Operable, _ ekv.Extender) error {

		// process key map, will always be the last value due to it being
		mapFile := files[mapKey]
		mapFileBytes, _ := mapFile.Get()
		mapSet, err := getMapFile(mapFileBytes, len(old))
		if err != nil {
			return err
		}

		// make edits to the map file and store changes
		for key, mutate := range mutates {
			elementName := keyConversions[key]
			file := files[key]
			old[elementName], _ = file.Get()
			if mutate.Deletion {
				mapSet.Delete(elementName)
				file.Delete()
			} else {
				mapSet.Add(elementName)
				file.Set(mutate.Value)
			}
		}

		// add the updated map file to updates
		mapFileBytesNew, err := json.Marshal(mapSet)
		if err != nil {
			return err
		}

		mapFile.Set(mapFileBytesNew)

		return nil
	}

	//Write to KV
	buffIndex, err := rw.mb.DoTransactionAndWriteToBuffer(op, keys, mutates)
	if err != nil {
		rw.syncLock.RUnlock()
		return nil, err
	}

	// send signals to collective all transactions
	rw.adds <- transaction{mutates, buffIndex}

	return old, err
}

func copyData(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// Delete will add a mutate to the remoteWriter to Delete the
// key remotely and Delete it on disk. This will saveLastMutationTime the
// serialized remoteWriter to local and remote storage. The callback for
// remote storage will be NewOrLoadTransactionLog or SetRemoteCallback.
// this blocks so it cannot be run conncurently with the collector
func (rw *remoteWriter) Delete(key string) (
	oldData []byte, existed bool, err error) {
	jww.INFO.Printf("[%s] Inserting Delete to remote at %s", logHeader, key)
	// do not operate while the collector is collecting. this will
	// be unlocked when the transaction is written to disk
	rw.syncLock.RLock()

	ts := netTime.Now()

	m := Mutate{
		Timestamp: ts.UTC().UnixNano(),
		Value:     nil,
		Deletion:  true,
	}

	op := func(files map[string]ekv.Operable, ext ekv.Extender) error {
		wFile := files[key]
		oldData, existed = wFile.Get()
		wFile.Delete()
		return nil
	}

	mutateMap := msmm(key, m)

	//Write to KV
	buffIndex, err := rw.mb.DoTransactionAndWriteToBuffer(op, []string{key}, mutateMap)
	if err != nil {
		rw.syncLock.RUnlock()
		return oldData, existed, err
	}

	rw.adds <- transaction{
		Mutate:    mutateMap,
		BuffIndex: buffIndex,
	}
	return oldData, existed, nil
}

func (rw *remoteWriter) Read() (patch *Patch, unlock func()) {
	rw.syncLock.Lock()
	unlock = func() {
		rw.syncLock.Unlock()
	}
	patch = rw.state
	return patch, unlock
}

func (rw *remoteWriter) RemoteUpToDate() bool {
	return atomic.LoadUint32(rw.remoteUpToDate) == 1
}

func (rw *remoteWriter) notify(updatedRemote bool) {
	var toWrite uint32
	if updatedRemote {
		toWrite = 1
	} else {
		toWrite = 0
	}
	old := atomic.SwapUint32(rw.remoteUpToDate, toWrite)
	if old != toWrite {
		rw.Notify(updatedRemote)
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

func getTxLogPath(syncPath, keyID string, deviceID InstanceID) string {
	return filepath.Join(syncPath,
		fmt.Sprintf(txLogPathFmt, deviceID, keyID))
}
