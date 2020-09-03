package user

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type User struct {
	ci *CryptographicIdentity

	regValidationSig []byte
	rvsMux           sync.RWMutex

	username    string
	usernameMux sync.RWMutex

	kv *versioned.KV
}

// builds a new user.
func NewUser(kv *versioned.KV, uid *id.ID, salt []byte, rsaKey *rsa.PrivateKey,
	isPrecanned bool) (*User, error) {

	ci := newCryptographicIdentity(uid, salt, rsaKey, isPrecanned, kv)

	return &User{ci: ci, kv: kv}, nil
}

func LoadUser(kv *versioned.KV) (*User, error) {
	ci, err := loadCryptographicIdentity(kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load user "+
			"due to failure to load cryptographic identity")
	}

	u := &User{ci: ci, kv: kv}
	u.loadRegistrationValidationSignature()
	u.loadUsername()

	return u, nil
}

func (u *User) GetCryptographicIdentity() *CryptographicIdentity {
	return u.ci
}
