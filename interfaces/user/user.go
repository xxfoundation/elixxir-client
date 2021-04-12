///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type User struct {
	//General Identity
	TransmissionID   *id.ID
	TransmissionSalt []byte
	TransmissionRSA  *rsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     *rsa.PrivateKey
	Precanned        bool

	//cmix Identity
	CmixDhPrivateKey *cyclic.Int
	CmixDhPublicKey  *cyclic.Int

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}

func (u User) GetContact() contact.Contact {
	return contact.Contact{
		ID:       u.ReceptionID.DeepCopy(),
		DhPubKey: u.E2eDhPublicKey,
		Facts:    make([]fact.Fact, 0),
	}
}
