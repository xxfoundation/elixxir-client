////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
)

// TestStoreLoadDeleteSIDHPublicKey tests the load/store/delete functions
// for SIDH Public Keys
func TestStoreLoadDeleteSIDHPublicKey(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)
	rng := fastRNG.NewStreamGenerator(1, 3, csprng.NewSystemRNG)
	myRng := rng.GetStream()
	x1 := NewSIDHPublicKey(sidh.KeyVariantSidhA)
	p1 := NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	p1.Generate(myRng)
	p1.GeneratePublicKey(x1)

	k1 := "testKey1"
	err := StoreSIDHPublicKey(vkv, x1, k1)
	if err != nil {
		t.Errorf("Failed to store key: %+v", err)
	}
	loaded1, err := LoadSIDHPublicKey(vkv, k1)
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if StringSIDHPubKey(x1) != StringSIDHPubKey(loaded1) {
		t.Errorf("Stored key did not match loaded:\n\t%s\n\t%s\n",
			StringSIDHPubKey(x1), StringSIDHPubKey(loaded1))
	}
	err = DeleteSIDHPublicKey(vkv, k1)
	if err != nil {
		t.Fatalf("DeleteSIDHPublicKey returned an error: %v", err)
	}
	_, err = LoadSIDHPublicKey(vkv, k1)
	if err == nil {
		t.Errorf("Should not load deleted key: %+v", err)
	}

	// Now do the same for Tag B keys

	x2 := NewSIDHPublicKey(sidh.KeyVariantSidhB)
	p2 := NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	p2.Generate(myRng)
	p2.GeneratePublicKey(x2)

	k2 := "testKey2"
	err = StoreSIDHPublicKey(vkv, x2, k2)
	if err != nil {
		t.Errorf("Failed to store key: %+v", err)
	}
	loaded2, err := LoadSIDHPublicKey(vkv, k2)
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if StringSIDHPubKey(x2) != StringSIDHPubKey(loaded2) {
		t.Errorf("Stored key did not match loaded:\n\t%s\n\t%s\n",
			StringSIDHPubKey(x2), StringSIDHPubKey(loaded2))
	}
	err = DeleteSIDHPublicKey(vkv, k2)
	if err != nil {
		t.Fatalf("DeleteSIDHPublicKey returned an error: %v", err)
	}
	_, err = LoadSIDHPublicKey(vkv, k2)
	if err == nil {
		t.Errorf("Should not load deleted key: %+v", err)
	}

	myRng.Close()
}

// TestStoreLoadDeleteSIDHPublicKey tests the load/store/delete functions
// for SIDH Private Keys
func TestStoreLoadDeleteSIDHPrivateKey(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)
	rng := fastRNG.NewStreamGenerator(1, 3, csprng.NewSystemRNG)
	myRng := rng.GetStream()
	p1 := NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	p1.Generate(myRng)

	k1 := "testKey1"
	err := StoreSIDHPrivateKey(vkv, p1, k1)
	if err != nil {
		t.Errorf("Failed to store key: %+v", err)
	}
	loaded1, err := LoadSIDHPrivateKey(vkv, k1)
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if StringSIDHPrivKey(p1) != StringSIDHPrivKey(loaded1) {
		t.Errorf("Stored key did not match loaded:\n\t%s\n\t%s\n",
			StringSIDHPrivKey(p1), StringSIDHPrivKey(loaded1))
	}
	err = DeleteSIDHPrivateKey(vkv, k1)
	if err != nil {
		t.Fatalf("DeleteSIDHPrivateKey returned an error: %v", err)
	}
	_, err = LoadSIDHPrivateKey(vkv, k1)
	if err == nil {
		t.Errorf("Should not load deleted key: %+v", err)
	}

	// Now do the same for Tag B keys

	p2 := NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	p2.Generate(myRng)

	k2 := "testKey2"
	err = StoreSIDHPrivateKey(vkv, p2, k2)
	if err != nil {
		t.Errorf("Failed to store key: %+v", err)
	}
	loaded2, err := LoadSIDHPrivateKey(vkv, k2)
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if StringSIDHPrivKey(p2) != StringSIDHPrivKey(loaded2) {
		t.Errorf("Stored key did not match loaded:\n\t%s\n\t%s\n",
			StringSIDHPrivKey(p2), StringSIDHPrivKey(loaded2))
	}
	err = DeleteSIDHPrivateKey(vkv, k2)
	if err != nil {
		t.Fatalf("DeleteSIDHPrivateKey returned an error: %v", err)
	}
	_, err = LoadSIDHPrivateKey(vkv, k2)
	if err == nil {
		t.Errorf("Should not load deleted key: %+v", err)
	}

	myRng.Close()
}
