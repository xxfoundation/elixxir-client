////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type User struct {
	*CryptographicIdentity

	transmissionRegValidationSig []byte
	receptionRegValidationSig    []byte
	// Time in which user registered with the network
	registrationTimestamp time.Time
	rvsMux                sync.RWMutex

	username    string
	usernameMux sync.RWMutex

	kv versioned.KV
}

// builds a new user.
func NewUser(syncKV versioned.KV, transmissionID, receptionID *id.ID, transmissionSalt,
	receptionSalt []byte, transmissionRsa, receptionRsa rsa.PrivateKey, isPrecanned bool,
	e2eDhPrivateKey, e2eDhPublicKey *cyclic.Int) (*User, error) {

	ci := newCryptographicIdentity(transmissionID, receptionID,
		transmissionSalt, receptionSalt, transmissionRsa,
		receptionRsa, isPrecanned,
		e2eDhPrivateKey, e2eDhPublicKey, syncKV)

	return &User{CryptographicIdentity: ci, kv: syncKV}, nil
}

func LoadUser(syncKV versioned.KV) (*User, error) {
	ci, err := loadCryptographicIdentity(syncKV)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load user "+
			"due to failure to load cryptographic identity")
	}

	u := &User{CryptographicIdentity: ci, kv: syncKV}
	u.loadTransmissionRegistrationValidationSignature()
	u.loadReceptionRegistrationValidationSignature()
	u.loadUsername()
	u.loadRegistrationTimestamp()

	return u, nil
}
