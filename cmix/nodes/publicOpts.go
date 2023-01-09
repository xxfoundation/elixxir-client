////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build !js || !wasm

package nodes

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/tls"
	"io"

	"gitlab.com/elixxir/crypto/rsa"
)

func useSHA() bool {
	return false
}

func verifyNodeSignature(certContents string, hashed []byte, sig []byte) error {

	opts := rsa.NewDefaultPSSOptions()

	sch := rsa.GetScheme()

	// Load nodes certificate
	gatewayCert, err := tls.LoadCertificate(certContents)
	if err != nil {
		return errors.Errorf("Unable to load nodes's certificate: %+v", err)
	}

	// Extract public key
	nodePubKeyOld, err := tls.ExtractPublicKey(gatewayCert)
	if err != nil {
		return errors.Errorf("Unable to load node's public key: %v", err)
	}

	nodePubKey := sch.ConvertPublic(&nodePubKeyOld.PublicKey)

	// Verify the response signature
	return nodePubKey.VerifyPSS(opts.Hash, hashed, sig, opts)
}

func signRegistrationRequest(rng io.Reader, hashed []byte, privateKey rsa.PrivateKey) ([]byte, error) {

	opts := rsa.NewDefaultPSSOptions()
	opts.Hash = hash.CMixHash

	// Verify the response signature
	return privateKey.SignPSS(rng, opts.Hash, hashed, opts)
}
