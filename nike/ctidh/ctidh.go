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
			privateKey: privKey,
		}, &PublicKey{
			publicKey: pubKey,
		}
}

func (e *ctidhNIKE) PublicKeyFromPEMFile(f string) (nike.PublicKey, error) {
	pubKey := ctidh.NewEmptyPublicKey()
	err := pubKey.FromPEMFile(f)
	if err != nil {
		return nil, err
	}
	return &PublicKey{
		publicKey: pubKey,
	}, nil
}

func (e *ctidhNIKE) PublicKeyFromPEM(pemBytes []byte) (nike.PublicKey, error) {
	pubKey := ctidh.NewEmptyPublicKey()
	err := pubKey.FromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	return &PublicKey{
		publicKey: pubKey,
	}, nil
}

func (e *ctidhNIKE) PrivateKeyFromPEMFile(f string) (nike.PrivateKey, error) {
	privKey := ctidh.NewEmptyPrivateKey()
	err := privKey.FromPEMFile(f)
	if err != nil {
		return nil, err
	}
	return &PrivateKey{
		privateKey: privKey,
	}, nil
}

func (e *ctidhNIKE) PrivateKeyFromPEM(pemBytes []byte) (nike.PrivateKey, error) {
	privKey := ctidh.NewEmptyPrivateKey()
	err := privKey.FromPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	return &PrivateKey{
		privateKey: privKey,
	}, nil
}

func (e *ctidhNIKE) PublicKeyToPEMFile(f string, pubKey nike.PublicKey) error {
	return pubKey.(*PublicKey).ToPEMFile(f)
}

func (e *ctidhNIKE) PublicKeyToPEM(pubKey nike.PublicKey) (*pem.Block, error) {
	return pubKey.(*PublicKey).ToPEM()
}

func (e *ctidhNIKE) PrivateKeyToPEMFile(f string, privKey nike.PrivateKey) error {
	return privKey.(*PrivateKey).ToPEMFile(f)
}

func (e *ctidhNIKE) PrivateKeyToPEM(privKey nike.PrivateKey) (*pem.Block, error) {
	return privKey.(*PrivateKey).ToPEM()
}

// UnmarshalBinaryPublicKey unmarshals the public key bytes.
func (e *ctidhNIKE) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubKey := ctidh.NewEmptyPublicKey()
	err := pubKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// UnmarshalBinaryPrivateKey unmarshals the public key bytes.
func (e *ctidhNIKE) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privKey := ctidh.NewEmptyPrivateKey()
	err := privKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return &PrivateKey{
		privateKey: privKey,
	}, nil
}

// DeriveSecret derives a shared secret given a private key
// from one party and a public key from another.
func (e *ctidhNIKE) DeriveSecret(privKey nike.PrivateKey, pubKey nike.PublicKey) []byte {
	return ctidh.DeriveSecret(privKey.(*PrivateKey).privateKey, pubKey.(*PublicKey).publicKey)
}

// DerivePublicKey derives a public key given a private key.
func (e *ctidhNIKE) DerivePublicKey(privKey nike.PrivateKey) nike.PublicKey {
	return ctidh.DerivePublicKey(privKey.(*PrivateKey).privateKey)
}

// PublicKeyEqual is a constant time key comparison.
func (e *ctidhNIKE) PublicKeyEqual(a, b nike.PublicKey) bool {
	return a.(*PublicKey).publicKey.Equal(b.(*PublicKey).publicKey)
}

// PrivateKeyEqual is a constant time key comparison.
func (e *ctidhNIKE) PrivateKeyEqual(a, b nike.PrivateKey) bool {
	return a.(*PrivateKey).privateKey.Equal(b.(*PrivateKey).privateKey)
}

type PrivateKey struct {
	privateKey *ctidh.PrivateKey
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
	publicKey *ctidh.PublicKey
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
