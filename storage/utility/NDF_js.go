////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/ndf"
)

const ndfStorageKeyNamePrefix = "ndfStorageKey/"

func LoadNDF(_ *versioned.KV, key string) (*ndf.NetworkDefinition, error) {
	value, err := StateKV.Get(ndfStorageKeyNamePrefix + key)
	if err != nil {
		return nil, err
	}

	return ndf.Unmarshal(value)
}

func SaveNDF(_ versioned.KV, key string, ndf *ndf.NetworkDefinition) error {
	marshaled, err := ndf.Marshal()
	if err != nil {
		return err
	}

	return StateKV.Set(ndfStorageKeyNamePrefix+key, marshaled)
}
