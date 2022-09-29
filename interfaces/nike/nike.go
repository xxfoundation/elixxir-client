package nike

import "encoding/pem"

// Key is an interface for types encapsulating key material.
type Key interface {

	// Reset resets the key material to all zeros.
	Reset()

	// Bytes serializes key material into a byte slice.
	Bytes() []byte

	// FromBytes loads key material from the given byte slice.
	FromBytes(data []byte) error
}

// PrivateKey is an interface for types encapsulating
// private key material.
type PrivateKey interface {
	Key

	DeriveSecret(PublicKey) []byte
}

// PublicKey is an interface for types encapsulating
// public key material.
type PublicKey interface {
	Key
}

// Nike is an interface encapsulating a
// non-interactive key exchange.
type Nike interface {

	// PublicKeySize returns the size in bytes of the public key.
	PublicKeySize() int

	// PrivateKeySize returns the size in bytes of the private key.
	PrivateKeySize() int

	// NewKeypair returns a newly generated key pair.
	NewKeypair() (PrivateKey, PublicKey)

	// UnmarshalBinaryPublicKey unmarshals the public key bytes.
	UnmarshalBinaryPublicKey(b []byte) (PublicKey, error)

	// UnmarshalBinaryPrivateKey unmarshals the public key bytes.
	UnmarshalBinaryPrivateKey(b []byte) (PrivateKey, error)

	// DeriveSecret derives a shared secret given a private key
	// from one party and a public key from another.
	DeriveSecret(PrivateKey, PublicKey) []byte

	// DerivePublicKey derives a public key given a private key.
	DerivePublicKey(PrivateKey) PublicKey

	// PublicKeyEqual is a constant time key comparison.
	PublicKeyEqual(PublicKey, PublicKey) bool

	// PrivateKeyEqual is a constant time key comparison.
	PrivateKeyEqual(PrivateKey, PrivateKey) bool

	// PublicKeyFromPEMFile unmarshals a public key from the PEM file.
	PublicKeyFromPEMFile(string) (PublicKey, error)

	// PublicKeyFromPEM unmarshals a public key from the PEM bytes.
	PublicKeyFromPEM([]byte) (PublicKey, error)

	// PublicKeyToPEMFile write the key to the PEM file.
	PublicKeyToPEMFile(string, PublicKey) error

	// PublicKeyToPEM writes the key to a PEM block.
	PublicKeyToPEM(PublicKey) (*pem.Block, error)

	// PrivateKeyFromPEMFile unmarshals a private key from the PEM file.
	PrivateKeyFromPEMFile(string) (PrivateKey, error)

	// PrivateKeyFromPEM unmarshals a private key from the PEM bytes.
	PrivateKeyFromPEM([]byte) (PrivateKey, error)

	// PrivateKeyToPEMFile writes the key to the PEM file.
	PrivateKeyToPEMFile(string, PrivateKey) error

	// PrivateKeyToPEM writes the key to a PEM block.
	PrivateKeyToPEM(PrivateKey) (*pem.Block, error)
}
