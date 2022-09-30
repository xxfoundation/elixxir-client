package ratchet

import (
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/primitives/format"
)

type EncryptedMessage struct {
	Ciphertext  []byte
	Residue     []byte
	Fingerprint format.Fingerprint
}

type XXRatchet interface {
	Encrypt(sendRatchetID session.SessionID,
		plaintext []byte) (*EncryptedMessage, error)
	Decrypt(receiveRatchetID session.SessionID,
		message *EncryptedMessage) (plaintext []byte, err error)

	// Rekey creates a new receiving ratchet defined
	// by the received rekey trigger public key.  This is called
	// by case 6 above.  This calls the cyHdlr.AddKey() for each
	// key fingerprint, and in theory can directly give it the
	// Receive Ratchet, eliminating the need to even bother with a
	// Decrypt function at this layer.
	Rekey(oldReceiverRatchetID session.SessionID,
		theirPublicKey nike.PublicKey) (session.SessionID, nike.PublicKey)

	// State Management Functions
	SetState(senderID session.SessionID, newState session.Negotiation) error
	SendRatchets() []session.SessionID
	SendRatchetsByState(state session.Negotiation) []session.SessionID
	ReceiveRatchets() []session.SessionID
}

type RekeyTrigger interface {
	TriggerRekey(ratchetID session.SessionID, myPublicKey nike.PublicKey)
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
