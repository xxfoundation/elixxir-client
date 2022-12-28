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

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

func verifyNodeSignature(certContents string, hash crypto.Hash,
	hashed []byte, sig []byte, opts *rsa.Options) error {

	// Load nodes certificate
	gatewayCert, err := tls.LoadCertificate(certContents)
	if err != nil {
		return errors.Errorf("Unable to load nodes's certificate: %+v", err)
	}

	// Extract public key
	nodePubKey, err := tls.ExtractPublicKey(gatewayCert)
	if err != nil {
		return errors.Errorf("Unable to load node's public key: %v", err)
	}

	startVerify := time.Now()
	// Verify the response signature
	err := rsa.Verify(nodePubKey, hash, hashed, sig, opts)

	jww.INFO.Printf("Signature verification took: %s", time.Since(startVerify))

	return err
}
