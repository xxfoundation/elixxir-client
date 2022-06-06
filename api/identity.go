package api

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

type Identity struct {
	ID            *id.ID
	RSAPrivatePem *rsa.PrivateKey
	Salt          []byte
	DHKeyPrivate  *cyclic.Int
}

// MakeIdentity generates a new cryptographic identity for receiving messages
func MakeIdentity(rng csprng.Source, grp *cyclic.Group) (Identity, error) {
	//make RSA Key
	rsaKey, err := rsa.GenerateKey(rng,
		rsa.DefaultRSABitLen)
	if err != nil {
		return Identity{}, err
	}

	//make salt
	salt := make([]byte, 32)
	_, err = rng.Read(salt)

	//make dh private key
	privkey := diffieHellman.GeneratePrivateKey(
		len(grp.GetPBytes()),
		grp, rng)

	//make the ID
	id, err := xx.NewID(rsaKey.GetPublic(),
		salt, id.User)
	if err != nil {
		return Identity{}, err
	}

	//create the identity object
	I := Identity{
		ID:            id,
		RSAPrivatePem: rsaKey,
		Salt:          salt,
		DHKeyPrivate:  privkey,
	}

	return I, nil
}
