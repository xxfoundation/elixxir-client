////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"io"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// Sign creates a signature authenticating an identity for a connection.
func sign(rng io.Reader, rsaPrivKey rsa.PrivateKey,
	connectionFp []byte) ([]byte, error) {
	// The connection fingerprint (hashed) will be used as a nonce
	opts := getCryptoPSSOpts()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Sign the connection fingerprint
	return rsaPrivKey.SignPSS(rng, opts.Hash, nonce, opts)

}

// Verify takes a signature for an authentication attempt
// and verifies the information.
func verify(partnerId *id.ID, partnerPubKey rsa.PublicKey,
	signature, connectionFp, salt []byte) error {

	// Verify the partner's known ID against the information passed
	// along the wire
	partnerWireId, err := xx.NewID(partnerPubKey, salt, id.User)
	if err != nil {
		return err
	}

	if !partnerId.Cmp(partnerWireId) {
		return errors.New("Failed confirm partner's ID over the wire")
	}

	// Hash the connection fingerprint
	opts := getCryptoPSSOpts()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Verify the signature
	err = partnerPubKey.VerifyPSS(opts.Hash, nonce, signature, opts)
	if err != nil {
		return err
	}

	return nil

}
