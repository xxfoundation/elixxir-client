////////////////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                                       //
//                                                                                        //
// Use of this source code is governed by a license that can be found in the LICENSE file //
////////////////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"bytes"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

var expectedSig = []byte{139, 67, 63, 6, 185, 76, 60, 217, 163, 84, 251, 231,
	197, 6, 33, 179, 53, 66, 88, 75, 105, 191, 16, 71, 126, 4, 16, 11, 41,
	237, 34, 245, 242, 97, 44, 58, 154, 120, 58, 235, 240, 140, 223, 80, 232,
	51, 94, 247, 226, 217, 79, 194, 215, 46, 187, 157, 55, 167, 180, 179, 12,
	228, 205, 98, 132, 200, 146, 180, 142, 0, 230, 79, 0, 129, 39, 205, 67,
	79, 252, 62, 187, 125, 130, 232, 125, 41, 99, 63, 106, 79, 234, 131, 109,
	103, 189, 149, 45, 169, 227, 85, 164, 121, 103, 254, 19, 224, 236, 28, 187,
	38, 240, 132, 192, 227, 145, 140, 56, 196, 91, 48, 228, 242, 123, 142, 123,
	221, 159, 160}

type CountingReader struct {
	count uint8
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
}

func TestSignVerify_Consistency(t *testing.T) {
	// use insecure seeded rng to reproduce key
	notRand := &CountingReader{count: uint8(0)}

	privKey, err := rsa.GenerateKey(notRand, 1024)
	if err != nil {
		t.Fatalf("SignVerify error: "+
			"Could not generate key: %v", err.Error())
	}

	connFp := []byte("connFp")

	signature, err := Sign(notRand, privKey, connFp)
	if err != nil {
		t.Logf("Sign error: %v", err)
	}

	salt := make([]byte, 32)
	copy(salt, "salt")

	partnerId, err := xx.NewID(privKey.GetPublic(), salt, id.User)
	if err != nil {
		t.Fatalf("NewId error: %v", err)
	}

	pubKeyPem := rsa.CreatePublicKeyPem(privKey.GetPublic())

	err = Verify(partnerId, signature, connFp, pubKeyPem, salt)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}

	if !bytes.Equal(signature, expectedSig) {
		t.Errorf("Consistency test failed."+
			"\nExpected: %v"+
			"\nReceived: %v", expectedSig, signature)
	}
}
