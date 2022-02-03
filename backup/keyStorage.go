////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	userKeyVersion    = 0
	userKeyStorageKey = "BackupKey"
)

// storeKey saves the user's backup key to storage.
func storeKey(key []byte, kv *versioned.KV) error {
	obj := &versioned.Object{
		Version:   userKeyVersion,
		Timestamp: netTime.Now(),
		Data:      key,
	}

	return kv.Set(userKeyStorageKey, userKeyVersion, obj)
}

// loadKey returns the user's backup key from storage.
func loadKey(kv *versioned.KV) ([]byte, error) {
	obj, err := kv.Get(userKeyStorageKey, userKeyVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// deleteKey deletes the user's backup key from storage.
func deleteKey(kv *versioned.KV) error {
	return kv.Delete(userKeyStorageKey, userKeyVersion)
}
