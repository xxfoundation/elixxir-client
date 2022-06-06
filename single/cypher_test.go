////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/large"
	"testing"
)

// Tests that makeCyphers returns a list of cyphers of the correct length and
// that each cypher in the list has the correct DH key and number.
func Test_makeCyphers(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
	messageCount := 16

	cyphers := makeCyphers(dhKey, uint8(messageCount), nil, nil)

	if len(cyphers) != messageCount {
		t.Errorf("Wrong number of cyphers.\nexpected: %d\nreceived: %d",
			messageCount, len(cyphers))
	}

	for i, c := range cyphers {
		if dhKey.Cmp(c.dhKey) != 0 {
			t.Errorf("Cypher #%d has incorrect DH key."+
				"\nexpected: %s\nreceived: %s",
				i, dhKey.Text(10), c.dhKey.Text(10))
		}
		if int(c.num) != i {
			t.Errorf("Cypher #%d has incorrect number."+
				"\nexpected: %d\nreceived: %d", i, i, c.num)
		}
	}
}

// Tests that cypher.getKey returns the expected key from the passed in newKey
// function. Also tests that the expected DH key and number are passed to the
// new key function.
func Test_cypher_getKey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	newKey := func(dhKey *cyclic.Int, keyNum uint64) []byte {
		return []byte(fmt.Sprintf("KEY #%d  %s", keyNum, dhKey.Text(10)))
	}

	c := &cypher{
		dhKey:  diffieHellman.GeneratePublicKey(grp.NewInt(42), grp),
		num:    6,
		newKey: newKey,
	}

	expectedKey := newKey(c.dhKey, uint64(c.num))

	key := c.getKey()
	if !bytes.Equal(expectedKey, key) {
		t.Errorf(
			"Unexpected key.\nexpected: %q\nreceived: %q", expectedKey, key)
	}
}

// Tests that cypher.getFingerprint returns the expected fingerprint from the
// passed in newFp function. Also tests that the expected DH key and number are
// passed to the new fingerprint function.
func Test_cypher_getFingerprint(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	newFp := func(dhKey *cyclic.Int, keyNum uint64) format.Fingerprint {
		return format.NewFingerprint([]byte(
			fmt.Sprintf("FP #%d  %s", keyNum, dhKey.Text(10))))
	}

	c := &cypher{
		dhKey: diffieHellman.GeneratePublicKey(grp.NewInt(42), grp),
		num:   6,
		newFp: newFp,
	}

	expectedFp := newFp(c.dhKey, uint64(c.num))

	fp := c.getFingerprint()
	if expectedFp != fp {
		t.Errorf("Unexpected fingerprint.\nexpected: %s\nreceived: %s",
			expectedFp, fp)
	}
}

// Tests that a payload encrypted by cypher.encrypt and decrypted by
// cypher.decrypt matches the original. Tests with both the response and request
// part key and fingerprint functions.
func Test_cypher_encrypt_decrypt(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	funcs := []struct {
		newKey newKeyFn
		newFp  newFpFn
	}{
		{singleUse.NewResponseKey, singleUse.NewResponseFingerprint},
		{singleUse.NewRequestPartKey, singleUse.NewRequestPartFingerprint},
	}
	c := &cypher{
		dhKey: diffieHellman.GeneratePublicKey(grp.NewInt(42), grp),
		num:   6,
	}

	for i, fn := range funcs {
		c.newKey = fn.newKey
		c.newFp = fn.newFp

		payload := []byte("I am a single-use payload message.")

		_, encryptedPayload, mac := c.encrypt(payload)

		decryptedPayload, err := c.decrypt(encryptedPayload, mac)
		if err != nil {
			t.Errorf("decrypt returned an error (%d): %+v", i, err)
		}

		if !bytes.Equal(payload, decryptedPayload) {
			t.Errorf("Decrypted payload does not match original (%d)."+
				"\nexpected: %q\nreceived: %q", i, payload, decryptedPayload)
		}
	}
}

// Error path: tests that cypher.decrypt returns the expected error when the MAC
// is invalid.
func Test_cypher_decrypt_InvalidMacError(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	c := &cypher{
		dhKey:  diffieHellman.GeneratePublicKey(grp.NewInt(42), grp),
		num:    6,
		newKey: singleUse.NewResponseKey,
		newFp:  singleUse.NewResponseFingerprint,
	}

	_, err := c.decrypt([]byte("contents"), []byte("mac"))
	if err == nil || err.Error() != errMacVerification {
		t.Errorf("decrypt did not return the expected error with invalid MAC."+
			"\nexpected: %s\nreceived: %+v", errMacVerification, err)
	}
}
