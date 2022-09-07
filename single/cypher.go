////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	// cypher.decrypt
	errMacVerification = "failed to verify the single-use MAC"
)

type newKeyFn func(dhKey *cyclic.Int, keyNum uint64) []byte
type newFpFn func(dhKey *cyclic.Int, keyNum uint64) format.Fingerprint

// makeCyphers generates all fingerprints for a given number of messages.
func makeCyphers(dhKey *cyclic.Int, messageCount uint8, newKey newKeyFn,
	newFp newFpFn) []cypher {

	cypherList := make([]cypher, messageCount)

	for i := range cypherList {
		cypherList[i] = cypher{
			dhKey:  dhKey,
			num:    uint8(i),
			newKey: newKey,
			newFp:  newFp,
		}
	}

	return cypherList
}

type cypher struct {
	dhKey  *cyclic.Int
	num    uint8
	newKey newKeyFn // Function used to create new key
	newFp  newFpFn  // Function used to create new fingerprint
}

// getKey generates a new encryption/description key from the DH key and number.
func (c *cypher) getKey() []byte {
	return c.newKey(c.dhKey, uint64(c.num))
}

// getFingerprint generates a new key fingerprint from the DH key and number.
func (c *cypher) getFingerprint() format.Fingerprint {
	return c.newFp(c.dhKey, uint64(c.num))
}

// encrypt encrypts the payload.
func (c *cypher) encrypt(
	payload []byte) (fp format.Fingerprint, encryptedPayload, mac []byte) {
	fp = c.getFingerprint()
	key := c.getKey()

	// FIXME: Encryption is identical to what is used by e2e.Crypt, lets make
	//  them the same code path.
	encryptedPayload = cAuth.Crypt(key, fp[:24], payload)
	mac = singleUse.MakeMAC(key, encryptedPayload)

	return fp, encryptedPayload, mac
}

// decrypt decrypts the payload.
func (c *cypher) decrypt(contents, mac []byte) ([]byte, error) {
	fp := c.getFingerprint()
	key := c.getKey()

	// Verify the cMix message MAC
	if !singleUse.VerifyMAC(key, contents, mac) {
		return nil, errors.New(errMacVerification)
	}

	// Decrypt the payload
	return cAuth.Crypt(key, fp[:24], contents), nil
}
