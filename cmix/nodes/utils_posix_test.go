////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// NOTE: download and verify of ndf is not available in wasm.
//go:build !js || !wasm
// +build !js !wasm

package nodes

import (
	"gitlab.com/elixxir/crypto/rsa"
)

func getDefaultPSSOptions() *rsa.PSSOptions {
	opts := rsa.NewDefaultPSSOptions()
	return opts
}
