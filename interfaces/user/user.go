package user

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type User struct {
	//General Identity
	ID        *id.ID
	Salt      []byte
	RSA       *rsa.PrivateKey
	Precanned bool

	//cmix Identity
	CmixDhPrivateKey *cyclic.Int
	CmixDhPublicKey  *cyclic.Int

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}

func (u User) GetID() []byte {
	return u.ID.Marshal()
}

func (u User) GetSalt() []byte {
	return u.Salt
}

func (u User) GetRSAPrivateKeyPem() []byte {
	return rsa.CreatePrivateKeyPem(u.RSA)
}

func (u User) GetRSAPublicKeyPem() []byte {
	return rsa.CreatePublicKeyPem(u.RSA.GetPublic())
}

func (u User) IsPrecanned() bool {
	return u.Precanned
}

func (u User) GetCmixDhPrivateKey() []byte {
	return u.CmixDhPrivateKey.Bytes()
}

func (u User) GetCmixDhPublicKey() []byte {
	return u.CmixDhPublicKey.Bytes()
}

func (u User) GetE2EDhPrivateKey() []byte {
	return u.E2eDhPrivateKey.Bytes()
}

func (u User) GetE2EDhPublicKey() []byte {
	return u.E2eDhPublicKey.Bytes()
}

func (u User) GetContact() interfaces.Contact {
	return contact.Contact{
		ID:       u.ID.DeepCopy(),
		DhPubKey: u.E2eDhPublicKey,
		Facts:    nil,
	}
}
