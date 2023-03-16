////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/sync"
)

////////////////////////////////////////////////////////////////////////////////
// RemoteKV Methods                                                           //
////////////////////////////////////////////////////////////////////////////////

// SyncRemoteKV implements a remote KV to handle transaction logs. These will
// write and read state data from another device to a remote storage interface.
type SyncRemoteKV struct {
	rkv *sync.RemoteKV
}

// RemoteStoreReport will contain the data from the remote storage interface.
type RemoteStoreReport struct {
	// LastModified is the timestamp (in ns) of the last time the specific path
	// was modified. Refer to SyncRemoteKV.GetLastModified.
	LastModified int64

	// LastWrite is the timestamp (in ns) of the last write to the remote
	// storage interface by any device. Refer to SyncRemoteKV.GetLastWrite.
	LastWrite int64
	// Data []byte
}

type KeyUpdateCallback interface {
	Callback(key, val string)
}

type RemoteStoreCallback interface {
	Callback(newTx []byte, err string)
}

// NewOrLoadSyncRemoteKV will construct a SyncRemoteKV.
//
// Parameters:
//   - e2eID - ID of the e2e object in the tracker.
//   - txLogPath - the path that the state data for this device will be written to
//     locally (e.g. sync/txLog.txt)
func NewOrLoadSyncRemoteKV(e2eID int, txLogPath string,
	keyUpdateCb KeyUpdateCallback, remoteStoreCb RemoteStoreCallback,
	upsertCbKeys []string) (*SyncRemoteKV, error) {
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	rng := e2eCl.api.GetRng().GetStream()

	// todo: properly define
	var deviceSecret []byte
	//deviceSecret = e2eCl.GetDeviceSecret()

	// todo: some of these should be wrapped.
	var local sync.LocalStore
	var remote sync.RemoteStore

	// todo: How to do this one?
	var upsertCb map[string]sync.UpsertCallback

	// Construct the key update CB
	var eventCb sync.KeyUpdateCallback = func(k, v string) {
		keyUpdateCb.Callback(k, v)
	}
	// Construct update CB
	var updateCb = func(newTx sync.Transaction, err error) {
		if err != nil {
			remoteStoreCb.Callback(nil, err.Error())
		}

		serialized, err := newTx.MarshalJSON()
		if err != nil {
			remoteStoreCb.Callback(nil, err.Error())
		}

		remoteStoreCb.Callback(serialized, "")
	}

	// Construct or load a transaction loc
	txLog, err := sync.NewOrLoadTransactionLog(txLogPath, local, remote,
		deviceSecret, rng)
	if err != nil {
		return nil, err
	}

	// Construct remote KV
	rkv, err := sync.NewOrLoadRemoteKV(txLog, e2eCl.api.GetStorage().GetKV(),
		upsertCb, eventCb, updateCb)
	if err != nil {
		return nil, err
	}

	return &SyncRemoteKV{rkv: rkv}, nil
}

// Write will write a transaction to the remote and local store.
func (s *SyncRemoteKV) Write(path string, data []byte) error {
	var updateCb sync.RemoteStoreCallback
	return s.rkv.Set(path, data, updateCb)
}

// Read retrieves the data stored in the underlying kv. Will return an error
// if the data at this key cannot be retrieved.
func (s *SyncRemoteKV) Read(path string) ([]byte, error) {
	return s.rkv.Get(path)
}

// GetLastModified will return when the file at the given file path was last
// modified. If the implementation that adheres to this interface does not
// support this, Write or Read should be implemented to either write a
// separate timestamp file or add a prefix.
func (s *SyncRemoteKV) GetLastModified(path string) ([]byte, error) {
	ts, err := s.rkv.TxLog.Remote.GetLastModified(path)
	if err != nil {
		return nil, err
	}

	rsr := &RemoteStoreReport{
		LastModified: ts.UnixNano(),
	}

	return json.Marshal(rsr)
}

// GetLastWrite will retrieve the most recent successful write operation
// that was received by RemoteStore.
func (s *SyncRemoteKV) GetLastWrite() ([]byte, error) {
	ts, err := s.rkv.TxLog.Remote.GetLastWrite()
	if err != nil {
		return nil, err
	}

	rsr := &RemoteStoreReport{
		LastWrite: ts.UnixNano(),
	}

	return json.Marshal(rsr)
}
