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
	"io"

	"gitlab.com/elixxir/crypto/rsa"
)

func useSHA() bool {
	return true
}

func getHash() func() hash.Hash {
	return crypto.SHA256.New
}

func verifyNodeSignature(certContents string, plaintext []byte, sig []byte) error {
	/*
		opts := rsa.NewDefaultPSSOptions()
		opts.Hash = crypto.SHA256

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
		// fixme: the js version doesnt expect hashed data, so pass it plaintext. make the api the same
		return nodePubKey.VerifyPSS(opts.Hash, plaintext, sig, opts)*/
	return nil
}

func signRegistrationRequest(rng io.Reader, plaintext []byte, privateKey rsa.PrivateKey) ([]byte, error) {

	opts := rsa.NewDefaultPSSOptions()
	opts.Hash = crypto.SHA256

	// Verify the response signature
	// fixme: the js version doesnt expect hashed data, so pass it plaintext. make the api the same
	return privateKey.SignPSS(rng, opts.Hash, plaintext, opts)
}
