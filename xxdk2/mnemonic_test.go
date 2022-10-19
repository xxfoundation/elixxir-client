////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk2

import (
	"bytes"
	"gitlab.com/xx_network/crypto/csprng"
	xxMnemonic "gitlab.com/xx_network/crypto/mnemonic"
	"gitlab.com/xx_network/primitives/utils"
	"io"
	"math/rand"
	"testing"
)

func TestStoreSecretWithMnemonic(t *testing.T) {
	secret := []byte("test123")
	storageDir := "ignore.1/"
	mnemonic, err := StoreSecretWithMnemonic(secret, storageDir)
	if err != nil {
		t.Errorf("StoreSecretWithMnemonic error; %v", err)
	}

	// Tests the mnemonic returned is valid
	_, err = xxMnemonic.DecodeMnemonic(mnemonic)
	if err != nil {
		t.Errorf("StoreSecretWithMnemonic did not return a decodable mnemonic: %v", err)
	}

	// Test that the file was written to
	if !utils.Exists(storageDir + mnemonicFile) {
		t.Errorf("Mnemonic file does not exist in storage: %v", err)
	}

}

func TestLoadSecretWithMnemonic(t *testing.T) {
	secret := []byte("test123")
	storageDir := "ignore.1"
	mnemonic, err := StoreSecretWithMnemonic(secret, storageDir)
	if err != nil {
		t.Errorf("StoreSecretWithMnemonic error; %v", err)
	}

	received, err := LoadSecretWithMnemonic(mnemonic, storageDir)
	if err != nil {
		t.Errorf("LoadSecretWithMnemonic error: %v", err)
	}

	if !bytes.Equal(received, secret) {
		t.Fatalf("Loaded secret does not match original data."+
			"\n\tExpected: %v\n\tReceived: %v", secret, received)
	}

	_, err = LoadSecretWithMnemonic(mnemonic, "badDirectory")
	if err == nil {
		t.Fatalf("LoadSecretWithMnemonic should error when provided a path " +
			"where a recovery file does not exist.")
	}
}

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }
