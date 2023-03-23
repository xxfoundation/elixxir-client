////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is compiled for all architectures except WebAssembly.
//go:build !js || !wasm

package utility

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentNDFVersion = 0

func LoadNDF(kv *KV, key string) (*ndf.NetworkDefinition, error) {
	data, err := kv.Get(key, currentNDFVersion)
	if err != nil {
		return nil, err
	}

	netDef, err := ndf.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return netDef, err
}

func SaveNDF(kv *KV, key string, ndf *ndf.NetworkDefinition) error {
	marshaled, err := ndf.Marshal()
	if err != nil {
		return err
	}

	now := netTime.Now()

	obj := &versioned.Object{
		Version:   currentNDFVersion,
		Timestamp: now,
		Data:      marshaled,
	}

	return kv.Set(key, obj.Marshal())
}
