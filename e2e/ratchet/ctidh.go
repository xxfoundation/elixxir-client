//go:build ctidh
// +build ctidh

package ratchet

import (
	ctidh "git.xx.network/elixxir/ctidh_cgo"

	"gitlab.com/elixxir/client/interfaces"
)

// CtidhNike implements the Nike interface using our ctidh module.
type CtidhNike struct {
}

// NewCtidhNike returns a new Ctidh Nike.
func NewCtidhNike() *CtidhNike {
	return new(CtidhNike)
}

var _ interfaces.PrivateKey = (*ctidh.PrivateKey)(nil)
var _ interfaces.PublicKey = (*ctidh.PublicKey)(nil)
var _ interfaces.Nike = (*CtidhNike)(nil)

// PublicKeySize returns the size in bytes of the public key.
func (e *CtidhNike) PublicKeySize() int {
	return ctidh.PublicKeySize
}

// PrivateKeySize returns the size in bytes of the private key.
func (e *CtidhNike) PrivateKeySize() int {
	return ctidh.PrivateKeySize
}

// NewEmptyPublicKey returns an uninitialized
// PublicKey which is suitable to be loaded
// via some serialization format via FromBytes
// or FromPEMFile methods.
func (e *CtidhNike) NewEmptyPublicKey() interfaces.PublicKey {
	return ctidh.NewEmptyPublicKey()
}

// NewEmptyPrivateKey returns an uninitialized
// PrivateKey which is suitable to be loaded
// via some serialization format via FromBytes
// or FromPEMFile methods.
func (e *CtidhNike) NewEmptyPrivateKey() interfaces.PrivateKey {
	return ctidh.NewEmptyPrivateKey()
}

// NewKeypair returns a newly generated key pair.
func (e *CtidhNike) NewKeypair() (interfaces.PrivateKey, interfaces.PublicKey) {
	privKey, pubKey := ctidh.GenerateKeyPair()
	return privKey, pubKey
}

// DeriveSecret derives a shared secret given a private key
// from one party and a public key from another.
func (e *CtidhNike) DeriveSecret(privKey interfaces.PrivateKey, pubKey interfaces.PublicKey) []byte {
	return ctidh.DeriveSecret(privKey.(*ctidh.PrivateKey), pubKey.(*ctidh.PublicKey))
}

// DerivePublicKey derives a public key given a private key.
func (e *CtidhNike) DerivePublicKey(privKey interfaces.PrivateKey) interfaces.PublicKey {
	return ctidh.DerivePublicKey(privKey.(*ctidh.PrivateKey))
}
