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
	"os"
	"syscall/js"
)

const NdfStorageKeyNamePrefix = "ndfStorageKey/"

var localStorage = js.Global().Get("localStorage")

func LoadNDF(_ versioned.KV, key string) (*ndf.NetworkDefinition, error) {
	keyValue := localStorage.Call("getItem", NdfStorageKeyNamePrefix+key)
	if keyValue.IsNull() {
		return nil, os.ErrNotExist
	}

	return ndf.Unmarshal([]byte(keyValue.String()))
}

func SaveNDF(_ versioned.KV, key string, ndf *ndf.NetworkDefinition) error {
	marshaled, err := ndf.Marshal()
	if err != nil {
		return err
	}

	localStorage.Call("setItem",
		NdfStorageKeyNamePrefix+key, string(marshaled))

	return nil
}
