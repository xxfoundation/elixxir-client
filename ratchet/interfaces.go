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

type XXRatchet interface {
	Encrypt(sendRatchetID ID,
		plaintext []byte) (*EncryptedMessage, error)
	Decrypt(receiveRatchetID ID,
		message *EncryptedMessage) (plaintext []byte, err error)

	Rekey(oldReceiverRatchetID ID,
		theirPublicKey nike.PublicKey) (ID, error)

	// State Management Functions
	SetState(senderID ID, newState NegotiationState) error
	SendRatchets() []ID
	SendRatchetsByState(state NegotiationState) []ID
	ReceiveRatchets() []ID
}

type RekeyTrigger interface {
	TriggerRekey(ratchetID ID, myPublicKey nike.PublicKey)
}

type FingerprintTracker interface {
	AddKey(fp format.Fingerprint)
	DeleteKey(fp format.Fingerprint)
}

type RatchetFactory interface {
	NewRatchets(myPrivateKey nike.PrivateKey, myPublicKey nike.PublicKey, partnerPublicKey nike.PublicKey) (SendRatchet, ReceiveRatchet)
	SendRatchetFromBytes([]byte) (SendRatchet, error)
	ReceiveRatchetFromBytes([]byte) (ReceiveRatchet, error)
}

type SendRatchet interface {
	ID() ID
	Encrypt(plaintext []byte) (*EncryptedMessage, error)
	Save() ([]byte, error)
	Next() SendRatchet
	MyPublicKey() nike.PublicKey
}

type ReceiveRatchet interface {
	ID() ID
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
	Next(theirPublicKey nike.PublicKey) ReceiveRatchet
	DeriveFingerprints() []format.Fingerprint
}

type SymmetricKeyRatchetFactory interface {
	FromBytes(serializedRatchet []byte) (SymmetricKeyRatchet, error)
	New(sharedSecret, salt []byte, fingerprintMapSize, threshold uint32) SymmetricKeyRatchet
}

type SymmetricKeyRatchet interface {
	ID() ID
	Encrypt(plaintext []byte) (*EncryptedMessage, error)
	Decrypt(*EncryptedMessage) (plaintext []byte, err error)
	Save() ([]byte, error)
	Salt() []byte
	Size() uint32
	Threshold() uint32
	DeriveFingerprints() []format.Fingerprint
}
