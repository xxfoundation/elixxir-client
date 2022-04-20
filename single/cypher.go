///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/crypto/cyclic"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
)

// makeCyphers generates all fingerprints for a given number of messages.
func makeCyphers(dhKey *cyclic.Int, messageCount uint8) []cypher {

	cypherList := make([]cypher, messageCount)

	for i := uint8(0); i < messageCount; i++ {
		cypherList[i] = cypher{
			dhKey: dhKey,
			num:   i,
		}
	}

	return cypherList
}

type cypher struct {
	dhKey *cyclic.Int
	num   uint8
}

func (rk *cypher) getKey() []byte {
	return singleUse.NewResponseKey(rk.dhKey, uint64(rk.num))
}

func (rk *cypher) GetFingerprint() format.Fingerprint {
	return singleUse.NewResponseFingerprint(rk.dhKey, uint64(rk.num))
}

func (rk *cypher) Encrypt(rp message.ResponsePart) (
	fp format.Fingerprint, encryptedPayload, mac []byte) {
	fp = rk.GetFingerprint()
	key := rk.getKey()
	// FIXME: Encryption is identical to what is used by e2e.Crypt, lets make
	//  them the same code path.
	encryptedPayload = cAuth.Crypt(key, fp[:24], rp.Marshal())
	mac = singleUse.MakeMAC(key, encryptedPayload)
	return fp, encryptedPayload, mac
}

func (rk *cypher) Decrypt(contents, mac []byte) ([]byte, error) {
	fp := rk.GetFingerprint()
	key := rk.getKey()

	// Verify the cMix message MAC
	if !singleUse.VerifyMAC(key, contents, mac) {
		return nil, errors.New("failed to verify the single-use MAC")
	}

	// Decrypt the payload
	return cAuth.Crypt(key, fp[:24], contents), nil
}
