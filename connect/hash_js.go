////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"crypto"
	"gitlab.com/elixxir/crypto/rsa"
)

// getCryptoPSSOpts returns the default pss options for signing/verifying
// when compiled for javascript, use sha256 instead of the default hash
func getCryptoPSSOpts() *rsa.PSSOptions {
	opts := rsa.NewDefaultPSSOptions()
	opts.Hash = crypto.SHA256
	return opts
}
