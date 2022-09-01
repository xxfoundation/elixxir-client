///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentGroupVersion = 0

func StoreGroup(kv *versioned.KV, grp *cyclic.Group, key string) error {
	now := netTime.Now()

	data, err := grp.GobEncode()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentGroupVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(key, &obj)
}

func LoadGroup(kv *versioned.KV, key string) (*cyclic.Group, error) {
	vo, err := kv.Get(key, currentGroupVersion)
	if err != nil {
		return nil, err
	}

	grp := &cyclic.Group{}

	return grp, grp.GobDecode(vo.Data)
}
