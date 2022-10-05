package hybrid

import (
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/nike/ctidh"
	"gitlab.com/elixxir/client/nike/dh"
)

var CTIDHDiffieHellman nike.Nike = &scheme{
	name:   "CTIDHDiffieHellman",
	first:  ctidh.CTIDHNIKE,
	second: dh.DHNIKE,
}

type scheme struct {
	name   string
	first  nike.Nike
	second nike.Nike
}

func (s *scheme) NewKeypair() (nike.PrivateKey, nike.PublicKey) {
	privKey1, pubKey1 := s.first.NewKeypair()
	privKey2, pubKey2 := s.second.NewKeypair()
	return &privateKey{
			first:  privKey1,
			second: privKey2,
		}, &publicKey{
			first:  pubKey1,
			second: pubKey2,
		}
}

func (s *scheme) DerivePublicKey(privKey nike.PrivateKey) nike.PublicKey {
	pubKey1 := privKey.(*privateKey).first.Scheme().DerivePublicKey(privKey.(*privateKey).first)
	pubKey2 := privKey.(*privateKey).second.Scheme().DerivePublicKey(privKey.(*privateKey).second)
	return &publicKey{
		first:  pubKey1,
		second: pubKey2,
	}
}

func (s *scheme) Name() string { return s.name }

func (s *scheme) PublicKeySize() int {
	return s.first.PublicKeySize() + s.second.PublicKeySize()
}

func (s *scheme) PrivateKeySize() int {
	return s.first.PrivateKeySize() + s.second.PrivateKeySize()
}

func (s *scheme) NewEmptyPrivateKey() nike.PrivateKey {
	return &privateKey{
		first:  s.first.NewEmptyPrivateKey(),
		second: s.second.NewEmptyPrivateKey(),
	}
}

func (s *scheme) NewEmptyPublicKey() nike.PublicKey {
	return &publicKey{
		first:  s.first.NewEmptyPublicKey(),
		second: s.second.NewEmptyPublicKey(),
	}
}

func (s *scheme) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubKey := s.NewEmptyPublicKey()
	err := pubKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func (s *scheme) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privKey := s.NewEmptyPrivateKey()
	err := privKey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

type privateKey struct {
	first  nike.PrivateKey
	second nike.PrivateKey
}

func (p *privateKey) Scheme() nike.Nike {
	return CTIDHDiffieHellman
}

func (p *privateKey) DeriveSecret(pubKey nike.PublicKey) []byte {
	secret1 := p.first.DeriveSecret(pubKey.(*publicKey).first)
	secret2 := p.second.DeriveSecret(pubKey.(*publicKey).second)
	return append(secret1, secret2...)
}

func (p *privateKey) Bytes() []byte {
	return append(p.first.Bytes(), p.second.Bytes()...)
}

func (p *privateKey) Reset() {
	p.first = nil
	p.second = nil
}

func (p *privateKey) FromBytes(data []byte) error {
	err := p.first.FromBytes(data[:p.first.Scheme().PrivateKeySize()])
	if err != nil {
		return err
	}
	return p.second.FromBytes(data[p.first.Scheme().PrivateKeySize():])
}

type publicKey struct {
	first  nike.PublicKey
	second nike.PublicKey
}

func (p *publicKey) Scheme() nike.Nike {
	return CTIDHDiffieHellman
}

func (p *publicKey) Reset() {
	p.first = nil
	p.second = nil
}

func (p *publicKey) Bytes() []byte {
	return append(p.first.Bytes(), p.second.Bytes()...)
}

func (p *publicKey) FromBytes(data []byte) error {
	err := p.first.FromBytes(data[:p.first.Scheme().PublicKeySize()])
	if err != nil {
		return err
	}
	return p.second.FromBytes(data[p.first.Scheme().PublicKeySize():])
}
