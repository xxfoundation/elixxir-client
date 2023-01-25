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
	cHash "gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/tls"
	"hash"
	"io"

	"gitlab.com/elixxir/crypto/rsa"
)

func useSHA() bool {
	return false
}

func getHash() func() hash.Hash {
	return cHash.CMixHash.New
}

func verifyNodeSignature(certContents string, toBeHashed []byte, sig []byte) error {

	opts := rsa.NewDefaultPSSOptions()

	sch := rsa.GetScheme()

	h := opts.Hash.New()
	h.Write(toBeHashed)
	hashed := h.Sum(nil)

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

func signRegistrationRequest(rng io.Reader, toBeHashed []byte, privateKey rsa.PrivateKey) ([]byte, error) {

	opts := rsa.NewDefaultPSSOptions()
	opts.Hash = cHash.CMixHash

	h := opts.Hash.New()
	h.Write(toBeHashed)
	hashed := h.Sum(nil)

	// Verify the response signature
	return privateKey.SignPSS(rng, opts.Hash, hashed, opts)
}
