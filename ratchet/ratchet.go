package ratchet

import (
	"gitlab.com/elixxir/primitives/format"

	"gitlab.com/elixxir/client/interfaces/nike"
)

type Scheme interface {
	FromBytes(serializedRatchet []byte) (Ratchet, error)
	FromRatchet(Ratchet, theirPublicKey nike.PublicKey) Ratchet
	New(myPrivateKey nike.PrivateKey, theirPublicKey nike.PublicKey) Ratchet
}

type Ratchet interface {
	Encrypt(plaintext []byte) *EncryptedMessage
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
}

type EncryptedMessage struct {
	Ciphertext  []byte
	Residue     []byte
	Fingerprint format.Fingerprint
}
