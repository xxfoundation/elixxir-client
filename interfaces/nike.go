package interfaces

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
}

// PublicKey is an interface for types encapsulating
// public key material.
type PublicKey interface {
	Key

	// Blind performs a blinding operation and mutates the public
	// key with the blinded value.
	Blind(blindingFactor []byte) error
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

	// NewEmptyPublicKey returns an uninitialized
	// PublicKey which is suitable to be loaded
	// via some serialization format via FromBytes
	// or FromPEMFile methods.
	NewEmptyPublicKey() PublicKey

	// NewEmptyPrivateKey returns an uninitialized
	// PrivateKey which is suitable to be loaded
	// via some serialization format via FromBytes
	// or FromPEMFile methods.
	NewEmptyPrivateKey() PrivateKey

	// DeriveSecret derives a shared secret given a private key
	// from one party and a public key from another.
	DeriveSecret(PrivateKey, PublicKey) []byte

	// DerivePublicKey derives a public key given a private key.
	DerivePublicKey(PrivateKey) PublicKey
}
