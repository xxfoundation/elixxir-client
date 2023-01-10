////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build js && wasm

package nodes

import (
	"io"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
)

func useSHA() bool {
	return false
}

func verifyNodeSignature(certContents string, toBeHashed []byte, sig []byte) error {

	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash

	h := opts.Hash.New()
	h.Write(toBeHashed)
	hashed := h.Sum(nil)

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

	// Verify the response signature
	return rsa.Verify(nodePubKey, opts.Hash, hashed, sig, opts)
}

func signRegistrationRequest(rng io.Reader, toBeHashed []byte, privateKey newRSA.PrivateKey) ([]byte, error) {

	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash

	h := opts.Hash.New()
	h.Write(toBeHashed)
	hashed := h.Sum(nil)

	// Verify the response signature
	return rsa.Sign(rng, privateKey.GetOldRSA(), opts.Hash, hashed, opts)
}
