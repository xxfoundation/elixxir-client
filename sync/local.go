////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	ekvLocalStoreVersion = 0
	ekvLocalStorePrefix  = "sync/LocalKV"
)

// EkvLocalStore is a structure adhering to LocalStore. This utilizes
// [versioned.KV] file IO operations.
type EkvLocalStore struct {
	data *versioned.KV
}

// NewEkvLocalStore is a constructor for EkvLocalStore.
func NewEkvLocalStore(kv *versioned.KV) *EkvLocalStore {
	return &EkvLocalStore{
		data: kv.Prefix(ekvLocalStorePrefix),
	}
}

// Read reads data from path. This will return an error if it fails to read from
// the file path.
//
// This utilizes [ekv.KeyValue] under the hood.
func (ls *EkvLocalStore) Read(path string) ([]byte, error) {
	obj, err := ls.data.Get(path, ekvLocalStoreVersion)
	if err != nil {
		return nil, err
	}
	return obj.Data, nil
}

// Write writes data to the path. This will return an error if it fails to
// write.
//
// This utilizes [ekv.KeyValue] under the hood.
func (ls *EkvLocalStore) Write(path string, data []byte) error {
	return ls.data.Set(path, &versioned.Object{
		Version:   ekvLocalStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}
