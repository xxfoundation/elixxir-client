//go:build ctidh
// +build ctidh

package ctidh

import (
	"encoding/pem"

	ctidh "git.xx.network/elixxir/ctidh_cgo"

	"gitlab.com/elixxir/client/interfaces/nike"
)

// ctidhNIKE implements the Nike interface using our ctidh module.
type ctidhNIKE struct{}

// NewCTIDHNIKE returns a new CTIDH NIKE.
func NewCTIDHNIKE() *ctidhNIKE {
	return new(ctidhNIKE)
}

var _ nike.PrivateKey = (*PrivateKey)(nil)
var _ nike.PublicKey = (*PublicKey)(nil)
var _ nike.Nike = (*ctidhNIKE)(nil)

// PublicKeySize returns the size in bytes of the public key.
func (e *ctidhNIKE) PublicKeySize() int {
	return ctidh.PublicKeySize
}

// PrivateKeySize returns the size in bytes of the private key.
func (e *ctidhNIKE) PrivateKeySize() int {
	return ctidh.PrivateKeySize
}

// NewKeypair returns a newly generated key pair.
func (e *ctidhNIKE) NewKeypair() (nike.PrivateKey, nike.PublicKey) {
	privKey, pubKey := ctidh.GenerateKeyPair()
	return &PrivateKey{
			nike:       e,
			privateKey: privKey,
		}, &PublicKey{
			nike:      e,
			publicKey: pubKey,
		}
}

func (e *ctidhNIKE) NewEmptyPublicKey() nike.PublicKey {
	return &PublicKey{
		nike:      e,
		publicKey: ctidh.NewEmptyPublicKey(),
	}
}

func (e *ctidhNIKE) NewEmptyPrivateKey() nike.PrivateKey {
	return &PrivateKey{
		nike:       e,
		privateKey: ctidh.NewEmptyPrivateKey(),
	}
}

// UnmarshalBinaryPublicKey unmarshals the public key bytes.
func (e *ctidhNIKE) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubKey := e.NewEmptyPublicKey()
	err := pubKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// UnmarshalBinaryPrivateKey unmarshals the public key bytes.
func (e *ctidhNIKE) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privKey := e.NewEmptyPrivateKey()
	err := privKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

type PrivateKey struct {
	privateKey *ctidh.PrivateKey
	nike       nike.Nike
}

func (p *PrivateKey) Scheme() nike.Nike {
	return p.nike
}

func (p *PrivateKey) Reset() {
	// no op
}

func (p *PrivateKey) ToPEMFile(f string) error {
	return p.privateKey.ToPEMFile(f)
}

func (p *PrivateKey) ToPEM() (*pem.Block, error) {
	return p.privateKey.ToPEM()
}

func (p *PrivateKey) Bytes() []byte {
	return p.privateKey.Bytes()
}

func (p *PrivateKey) FromBytes(data []byte) error {
	return p.privateKey.FromBytes(data)
}

func (p *PrivateKey) DeriveSecret(publicKey nike.PublicKey) []byte {
	return p.privateKey.DeriveSecret(publicKey.(*PublicKey).publicKey)
}

type PublicKey struct {
	nike      nike.Nike
	publicKey *ctidh.PublicKey
}

func (p *PublicKey) Scheme() nike.Nike {
	return p.nike
}

func (p *PublicKey) ToPEMFile(f string) error {
	return p.publicKey.ToPEMFile(f)
}

func (p *PublicKey) ToPEM() (*pem.Block, error) {
	return p.publicKey.ToPEM()
}

func (p *PublicKey) Reset() {
	// no op
}

func (p *PublicKey) Bytes() []byte {
	return p.publicKey.Bytes()
}

func (p *PublicKey) FromBytes(data []byte) error {
	return p.publicKey.FromBytes(data)
}
