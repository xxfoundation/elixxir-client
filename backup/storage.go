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
	Usernames []string
}

type Backup struct {
	TransmissionIdentity      TransmissionIdentity
	ReceptionIdentity         ReceptionIdentity
	UserDiscoveryRegistration UserDiscoveryRegistration
	Contacts                  Contacts
}

func (b *Backup) Load(filepath string) error {
	return nil // fixme
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
