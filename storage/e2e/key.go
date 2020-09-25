package e2e

import (
	"github.com/pkg/errors"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

type Key struct {
	// Links
	session *Session

	fp *format.Fingerprint

	// keyNum is the index of the key by order of creation
	// it is used to identify the key in the key.Session
	keyNum uint32
}

func newKey(session *Session, keynum uint32) *Key {
	return &Key{
		session: session,
		keyNum:  keynum,
	}
}

// return pointers to higher level management structures
func (k *Key) GetSession() *Session { return k.session }

// returns the key fingerprint if it has it, otherwise generates it
// this function does not memoize the fingerprint if it doesnt have it because
// in most cases it will not be used for a long time and as a result should not
// be stored in ram.
func (k *Key) Fingerprint() format.Fingerprint {
	if k.fp != nil {
		return *k.fp
	}
	return e2eCrypto.DeriveKeyFingerprint(k.session.baseKey, k.keyNum)
}

// the E2E key to encrypt msg to its intended recipient
// It also properly populates the associated data, including the MAC, fingerprint,
// and encrypted timestamp
func (k *Key) Encrypt(msg format.Message) format.Message {
	fp := k.Fingerprint()
	key := k.generateKey()

	// set the fingerprint
	msg.SetKeyFP(fp)

	// encrypt the payload
	encPayload := e2eCrypto.Crypt(key, fp, msg.GetContents())
	msg.SetContents(encPayload)

	// create the MAC
	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key[:])
	msg.SetMac(MAC)

	return msg
}

// Decrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// or in case of a decryption error (related to padding)
func (k *Key) Decrypt(msg format.Message) (format.Message, error) {
	fp := k.Fingerprint()
	key := k.generateKey()

	// Verify the MAC is correct
	if !hash.VerifyHMAC(msg.GetContents(), msg.GetMac(), key[:]) {
		return format.Message{}, errors.New("HMAC verification failed for E2E message")
	}

	// Decrypt the payload
	decryptedPayload := e2eCrypto.Crypt(key, fp, msg.GetContents())

	//put the decrypted payload back in the message
	msg.SetContents(decryptedPayload)

	return msg, nil
}

// Sets the key as used
func (k *Key) denoteUse() {
	k.session.useKey(k.keyNum)
}

// Generates the key and returns it
func (k *Key) generateKey() e2eCrypto.Key {
	return e2eCrypto.DeriveKey(k.session.baseKey, k.keyNum)
}
