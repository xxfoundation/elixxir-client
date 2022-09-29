package ratchet

import (
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/primitives/format"
)

type EncryptedMessage struct {
	Ciphertext  []byte
	Residue     []byte
	Fingerprint format.Fingerprint
}

type SymmetricKeyRatchetFactory interface {
	FromBytes(serializedRatchet []byte) (SymmetricKeyRatchet, error)
	New(sharedSecret, salt []byte, fingerprintMapSize uint32) SymmetricKeyRatchet
}

type SymmetricKeyRatchet interface {
	Encrypt(plaintext []byte) (*EncryptedMessage, error)
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
	Salt() []byte
	Size() uint32
}

type RatchetFactory interface {
	NewRatchets(myPrivateKey nike.PrivateKey, partnerPublicKey nike.PublicKey) (SendRatchet, ReceiveRatchet)
	SendRatchetFromBytes([]byte) (SendRatchet, error)
	ReceiveRatchetFromBytes([]byte) (ReceiveRatchet, error)
}

type SendRatchet interface {
	Encrypt(plaintext []byte) (*EncryptedMessage, error)
	Save() ([]byte, error)
	Next() SendRatchet
	MyPublicKey() nike.PublicKey
}

type ReceiveRatchet interface {
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
	Next(theirPublicKey nike.PublicKey) ReceiveRatchet
}
