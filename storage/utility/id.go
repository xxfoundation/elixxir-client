////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentIDVersion = 0

func StoreID(kv *versioned.KV, sid *id.ID, key string) error {
	now := netTime.Now()

	data, err := sid.MarshalJSON()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentIDVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(key, currentIDVersion, &obj)
}

func LoadID(kv *versioned.KV, key string) (*id.ID, error) {
	vo, err := kv.Get(key, currentIDVersion)
	if err != nil {
		return nil, err
	}

	sid := &id.ID{}

	return sid, sid.UnmarshalJSON(vo.Data)
}

// DeleteCID deletes a given cyclic key from storage
func DeleteCID(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentIDVersion)
}
