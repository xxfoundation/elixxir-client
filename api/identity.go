package api

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
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
func (c *Client) MakeIdentity() (*Identity, error) {
	stream := c.GetRng().GetStream()
	defer stream.Close()

	//make RSA Key
	rsaKey, err := rsa.GenerateKey(stream,
		rsa.DefaultRSABitLen)
	if err != nil {
		return nil, err
	}

	//make salt
	salt := make([]byte, 32)
	_, err = stream.Read(salt)

	//make dh private key
	privkey := diffieHellman.GeneratePrivateKey(
		len(c.GetStorage().GetE2EGroup().GetPBytes()),
		c.GetStorage().GetE2EGroup(), stream)

	//make the ID
	id, err := xx.NewID(rsaKey.GetPublic(),
		salt, id.User)
	if err != nil {
		return nil, err
	}

	//create the identity object
	I := &Identity{
		ID:            id,
		RSAPrivatePem: rsaKey,
		Salt:          salt,
		DHKeyPrivate:  privkey,
	}

	return I, nil
}
