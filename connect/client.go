///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// buildClientAuthRequest is a helper function which constructs a marshalled
// IdentityAuthentication message.
func buildClientAuthRequest(newPartner partner.Manager,
	rng *fastRNG.StreamGenerator, rsaPrivKey *rsa.PrivateKey,
	salt []byte) ([]byte, error) {

	// The connection fingerprint (hashed) will be used as a nonce
	connectionFp := newPartner.ConnectionFingerprint().Bytes()
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Sign the connection fingerprint
	stream := rng.GetStream()
	defer stream.Close()
	signature, err := rsa.Sign(stream, rsaPrivKey,
		opts.Hash, nonce, opts)
	if err != nil {
		return nil, errors.Errorf("failed to sign nonce: %+v", err)
	}

	// Construct message
	pemEncodedRsaPubKey := rsa.CreatePublicKeyPem(rsaPrivKey.GetPublic())
	iar := &IdentityAuthentication{
		Signature: signature,
		RsaPubKey: pemEncodedRsaPubKey,
		Salt:      salt,
	}
	payload, err := proto.Marshal(iar)
	if err != nil {
		return nil, errors.Errorf("failed to marshal identity request "+
			"message: %+v", err)
	}

	return payload, nil
}
