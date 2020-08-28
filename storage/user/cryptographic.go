package user

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentCryptographicIdentityVersion = 0
const cryptographicIdentityKey = "cryptographicIdentity"

type CryptographicIdentity struct {
	userID      *id.ID
	salt        []byte
	rsaKey      *rsa.PrivateKey
	isPrecanned bool
}

func newCryptographicIdentity(uid *id.ID, salt []byte, rsaKey *rsa.PrivateKey,
	isPrecanned bool, kv *versioned.KV) (*CryptographicIdentity, error) {

	_, err := kv.Get(cryptographicIdentityKey)
	if err == nil {
		return nil, errors.New("cannot create cryptographic identity " +
			"when one already exists")
	}

	ci := &CryptographicIdentity{
		userID:      uid,
		salt:        salt,
		rsaKey:      rsaKey,
		isPrecanned: isPrecanned,
	}

	return ci, ci.save(kv)
}

func loadCryptographicIdentity(kv *versioned.KV) (*CryptographicIdentity, error) {
	obj, err := kv.Get(cryptographicIdentityKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get user "+
			"cryptographic identity from EKV")
	}

	var resultBuffer bytes.Buffer
	var result *CryptographicIdentity
	resultBuffer.Write(obj.Data)
	dec := gob.NewDecoder(&resultBuffer)
	err = dec.Decode(result)

	return result, err
}

func (ci *CryptographicIdentity) save(kv *versioned.KV) error {
	var userDataBuffer bytes.Buffer
	enc := gob.NewEncoder(&userDataBuffer)
	err := enc.Encode(ci)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   currentCryptographicIdentityVersion,
		Timestamp: time.Now(),
		Data:      userDataBuffer.Bytes(),
	}

	return kv.Set(cryptographicIdentityKey, obj)
}

func (ci *CryptographicIdentity) GetUserID() *id.ID {
	return ci.userID.DeepCopy()
}

func (ci *CryptographicIdentity) GetSalt() []byte {
	return ci.salt
}

func (ci *CryptographicIdentity) GetRSA() *rsa.PrivateKey {
	return ci.rsaKey
}

func (ci *CryptographicIdentity) IsPrecanned() bool {
	return ci.isPrecanned
}
