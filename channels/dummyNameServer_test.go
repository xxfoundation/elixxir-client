////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
)

const numTests = 10

// Smoke test.
func TestNewDummyNameService(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	_, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

}

// Smoke test.
func TestDummyNameService_GetUsername(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	ns, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

	if username != ns.GetUsername() {
		t.Fatalf("GetUsername did not return expected value."+
			"\nExpected: %s"+
			"\nReceived: %s", username, ns.GetUsername())
	}

}

// Smoke test.
func TestDummyNameService_SignChannelMessage(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	ns, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

	message := []byte("the secret is in the sauce.")

	signature, err := ns.SignChannelMessage(message)
	if err != nil {
		t.Fatalf("SignChannelMessage error: %v", err)
	}

	if len(signature) != ed25519.SignatureSize {
		t.Errorf("DummyNameService's SignChannelMessage did not return a "+
			"signature of expected size, according to ed25519 specifications."+
			"\nExpected: %d"+
			"\nReceived: %d", ed25519.SignatureSize, len(signature))
	}

}

// Smoke test.
func TestDummyNameService_GetChannelValidationSignature(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	ns, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

	validationSig, _ := ns.GetChannelValidationSignature()

	if len(validationSig) != ed25519.SignatureSize {
		t.Errorf("DummyNameService's GetChannelValidationSignature did not "+
			"return a validation signature of expected size, according to "+
			"ed25519 specifications."+
			"\nExpected: %d"+
			"\nReceived: %d", ed25519.SignatureSize, len(validationSig))
	}

}

// Smoke test.
func TestDummyNameService_ValidateChannelMessage(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	ns, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

	for i := 0; i < numTests; i++ {
		if !ns.ValidateChannelMessage(username, netTime.Now(), nil, nil) {
			t.Errorf("ValidateChannelMessage returned false. This should " +
				"only ever return true.")
		}
	}
}

// Smoke test.
func TestDummyNameService_GetChannelPubkey(t *testing.T) {
	rng := csprng.NewSystemRNG()
	username := "floridaMan"
	ns, err := NewDummyNameService(username, rng)
	if err != nil {
		t.Fatalf("NewDummyNameService error: %+v", err)
	}

	if len(ns.GetChannelPubkey()) != ed25519.PublicKeySize {
		t.Errorf("DummyNameService's GetChannelPubkey did not "+
			"return a validation signature of expected size, according to "+
			"ed25519 specifications."+
			"\nExpected: %d"+
			"\nReceived: %d", ed25519.PublicKeySize, ns.GetChannelPubkey())
	}
}
