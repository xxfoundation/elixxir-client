////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"bytes"
	"testing"

	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// NOTE: there are 2 signatures to deal with race condition-styled behaviors
// added in recent versions of go. Basically one or the other of the following
// will be generated.
var expectedSig1 = []byte{139, 67, 63, 6, 185, 76, 60, 217, 163, 84,
	251, 231, 197, 6, 33, 179, 53, 66, 88, 75, 105, 191, 16, 71,
	126, 4, 16, 11, 41, 237, 34, 245, 242, 97, 44, 58, 154, 120,
	58, 235, 240, 140, 223, 80, 232, 51, 94, 247, 226, 217, 79,
	194, 215, 46, 187, 157, 55, 167, 180, 179, 12, 228, 205, 98,
	132, 200, 146, 180, 142, 0, 230, 79, 0, 129, 39, 205, 67, 79,
	252, 62, 187, 125, 130, 232, 125, 41, 99, 63, 106, 79, 234,
	131, 109, 103, 189, 149, 45, 169, 227, 85, 164, 121, 103, 254,
	19, 224, 236, 28, 187, 38, 240, 132, 192, 227, 145, 140, 56,
	196, 91, 48, 228, 242, 123, 142, 123, 221, 159, 160}

var expectedSig2 = []byte{187, 204, 247, 50, 98, 78, 28, 104, 15, 123,
	40, 138, 202, 195, 4, 176, 246, 11, 97, 148, 47, 134, 15, 25, 97, 196,
	88, 207, 85, 5, 149, 140, 47, 106, 89, 19, 19, 18, 209, 205, 163, 177,
	176, 246, 237, 215, 242, 199, 69, 26, 47, 124, 212, 115, 102, 59, 214,
	181, 22, 76, 43, 134, 136, 158, 39, 47, 107, 182, 169, 102, 201, 205,
	224, 220, 245, 125, 244, 19, 104, 187, 239, 194, 243, 172, 82, 31,
	135, 254, 80, 54, 147, 249, 209, 240, 79, 91, 83, 183, 247, 203, 96,
	135, 69, 250, 79, 129, 234, 70, 215, 98, 65, 182, 112, 31, 53, 254,
	18, 139, 11, 188, 247, 235, 236, 61, 30, 21, 164, 128}

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

	signature, err := sign(notRand, privKey, connFp)
	if err != nil {
		t.Logf("Sign error: %v", err)
	}

	salt := make([]byte, 32)
	copy(salt, "salt")

	partnerId, err := xx.NewID(privKey.GetPublic(), salt, id.User)
	if err != nil {
		t.Fatalf("NewId error: %v", err)
	}

	err = verify(partnerId, privKey.GetPublic(), signature, connFp, salt)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}

	if !bytes.Equal(signature, expectedSig1) &&
		!bytes.Equal(signature, expectedSig2) {
		t.Errorf("Consistency test failed."+
			"\nExpected1: %v\nExpected2: %v"+
			"\nReceived: %v", expectedSig1, expectedSig2, signature)
	}
}
