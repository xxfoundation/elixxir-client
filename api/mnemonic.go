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
	"gitlab.com/xx_network/primitives/utils"
	"golang.org/x/crypto/chacha20poly1305"
	"path/filepath"
	"strings"
)

const mnemonicFile = ".recovery"

// StoreSecretWithMnemonic creates a mnemonic and uses it to encrypt the secret.
// This encrypted data saved in storage.
func StoreSecretWithMnemonic(secret []byte, path string) (string, error) {
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG).GetStream()

	// Ensure path is appended by filepath separator "/"
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path = path + string(filepath.Separator)
	}

	// Create a mnemonic
	mnemonic, err := xxMnemonic.GenerateMnemonic(rng, 32)
	if err != nil {
		return "", errors.Errorf("Failed to generate mnemonic: %v", err)
	}

	// Decode mnemonic
	decodedMnemonic, err := xxMnemonic.DecodeMnemonic(mnemonic)
	if err != nil {
		return "", errors.Errorf("Failed to decode mnemonic: %v", err)
	}

	// Encrypt secret with mnemonic as key
	ciphertext, err := encryptWithMnemonic(secret, decodedMnemonic, rng)
	if err != nil {
		return "", errors.Errorf("Failed to encrypt secret with mnemonic: %v", err)
	}

	// Save encrypted secret to file
	recoveryFile := path + mnemonicFile
	err = utils.WriteFileDef(recoveryFile, ciphertext)
	if err != nil {
		return "", errors.Errorf("Failed to save mnemonic information to file")
	}

	return mnemonic, nil
}

// LoadSecretWithMnemonic loads the encrypted secret from storage and decrypts
// the secret using the given mnemonic.
func LoadSecretWithMnemonic(mnemonic, path string) (secret []byte, err error) {
	// Ensure path is appended by filepath separator "/"
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path = path + string(filepath.Separator)
	}

	// Ensure that the recovery file exists
	recoveryFile := path + mnemonicFile
	if !utils.Exists(recoveryFile) {
		return nil, errors.Errorf("Recovery file does not exist. " +
			"Did you properly set up recovery or provide an incorrect filepath?")
	}

	// Read file from storage
	data, err := utils.ReadFile(recoveryFile)
	if err != nil {
		return nil, errors.Errorf("Failed to load mnemonic information: %v", err)
	}

	// Decode mnemonic
	decodedMnemonic, err := xxMnemonic.DecodeMnemonic(mnemonic)
	if err != nil {
		return nil, errors.Errorf("Failed to decode mnemonic: %v", err)
	}

	// Decrypt the stored secret
	secret, err = decryptWithMnemonic(data, decodedMnemonic)
	if err != nil {
		return nil, errors.Errorf("Failed to decrypt secret: %v", err)
	}

	return secret, nil
}

// encryptWithMnemonic is a helper function which encrypts the given secret
// using the mnemonic as the key.
func encryptWithMnemonic(data, decodedMnemonic []byte,
	rng csprng.Source) (ciphertext []byte, error error) {
	chaCipher, err := chacha20poly1305.NewX(decodedMnemonic[:])
	if err != nil {
		return nil, errors.Errorf("Failed to initalize encryption algorithm: %v", err)
	}

	// Generate the nonce
	nonce := make([]byte, chaCipher.NonceSize())
	nonce, err = csprng.Generate(chaCipher.NonceSize(), rng)
	if err != nil {
		return nil, errors.Errorf("Failed to generate nonce: %v", err)
	}

	ciphertext = chaCipher.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptWithMnemonic is a helper function which decrypts the secret
// from storage, using the mnemonic as the key.
func decryptWithMnemonic(data, decodedMnemonic []byte) ([]byte, error) {
	chaCipher, err := chacha20poly1305.NewX(decodedMnemonic[:])
	if err != nil {
		return nil, errors.Errorf("Failed to initalize encryption algorithm: %v", err)
	}

	nonceLen := chaCipher.NonceSize()
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := chaCipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot decrypt with password!")
	}
	return plaintext, nil
}
