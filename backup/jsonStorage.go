////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	jsonStorageVersion = 0
	jsonStorageKey     = "JsonStorage"
)

func storeJson(json string, kv *versioned.KV) error {
	obj := &versioned.Object{
		Version:   jsonStorageVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(json),
	}

	return kv.Set(jsonStorageKey, obj)
}

func loadJson(kv *versioned.KV) string {
	obj, err := kv.Get(jsonStorageKey, jsonStorageVersion)
	if err != nil {
		return ""
	}

	return string(obj.Data)
}
