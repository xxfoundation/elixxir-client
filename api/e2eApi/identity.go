////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package e2eApi

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

type TransmissionIdentity struct {
	ID            *id.ID
	RSAPrivatePem *rsa.PrivateKey
	Salt          []byte
	DHKeyPrivate  *cyclic.Int
}

// MakeTransmissionIdentity generates a new cryptographic identity for receiving messages
func MakeTransmissionIdentity(rng csprng.Source, grp *cyclic.Group) (TransmissionIdentity, error) {
	//make RSA Key
	rsaKey, err := rsa.GenerateKey(rng,
		rsa.DefaultRSABitLen)
	if err != nil {
		return TransmissionIdentity{}, err
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
		return TransmissionIdentity{}, err
	}

	//create the identity object
	I := TransmissionIdentity{
		ID:            newId,
		RSAPrivatePem: rsaKey,
		Salt:          salt,
		DHKeyPrivate:  privKey,
	}

	return I, nil
}

// DeepCopy produces a safe copy of a TransmissionIdentity
func (t TransmissionIdentity) DeepCopy() TransmissionIdentity {
	saltCopy := make([]byte, len(t.Salt))
	copy(saltCopy, t.Salt)
	return TransmissionIdentity{
		ID:            t.ID.DeepCopy(),
		RSAPrivatePem: t.RSAPrivatePem,
		Salt:          saltCopy,
		DHKeyPrivate:  t.DHKeyPrivate.DeepCopy(),
	}
}
