////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cypher

import (
	"bytes"
	"gitlab.com/elixxir/primitives/format"
	"testing"
)

// Tests that contents that are encrypted with Cypher.Encrypt match the
// decrypted contents of Cypher.Decrypt.
func TestCypher_Encrypt_Decrypt(t *testing.T) {
	m, _ := newTestManager(16, t)
	numPrimeBytes := 512

	// Create contents of the right size
	contents := make([]byte, format.NewMessage(numPrimeBytes).ContentsSize())
	copy(contents, "This is some message contents.")

	c, err := m.PopCypher()
	if err != nil {
		t.Errorf("Failed to pop cypher: %+v", err)
	}

	// Encrypt contents
	cipherText, mac, fp := c.Encrypt(contents)

	// Create message to decrypt
	msg := format.NewMessage(numPrimeBytes)
	msg.SetContents(cipherText)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)

	// Decrypt message
	decryptedContents, err := c.Decrypt(msg)
	if err != nil {
		t.Errorf("Decrypt returned an error: %+v", err)
	}

	// Tests that the decrypted contents match the original
	if !bytes.Equal(contents, decryptedContents) {
		t.Errorf("Decrypted contents do not match original."+
			"\nexpected: %q\nreceived: %q", contents, decryptedContents)
	}
}

// Tests that Cypher.Decrypt returns an error when the contents are the wrong
// size.
func TestCypher_Decrypt_MacError(t *testing.T) {
	m, _ := newTestManager(16, t)

	// Create contents of the wrong size
	contents := []byte("This is some message contents.")

	c, err := m.PopCypher()
	if err != nil {
		t.Errorf("Failed to pop cypher: %+v", err)
	}

	// Encrypt contents
	cipherText, mac, fp := c.Encrypt(contents)

	// Create message to decrypt
	msg := format.NewMessage(512)
	msg.SetContents(cipherText)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)

	// Decrypt message
	_, err = c.Decrypt(msg)
	if err == nil {
		t.Error("Failed to receive an error when the contents are the wrong " +
			"length.")
	}
}

// Tests that Cypher.GetFingerprint returns unique fingerprints.
func TestCypher_GetFingerprint(t *testing.T) {
	m, _ := newTestManager(16, t)
	fpMap := make(map[format.Fingerprint]bool, m.fpVector.GetNumKeys())

	for c, err := m.PopCypher(); err == nil; c, err = m.PopCypher() {
		fp := c.GetFingerprint()

		if fpMap[fp] {
			t.Errorf("Fingerprint %s already exists.", fp)
		} else {
			fpMap[fp] = true
		}
	}
}
