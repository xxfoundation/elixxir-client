////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	jsonStorageVersion = 0
	jsonStorageKey     = "JsonStorage"
)

func storeJson(json string, kv *utility.KV) error {
	obj := &versioned.Object{
		Version:   jsonStorageVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(json),
	}

	return kv.Set(jsonStorageKey, obj.Marshal())
}

func loadJson(kv *utility.KV) string {
	backupJsonData, err := kv.Get(jsonStorageKey, jsonStorageVersion)
	if err != nil {
		return ""
	}

	return string(backupJsonData)
}
