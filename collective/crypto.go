////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"io"
)

type encryptor interface {
	Encrypt(data []byte) []byte
	Decrypt(data []byte) ([]byte, error)
	KeyID(deviceID cmix.InstanceID) string
}

type deviceCrypto struct {
	secret string
	rngGen *fastRNG.StreamGenerator
}

func (dc *deviceCrypto) Encrypt(data []byte) []byte {
	stream := dc.rngGen.GetStream()
	defer stream.Close()
	return encrypt(data, dc.secret, stream)
}

func (dc *deviceCrypto) Decrypt(data []byte) ([]byte, error) {
	return decrypt(data, dc.secret)
}

func (dc *deviceCrypto) KeyID(deviceID cmix.InstanceID) string {
	return keyID(dc.secret, deviceID)
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

func decrypt(data []byte, secret string) ([]byte, error) {
	chaCipher := initChaCha20Poly1305(secret)
	nonceLen := chaCipher.NonceSize()
	if (len(data) - nonceLen) <= 0 {
		errMsg := fmt.Sprintf("Read %d bytes, too short to decrypt",
			len(data))
		return nil, errors.New(errMsg)
	}
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := chaCipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot decrypt with secret!")
	}
	return plaintext, nil
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

func keyID(secret string, deviceID cmix.InstanceID) string {
	// this will panic on error, intentional
	h, _ := hash.NewCMixHash()
	h.Write([]byte(secret))
	h.Write(deviceID[:])
	keyIDBytes := h.Sum(nil)

	return base64.RawStdEncoding.EncodeToString(keyIDBytes)
}
