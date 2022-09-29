package ratchet

import (
	"gitlab.com/elixxir/primitives/format"
)

type Scheme interface {
	FromBytes(serializedRatchet []byte) (Ratchet, error)
	New(sharedSecret, salt []byte, fingerprintMapSize uint) Ratchet
}

type Ratchet interface {
	Encrypt(plaintext []byte) (*EncryptedMessage, error)
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
}

type EncryptedMessage struct {
	Ciphertext  []byte
	Residue     []byte
	Fingerprint format.Fingerprint
}
