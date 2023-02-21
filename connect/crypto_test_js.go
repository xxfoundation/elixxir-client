////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// TestSignVerify_Consistency DOES NOT PERFORM A PROPER CONSISTENCY TEST WHEN
// COMPILED FOR WASM.  This test currently only confirms that the formed
// signature can be verified when running in js/wasm.
func TestSignVerify_Consistency(t *testing.T) {
	// use insecure seeded rng to reproduce key
	notRand := rand.New(rand.NewSource(11))

	sch := rsa.GetScheme()

	privKey, err := sch.Generate(notRand, 1024)
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

	partnerId, err := xx.NewID(privKey.Public(), salt, id.User)
	if err != nil {
		t.Fatalf("NewId error: %v", err)
	}

	err = verify(partnerId, privKey.Public(), signature, connFp, salt)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}

	// NOTE: since the signature formed during wasm tests uses the browser RNG,
	// we cannot perform a proper consistency test here.  The best we can do is
	// verify the signature.
}
