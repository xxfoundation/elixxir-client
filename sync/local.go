// //////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//
//	//
//
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
// //////////////////////////////////////////////////////////////////////////////
package sync

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	ekvLocalStoreVersion = 0
)

// EkvLocalStore is a structure adhering to LocalStore. This utilizes
// versioned.KV file IO operations.
type EkvLocalStore struct {
	data *versioned.KV
}

// NewEkvLocalStore is a constructor for EkvLocalStore.
func NewEkvLocalStore(baseDir, password string) (*EkvLocalStore, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	if err != nil {
		return nil, err
	}
	return &EkvLocalStore{
		data: versioned.NewKV(fs),
	}, nil
}

// Read reads data from path. This will return an error if it fails to read
// from the file path.
//
// This utilizes ekv.KeyValue under the hood.
func (ls *EkvLocalStore) Read(path string) ([]byte, error) {
	obj := &versioned.Object{
		Version: ekvLocalStoreVersion,
	}
	obj, err := ls.data.Get(path, ekvLocalStoreVersion)
	if err != nil {
		return nil, err
	}
	return obj.Data, nil
}

// Write will write data to path. This will return an error if it fails to
// write.
//
// This utilizes ekv.KeyValue under the hood.
func (ls *EkvLocalStore) Write(path string, data []byte) error {
	return ls.data.Set(path, &versioned.Object{
		Version:   ekvLocalStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}
