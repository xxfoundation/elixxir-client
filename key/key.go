package key

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

type Key struct {
	// Links
	session *Session

	fp *format.Fingerprint

	// Designation of crypto type
	outer parse.CryptoType

	// keyNum is the index of the key by order of creation
	// it is used to identify the key in the key.Session
	keyNum uint32
}

func newKey(session *Session, outer parse.CryptoType, keynum uint32) *Key {
	return &Key{
		session: session,
		outer:   outer,
		keyNum:  keynum,
	}
}

// return pointers to higher level management structures
func (k *Key) GetSession() *Session { return k.session }

// return the type of encryption this key is for
func (k *Key) GetCryptoType() parse.CryptoType { return k.outer }

// returns the key fingerprint if it has it, otherwise generates it
// this function does not memoize the fingerprint if it doesnt have it because
// in most cases it will not be used for a long time and as a result should not
// be stored in ram.
func (k *Key) Fingerprint() format.Fingerprint {
	if k.fp != nil {
		return *k.fp
	}

	var fp format.Fingerprint
	switch k.outer {
	case parse.E2E:
		fp = e2e.DeriveKeyFingerprint(k.session.baseKey, k.keyNum)
	case parse.Rekey:
		fp = e2e.DeriveReKeyFingerprint(k.session.baseKey, k.keyNum)
	default:
		globals.Log.FATAL.Panicf("Key has invalid cryptotype: %s",
			k.outer)
	}

	return fp
}

// the E2E key to encrypt msg to its intended recipient
// It also properly populates the associated data, including the MAC, fingerprint,
// and encrypted timestamp
func (k *Key) Encrypt(msg format.Message) format.Message {
	fp := k.Fingerprint()
	key := k.generateKey()

	// set the fingerprint
	msg.SetKeyFP(fp)

	// encrypt the timestamp
	msg.SetTimestamp(encryptTimestamp(fp, key, msg.GetTimestamp()[:15]))

	// encrypt the payload
	encPayload, err := e2e.Encrypt(key, fp, msg.Contents.GetRightAligned(), format.ContentsLen)
	if err != nil {
		globals.Log.ERROR.Panicf(err.Error())
	}
	msg.Contents.Set(encPayload)

	// create the MAC
	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key[:])
	msg.SetMAC(MAC)

	return msg
}

// the E2E key to encrypt msg to its intended recipient
// It also properly populates the associated data, including the MAC, fingerprint,
// and encrypted timestamp
// does not handle padding, so the underlying data must be unique or there are
// cryptographic vulnerabilities
func (k *Key) EncryptUnsafe(msg format.Message) format.Message {
	fp := k.Fingerprint()
	key := k.generateKey()

	// set the fingerprint
	msg.SetKeyFP(fp)

	// encrypt the timestamp
	msg.SetTimestamp(encryptTimestamp(fp, key, msg.GetTimestamp()))

	// encrypt the payload
	encPayload := e2e.CryptUnsafe(key, fp, msg.Contents.Get())
	msg.Contents.Set(encPayload)

	// create the MAC
	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key[:])
	msg.SetMAC(MAC)

	return msg
}

// Decrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// or in case of a decryption error (related to padding)
func (k *Key) Decrypt(msg format.Message) (format.Message, error) {
	fp := k.Fingerprint()
	key := k.generateKey()

	// Verify the MAC is correct
	if !hash.VerifyHMAC(msg.Contents.Get(), msg.GetMAC(), key[:]) {
		return format.Message{}, errors.New("HMAC verification failed for E2E message")
	}

	//decrypt the timestamp
	decryptedTimestamp, err := decryptTimestamp(fp, key, msg.GetTimestamp())
	if err != nil {
		return format.Message{}, errors.Errorf("Failed to decrypt E2E "+
			"message: %s", err.Error())
	}
	msg.SetTimestamp(decryptedTimestamp)

	// Decrypt the payload
	decryptedPayload, err := e2e.Decrypt(key, fp, msg.Contents.Get())

	if err != nil {
		return format.Message{}, errors.Errorf("Failed to decrypt E2E "+
			"message: %s", err.Error())
	}

	//put the decrypted payload back in the message
	msg.Contents.SetRightAligned(decryptedPayload)

	return msg, nil
}

// Decrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// assumes the payload has no padding
func (k *Key) DecryptUnsafe(msg format.Message) (format.Message, error) {
	fp := k.Fingerprint()
	key := k.generateKey()

	// Verify the MAC is correct
	if !hash.VerifyHMAC(msg.Contents.Get(), msg.GetMAC(), key[:]) {
		return format.Message{}, errors.New("HMAC verification failed for E2E message")
	}

	//decrypt the timestamp
	decryptedTimestamp, err := decryptTimestamp(fp, key, msg.GetTimestamp())
	if err != nil {
		return format.Message{}, errors.Errorf("Failed to decrypt E2E "+
			"message: %s", err.Error())
	}
	msg.SetTimestamp(decryptedTimestamp)

	// Decrypt the payload
	decryptedPayload := e2e.CryptUnsafe(key, fp, msg.Contents.Get())

	//put the decrypted payload back in the message
	msg.Contents.Set(decryptedPayload)
	return msg, nil
}

// Sets the key as used
func (k *Key) denoteUse() error {
	switch k.outer {
	case parse.E2E:
		err := k.session.useKey(k.keyNum)
		if err != nil {
			return errors.WithMessage(err, "Could not use e2e key")
		}

	case parse.Rekey:
		err := k.session.useReKey(k.keyNum)
		if err != nil {
			return errors.WithMessage(err, "Could not use e2e rekey")
		}
	default:
		globals.Log.FATAL.Panicf("Key has invalid cryptotype: %s",
			k.outer)
	}
	return nil
}

// Generates the key and returns it
func (k *Key) generateKey() e2e.Key {

	var key e2e.Key
	switch k.outer {
	case parse.E2E:
		key = e2e.DeriveKey(k.session.baseKey, k.keyNum)
	case parse.Rekey:
		key = e2e.DeriveReKey(k.session.baseKey, k.keyNum)
	default:
		globals.Log.FATAL.Panicf("Key has invalid cryptotype: %s",
			k.outer)
	}

	return key
}

//encrypts the timestamp
func encryptTimestamp(fp format.Fingerprint, key e2e.Key, ts []byte) []byte {
	// Encrypt the timestamp using key
	// Timestamp bytes were previously stored
	// and GO only uses 15 bytes, so use those
	var iv [e2e.AESBlockSize]byte
	copy(iv[:], fp[:e2e.AESBlockSize])
	encryptedTimestamp, err := e2e.EncryptAES256WithIV(key[:], iv,
		ts[:15])
	if err != nil {
		panic(err)
	}
	return encryptedTimestamp
}

//decrypts the timestamp
func decryptTimestamp(fp format.Fingerprint, key e2e.Key, ts []byte) ([]byte, error) {
	//create the IV array
	var iv [e2e.AESBlockSize]byte
	copy(iv[:], fp[:e2e.AESBlockSize])

	// decrypt the timestamp in the associated data
	decryptedTimestamp, err := e2e.DecryptAES256WithIV(key[:], iv, ts)
	if err != nil {
		return nil, errors.Errorf("Timestamp decryption failed for "+
			"E2E message: %s", err.Error())
	}

	//pad the timestamp
	decryptedTimestamp = append(decryptedTimestamp, 0)
	return decryptedTimestamp, nil
}
