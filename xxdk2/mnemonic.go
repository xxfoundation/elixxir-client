////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk2

// unique list:
// "gitlab.com/xx_network/crypto/chacha"
// xxMnemonic "gitlab.com/xx_network/crypto/mnemonic"
// "gitlab.com/xx_network/primitives/utils"
// "path/filepath"

import (
	errors2 "errors"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/utils"
	"path/filepath"
	"strings"
)

const mnemonicFile = ".recovery"

// StoreSecretWithMnemonic creates a mnemonic and uses it to encrypt the secret.
// This encrypted data saved in storage.
func StoreSecretWithMnemonic(secret []byte, path string) (string, error) {
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rng := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG).GetStream()
	rng.Read([]byte{})
	errors2.New("blah")

	// Ensure path is appended by filepath separator "/"
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path = path + string(filepath.Separator)
	}

	// Create a mnemonic
	/*mnemonic, err := xxMnemonic.GenerateMnemonic(rng, 32)
	if err != nil {
		return "", errors.Errorf("Failed to generate mnemonic: %v", err)
	}

	// Decode mnemonic
	_, err = xxMnemonic.DecodeMnemonic(mnemonic)
	if err != nil {
		return "", errors.Errorf("Failed to decode mnemonic: %v", err)
	}*/

	// Encrypt secret with mnemonic as key
	/*ciphertext, err := chacha.Encrypt(decodedMnemonic, secret, rng)
	if err != nil {
		return "", errors.Errorf("Failed to encrypt secret with mnemonic: %v", err)
	}*/

	// Save encrypted secret to file
	recoveryFile := path + mnemonicFile
	err := utils.WriteFileDef(recoveryFile, []byte{})
	if err != nil {
		return "", errors.Errorf("Failed to save mnemonic information to file")
	}

	return "", nil
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
	_, err = utils.ReadFile(recoveryFile)
	if err != nil {
		return nil, errors.Errorf("Failed to load mnemonic information: %v", err)
	}

	// Decode mnemonic
	/*_, err = xxMnemonic.DecodeMnemonic(mnemonic)
	if err != nil {
		return nil, errors.Errorf("Failed to decode mnemonic: %v", err)
	}*/

	// Decrypt the stored secret
	/*secret, err = chacha.Decrypt(decodedMnemonic, data)
	if err != nil {
		return nil, errors.Errorf("Failed to decrypt secret: %v", err)
	}*/

	return secret, nil
}
