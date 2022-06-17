////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

type ReceptionIdentity struct {
	ID            *id.ID
	RSAPrivatePem *rsa.PrivateKey
	Salt          []byte
	DHKeyPrivate  *cyclic.Int
}

// MakeReceptionIdentity generates a new cryptographic identity for receiving messages
func MakeReceptionIdentity(rng csprng.Source, grp *cyclic.Group) (ReceptionIdentity, error) {
	//make RSA Key
	rsaKey, err := rsa.GenerateKey(rng,
		rsa.DefaultRSABitLen)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	//make salt
	salt := make([]byte, 32)
	_, err = rng.Read(salt)

	//make dh private key
	privKey := diffieHellman.GeneratePrivateKey(
		len(grp.GetPBytes()),
		grp, rng)

	//make the ID
	newId, err := xx.NewID(rsaKey.GetPublic(),
		salt, id.User)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	//create the identity object
	I := ReceptionIdentity{
		ID:            newId,
		RSAPrivatePem: rsaKey,
		Salt:          salt,
		DHKeyPrivate:  privKey,
	}

	return I, nil
}

// DeepCopy produces a safe copy of a ReceptionIdentity
func (r ReceptionIdentity) DeepCopy() ReceptionIdentity {
	saltCopy := make([]byte, len(r.Salt))
	copy(saltCopy, r.Salt)
	return ReceptionIdentity{
		ID:            r.ID.DeepCopy(),
		RSAPrivatePem: r.RSAPrivatePem,
		Salt:          saltCopy,
		DHKeyPrivate:  r.DHKeyPrivate.DeepCopy(),
	}
}

// GetContact accepts a xxdk.ReceptionIdentity object and returns a contact.Contact object
func (r ReceptionIdentity) GetContact(grp *cyclic.Group) contact.Contact {
	dhPub := grp.ExpG(r.DHKeyPrivate, grp.NewInt(1))

	ct := contact.Contact{
		ID:             r.ID,
		DhPubKey:       dhPub,
		OwnershipProof: nil,
		Facts:          nil,
	}
	return ct
}
