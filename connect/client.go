////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
)

// buildClientAuthRequest is a helper function which constructs a marshalled
// IdentityAuthentication message.
func buildClientAuthRequest(newPartner partner.Manager,
	rng *fastRNG.StreamGenerator, rsaPrivKey rsa.PrivateKey,
	salt []byte) ([]byte, error) {

	// Create signature
	connectionFp := newPartner.ConnectionFingerprint().Bytes()
	stream := rng.GetStream()
	defer stream.Close()
	signature, err := sign(stream, rsaPrivKey, connectionFp)

	// Construct message
	pemEncodedRsaPubKey := rsaPrivKey.Public().MarshalPem()
	iar := &IdentityAuthentication{
		Signature: signature,
		RsaPubKey: pemEncodedRsaPubKey,
		Salt:      salt,
	}

	// Marshal message
	payload, err := proto.Marshal(iar)
	if err != nil {
		return nil, errors.Errorf("failed to marshal identity request "+
			"message: %+v", err)
	}
	return payload, nil
}
