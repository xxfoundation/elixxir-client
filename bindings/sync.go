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

// RemoteStoreReport will contain the
type RemoteStoreReport struct {
	LastModified int64
	LastWrite    int64
	// Data []byte
}

// NewOrLoadRemoteKV
func NewOrLoadRemoteKV(e2eID int, txLogPath string) (*SyncRemoteKV, error) {
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// todo: properly define
	var deviceSecret []byte
	//deviceSecret = e2eCl.GetDeviceSecret()
	var local sync.LocalStore
	var remote sync.RemoteStore
	// todo: some of these should be wrapped
	var upsertCb map[string]sync.UpsertCallback
	var eventCb sync.KeyUpdateCallback
	var updateCb sync.RemoteStoreCallback

	rng := e2eCl.api.GetRng().GetStream()

	//

	txLog, err := sync.NewOrLoadTransactionLog(txLogPath, local, remote,
		deviceSecret, rng)
	if err != nil {
		return nil, err
	}

	rkv, err := sync.NewOrLoadRemoteKV(txLog, e2eCl.api.GetStorage().GetKV(),
		upsertCb, eventCb, updateCb)
	if err != nil {
		return nil, err
	}

	return &SyncRemoteKV{rkv: rkv}, nil
}

func (s *SyncRemoteKV) Write(path string, data []byte) error {
	var updateCb sync.RemoteStoreCallback
	return s.rkv.Set(path, data, updateCb)
}

func (s *SyncRemoteKV) Read(path string) ([]byte, error) {
	return s.rkv.Get(path)
}

func (s *SyncRemoteKV) GetLastModified(path string) ([]byte, error) {
	ts, err := s.rkv.TxLog.Remote.GetLastModified(path)
	if err != nil {
		return 0, err
	}

	rsr := &RemoteStoreReport{
		LastModified: ts.UnixNano(),
	}

	return json.Marshal(rsr)
}

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
