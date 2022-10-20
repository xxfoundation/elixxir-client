////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build js && wasm

package nodes

import (
	"crypto"

	"gitlab.com/xx_network/crypto/signature/rsa"
)

func verifyNodeSignature(pub *rsa.PublicKey, hash crypto.Hash,
	hashed []byte, sig []byte, opts *rsa.Options) error {
	jww.WARN.Printf("node signature checking disabled for wasm")
	return nil
}
