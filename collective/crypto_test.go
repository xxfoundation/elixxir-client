////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"crypto/rand"
	"testing"
)

// TestCrypto smoke tests the crypto helper functions
func TestCrypto(t *testing.T) {
	plaintext := []byte("Hello, World!")
	password := "test_password"
	ciphertext := encrypt(plaintext, password, rand.Reader)
	decrypted, err := decrypt(ciphertext, password)
	if err != nil {
		t.Errorf("%+v", err)
	}

	for i := 0; i < len(plaintext); i++ {
		if plaintext[i] != decrypted[i] {
			t.Errorf("%b != %b", plaintext[i], decrypted[i])
		}
	}
}

// TestShortData tests that the decrypt function does not panic when given
// too little data.
func TestShortData(t *testing.T) {
	// Anything under 24 should cause an error.
	ciphertext := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0}
	_, err := decrypt(ciphertext, "dummypassword")
	if err == nil {
		t.Errorf("Expected error on short decryption")
	}
	expectedErrMsg := "Read 24 bytes, too short to decrypt"
	if err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Unexpected error: %+v", err)
	}

	// Empty string shouldn't panic should cause an error.
	ciphertext = []byte{}
	_, err = decrypt(ciphertext, "dummypassword")
	if err == nil {
		t.Errorf("Expected error on short decryption")
	}
	expectedErrMsg = "Read 0 bytes, too short to decrypt"
	if err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Unexpected error: %+v", err)
	}
}
