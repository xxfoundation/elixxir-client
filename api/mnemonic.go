///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	xxMnemonic "gitlab.com/xx_network/crypto/mnemonic"
	"golang.org/x/crypto/salsa20"
)

const (
	nonceSize = 8
)

// StoreSecretWithMnemonic creates a mnemonic and uses it to encrypt the secret.
// This encrypted data saved in storage.
func (c *Client) StoreSecretWithMnemonic(secret []byte) (string, error) {
	rng := c.rng.GetStream()

	// Create a mnemonic
	mnemonic, err := xxMnemonic.GenerateMnemonic(rng, 32)
	if err != nil {
		return "", errors.Errorf("Failed to generate mnemonic: %v", err)
	}

	// Encrypt secret with mnemonic as key
	ciphertext, nonce, err := encryptWithMnemonic(mnemonic, secret, rng)
	if err != nil {
		return "", errors.Errorf("Failed to encrypt secret with mnemonic: %v", err)
	}

	// Concatenate ciphertext with nonce for storage
	data := marshalMnemonicInformation(nonce, ciphertext)

	// Save data to storage
	err = c.storage.SaveMnemonicInformation(data)
	if err != nil {
		return "", errors.Errorf("Failed to store mnemonic information: %v", err)
	}

	return mnemonic, nil
}

// LoadSecretWithMnemonic loads the encrypted secret from storage and decrypts
// the secret using the given mnemonic.
func (c *Client) LoadSecretWithMnemonic(mnemonic string) (secret []byte, err error) {
	data, err := c.storage.LoadMnemonicInformation()
	if err != nil {
		return nil, errors.Errorf("Failed to load mnemonic information: %v", err)
	}

	nonce, ciphertext := unmarshalMnemonicInformation(data)

	secret = decryptWithMnemonic(nonce, ciphertext, mnemonic)

	return secret, nil
}

// encryptWithMnemonic is a helper function which encrypts the given secret
// using the mnemonic as the key.
func encryptWithMnemonic(mnemonic string, secret []byte,
	rng *fastRNG.Stream) (ciphertext, nonce []byte, err error) {

	// Place the key into a 32 byte array for salsa 20
	var keyArray [32]byte
	copy(keyArray[:], mnemonic)

	// Generate the nonce
	nonce, err = csprng.Generate(nonceSize, rng)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to generate nonce for encryption: %v", err)
	}

	// Encrypt the secret
	ciphertext = make([]byte, len(secret))
	salsa20.XORKeyStream(ciphertext, secret, nonce, &keyArray)

	return ciphertext, nonce, nil
}

// decryptWithMnemonic is a helper function which decrypts the secret
// from storage, using the mnemonic as the key.
func decryptWithMnemonic(nonce, ciphertext []byte, mnemonic string) (secret []byte) {
	// Place the key into a 32 byte array for salsa 20
	var keyArray [32]byte
	copy(keyArray[:], mnemonic)

	// Decrypt the secret
	secret = make([]byte, len(ciphertext))
	salsa20.XORKeyStream(secret, ciphertext, nonce, &keyArray)

	return secret
}

// marshalMnemonicInformation is a helper function which concatenates the nonce
// and ciphertext.
func marshalMnemonicInformation(nonce, ciphertext []byte) []byte {
	return append(nonce, ciphertext...)
}

// unmarshalMnemonicInformation is a helper function which separates the
// concatenated data containing the nonce and ciphertext of the mnemonic
// handling. This is the inverse of marshalMnemonicInformation.
func unmarshalMnemonicInformation(data []byte) (nonce, ciphertext []byte) {
	return data[:nonceSize], data[nonceSize:]
}
