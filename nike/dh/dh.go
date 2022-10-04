package dh

import (
	"errors"

	"gitlab.com/elixxir/client/interfaces/nike"

	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"

	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
)

const (
	bitSize        = 2048
	groupSize      = bitSize / 8
	privateKeySize = groupSize + 8
	publicKeySize  = groupSize + 8
)

var primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
	"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
	"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
	"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
	"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
	"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
	"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
	"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
	"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
	"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
	"15728E5A8AACAA68FFFFFFFFFFFFFFFF"

type dhNIKE struct{}

// DHNIKE is essentially a factory type that builds
// various types related to the DiffieHellman NIKE interface implementation.
var DHNIKE = &dhNIKE{}

var _ nike.PrivateKey = (*privateKey)(nil)
var _ nike.PublicKey = (*publicKey)(nil)
var _ nike.Nike = (*dhNIKE)(nil)

func (d *dhNIKE) PublicKeySize() int {
	return publicKeySize
}

func (d *dhNIKE) PrivateKeySize() int {
	return privateKeySize
}

func (d *dhNIKE) NewEmptyPrivateKey() nike.PrivateKey {
	return &privateKey{
		privateKey: new(cyclic.Int),
	}
}

func (d *dhNIKE) NewEmptyPublicKey() nike.PublicKey {
	return &publicKey{
		publicKey: new(cyclic.Int),
	}
}

// UnmarshalBinaryPublicKey unmarshals the public key bytes.
func (d *dhNIKE) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubKey := d.NewEmptyPublicKey()
	err := pubKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// UnmarshalBinaryPrivateKey unmarshals the public key bytes.
func (d *dhNIKE) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privKey := d.NewEmptyPrivateKey()
	err := privKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func (d *dhNIKE) group() *cyclic.Group {
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	return cyclic.NewGroup(p, g)
}

func (d *dhNIKE) NewKeypair() (nike.PrivateKey, nike.PublicKey) {
	rng := csprng.NewSystemRNG()
	group := d.group()
	privKey := diffieHellman.GeneratePrivateKey(privateKeySize, group, rng)
	pubKey := diffieHellman.GeneratePublicKey(privKey, group)
	return &privateKey{
			privateKey: privKey,
		}, &publicKey{
			publicKey: pubKey,
		}
}

type privateKey struct {
	privateKey *cyclic.Int
}

func (p *privateKey) DeriveSecret(pubKey nike.PublicKey) []byte {
	c := diffieHellman.GenerateSessionKey(p.privateKey,
		(pubKey.(*publicKey)).publicKey,
		DHNIKE.group())
	return c.Bytes()
}

func (p *privateKey) Reset() {
	p.privateKey = nil
}

func (p *privateKey) Bytes() []byte {
	return p.privateKey.BinaryEncode()
}

func (p *privateKey) FromBytes(data []byte) error {
	if len(data) != DHNIKE.PrivateKeySize() {
		return errors.New("invalid key size")
	}
	return p.privateKey.BinaryDecode(data)
}

func (p *privateKey) Scheme() nike.Nike {
	return DHNIKE
}

type publicKey struct {
	publicKey *cyclic.Int
}

func (p *publicKey) Reset() {
	p.publicKey = nil
}

func (p *publicKey) Bytes() []byte {
	return p.publicKey.BinaryEncode()
}

func (p *publicKey) FromBytes(data []byte) error {
	if len(data) != DHNIKE.PublicKeySize() {
		return errors.New("invalid key size")
	}
	err := p.publicKey.BinaryDecode(data)
	if err != nil {
		return nil
	}
	if !diffieHellman.CheckPublicKey(DHNIKE.group(), p.publicKey) {
		return errors.New("not a valid public key")
	}
	return nil
}

func (p *publicKey) Scheme() nike.Nike {
	return DHNIKE
}
