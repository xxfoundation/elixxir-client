package user

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
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

type ciDisk struct {
	UserID      *id.ID
	Salt        []byte
	RsaKey      *rsa.PrivateKey
	IsPrecanned bool
}

func newCryptographicIdentity(uid *id.ID, salt []byte, rsaKey *rsa.PrivateKey,
	isPrecanned bool, kv *versioned.KV) *CryptographicIdentity {

	ci := &CryptographicIdentity{
		userID:      uid,
		salt:        salt,
		rsaKey:      rsaKey,
		isPrecanned: isPrecanned,
	}

	if err := ci.save(kv); err != nil {
		jww.FATAL.Panicf("Failed to store the new Cryptographic"+
			" Identity: %s", err)
	}

	return ci
}

func loadCryptographicIdentity(kv *versioned.KV) (*CryptographicIdentity, error) {
	obj, err := kv.Get(cryptographicIdentityKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get user "+
			"cryptographic identity from EKV")
	}

	var resultBuffer bytes.Buffer
	result := &CryptographicIdentity{}
	decodable := &ciDisk{}

	resultBuffer.Write(obj.Data)
	dec := gob.NewDecoder(&resultBuffer)
	err = dec.Decode(decodable)

	if decodable != nil {
		result.isPrecanned = decodable.IsPrecanned
		result.rsaKey = decodable.RsaKey
		result.salt = decodable.Salt
		result.userID = decodable.UserID
	}

	return result, err
}

func (ci *CryptographicIdentity) save(kv *versioned.KV) error {
	var userDataBuffer bytes.Buffer

	encodable := &ciDisk{
		UserID:      ci.userID,
		Salt:        ci.salt,
		RsaKey:      ci.rsaKey,
		IsPrecanned: ci.isPrecanned,
	}

	enc := gob.NewEncoder(&userDataBuffer)
	err := enc.Encode(encodable)
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
