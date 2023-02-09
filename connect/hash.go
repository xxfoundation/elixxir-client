////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

package connect

import (
	"gitlab.com/elixxir/crypto/rsa"
)

// getCryptoPSSOpts returns the default pss options for signing/verifying
func getCryptoPSSOpts() *rsa.PSSOptions {
	return rsa.NewDefaultPSSOptions()
}
