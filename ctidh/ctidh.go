//go:build ctidh
// +build ctidh

package ctidh

import (
	ctidh "git.xx.network/elixxir/ctidh_cgo"

	"gitlab.com/elixxir/client/interfaces/nike"
)

// CtidhNike implements the Nike interface using our ctidh module.
type CtidhNike struct {
}

// NewCtidhNike returns a new Ctidh Nike.
func NewCtidhNike() *CtidhNike {
	return new(CtidhNike)
}

var _ nike.PrivateKey = (*ctidh.PrivateKey)(nil)
var _ nike.PublicKey = (*ctidh.PublicKey)(nil)
var _ nike.Nike = (*CtidhNike)(nil)

// PublicKeySize returns the size in bytes of the public key.
func (e *CtidhNike) PublicKeySize() int {
	return ctidh.PublicKeySize
}

// PrivateKeySize returns the size in bytes of the private key.
func (e *CtidhNike) PrivateKeySize() int {
	return ctidh.PrivateKeySize
}

// NewKeypair returns a newly generated key pair.
func (e *CtidhNike) NewKeypair() (nike.PrivateKey, nike.PublicKey) {
	privKey, pubKey := ctidh.GenerateKeyPair()
	return privKey, pubKey
}

// UnmarshalBinaryPublicKey unmarshals the public key bytes.
func (e *CtidhNike) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubKey := ctidh.NewEmptyPublicKey()
	err := pubKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// UnmarshalBinaryPrivateKey unmarshals the public key bytes.
func (e *CtidhNike) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privKey := ctidh.NewEmptyPrivateKey()
	err := privKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// DeriveSecret derives a shared secret given a private key
// from one party and a public key from another.
func (e *CtidhNike) DeriveSecret(privKey nike.PrivateKey, pubKey nike.PublicKey) []byte {
	return ctidh.DeriveSecret(privKey.(*ctidh.PrivateKey), pubKey.(*ctidh.PublicKey))
}

// DerivePublicKey derives a public key given a private key.
func (e *CtidhNike) DerivePublicKey(privKey nike.PrivateKey) nike.PublicKey {
	return ctidh.DerivePublicKey(privKey.(*ctidh.PrivateKey))
}
