////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package sync covers logic regarding account synchronization.
package sync

import (
	"crypto/cipher"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"io"
)

// Used for keyed hashes for, e.g., the "key" in the KV store
func deriveKeyName(key, secret string) []byte {
	dHash := blake2b.Sum256([]byte(key))
	pHash := blake2b.Sum256([]byte(secret))
	s := append(pHash[:], dHash[:]...)
	h := blake2b.Sum256(s)
	return h[:]
}

func initChaCha20Poly1305(secret string) cipher.AEAD {
	pwHash := blake2b.Sum256([]byte(secret))
	chaCipher, err := chacha20poly1305.NewX(pwHash[:])
	if err != nil {
		panic(fmt.Sprintf("Could not init XChaCha20Poly1305 mode: %s",
			err.Error()))
	}
	return chaCipher
}

func encrypt(data []byte, secret string, csprng io.Reader) []byte {
	chaCipher := initChaCha20Poly1305(secret)
	nonce := make([]byte, chaCipher.NonceSize())
	if _, err := io.ReadFull(csprng, nonce); err != nil {
		panic(fmt.Sprintf("Could not generate nonce: %s", err.Error()))
	}
	ciphertext := chaCipher.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, password string) ([]byte, error) {
	chaCipher := initChaCha20Poly1305(password)
	nonceLen := chaCipher.NonceSize()
	if (len(data) - nonceLen) <= 0 {
		errMsg := fmt.Sprintf("Read %d bytes, too short to decrypt",
			len(data))
		return nil, errors.New(errMsg)
	}
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := chaCipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot decrypt with password!")
	}
	return plaintext, nil
}
