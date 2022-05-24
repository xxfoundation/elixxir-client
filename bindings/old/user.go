///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type User struct {
	u *user.Info
}

func (u *User) GetTransmissionID() []byte {
	return u.u.TransmissionID.Marshal()
}

func (u *User) GetReceptionID() []byte {
	return u.u.ReceptionID.Marshal()
}

func (u *User) GetTransmissionSalt() []byte {
	return u.u.TransmissionSalt
}

func (u *User) GetReceptionSalt() []byte {
	return u.u.ReceptionSalt
}

func (u *User) GetTransmissionRSAPrivateKeyPem() []byte {
	return rsa.CreatePrivateKeyPem(u.u.TransmissionRSA)
}

func (u *User) GetTransmissionRSAPublicKeyPem() []byte {
	return rsa.CreatePublicKeyPem(u.u.TransmissionRSA.GetPublic())
}

func (u *User) GetReceptionRSAPrivateKeyPem() []byte {
	return rsa.CreatePrivateKeyPem(u.u.ReceptionRSA)
}

func (u *User) GetReceptionRSAPublicKeyPem() []byte {
	return rsa.CreatePublicKeyPem(u.u.ReceptionRSA.GetPublic())
}

func (u *User) IsPrecanned() bool {
	return u.u.Precanned
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
