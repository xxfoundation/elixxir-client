///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package backup

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/chacha20poly1305"

	"gitlab.com/xx_network/primitives/id"
)

type TransmissionIdentity struct {
	RSASigningPrivateKey []byte
	RegistrarSignature   []byte
	Salt                 []byte
	ComputedID           []byte
}

type ReceptionIdentity struct {
	RSASigningPrivateKey []byte
	RegistrarSignature   []byte
	Salt                 []byte
	ComputedID           []byte
	DHPrivateKey         []byte
	DHPublicKey          []byte
}

type UserDiscoveryRegistration struct {
	Username string
	Email    string
	Phone    string
}

type Contacts struct {
	UserIdentities []id.ID
}

type Backup struct {
	TransmissionIdentity      TransmissionIdentity
	ReceptionIdentity         ReceptionIdentity
	UserDiscoveryRegistration UserDiscoveryRegistration
	Contacts                  Contacts
}

func (b *Backup) Load(filepath string, key []byte) error {

	if len(key) != chacha20poly1305.KeySize {
		return errors.New("Backup.Store: incorrect key size")
	}

	blob, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	if len(blob) < chacha20poly1305.NonceSize+chacha20poly1305.Overhead+1 {
		return errors.New("ciphertext size is too small")
	}

	offset := chacha20poly1305.NonceSizeX
	nonce := blob[:offset]
	ciphertext := blob[offset:]

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return err
	}

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	err = json.Unmarshal(plaintext, b)

	return err
}

func (b *Backup) Store(filepath string, key []byte, nonce []byte) error {

	if len(key) != chacha20poly1305.KeySize {
		return errors.New("Backup.Store: incorrect key size")
	}

	if len(nonce) != chacha20poly1305.NonceSizeX {
		return errors.New("Backup.Store: incorrect nonce size")
	}

	blob, err := json.Marshal(b)
	if err != nil {
		return err
	}

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return err
	}

	ciphertext := cipher.Seal(nil, nonce, blob, nil)
	content := append(nonce, ciphertext...)

	tmpfile, err := ioutil.TempFile("", "state")
	if err != nil {
		return err
	}

	tmpPath := tmpfile.Name()

	_, err = tmpfile.Write(content)
	if err != nil {
		return err
	}

	err = tmpfile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, filepath)
	return err
}
