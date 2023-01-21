////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

package nodes

import (
	"gitlab.com/elixxir/crypto/rsa"
)

func getDefaultPSSOptions() *rsa.PSSOptions {
	opts := rsa.NewDefaultPSSOptions()
	return opts
}
