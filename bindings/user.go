///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type User struct {
	u *user.User
}

func (u *User) GetID() []byte {
	return u.u.ID.Marshal()
}

func (u *User) GetSalt() []byte {
	return u.u.Salt
}

func (u *User) GetRSAPrivateKeyPem() []byte {
	return rsa.CreatePrivateKeyPem(u.u.RSA)
}

func (u *User) GetRSAPublicKeyPem() []byte {
	return rsa.CreatePublicKeyPem(u.u.RSA.GetPublic())
}

func (u *User) IsPrecanned() bool {
	return u.u.Precanned
}

func (u *User) GetCmixDhPrivateKey() []byte {
	return u.u.CmixDhPrivateKey.Bytes()
}

func (u *User) GetCmixDhPublicKey() []byte {
	return u.u.CmixDhPublicKey.Bytes()
}

func (u *User) GetE2EDhPrivateKey() []byte {
	return u.u.E2eDhPrivateKey.Bytes()
}

func (u *User) GetE2EDhPublicKey() []byte {
	return u.u.E2eDhPublicKey.Bytes()
}

func (u *User) GetContact() *Contact {
	c := u.u.GetContact()
	return &Contact{c: &c}
}
