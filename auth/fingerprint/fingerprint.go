////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fingerprint

import (
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
)

const (
	NegotiationFingerprintLen = 32
)

// CreateNegotiationFingerprint creates a fingerprint for a re-authentication
// negotiation from the partner's DH public key and SIDH public key.
func CreateNegotiationFingerprint(partnerDhPubKey *cyclic.Int,
	partnerPQPubKey nike.PublicKey) []byte {
	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf(
			"Could not get hash to make request fingerprint: %+v", err)
	}

	h.Write(partnerDhPubKey.Bytes())
	h.Write(partnerPQPubKey.Bytes())

	return h.Sum(nil)[:NegotiationFingerprintLen]
}
