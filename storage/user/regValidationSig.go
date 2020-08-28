package user

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const currentRegValidationSigVersion = 0
const regValidationSigKey = "registrationValidationSignature"

// Returns the Registration Validation Signature stored in RAM. May return
// nil of no signature is stored
func (u *User) GetRegistrationValidationSignature() []byte {
	u.rvsMux.RLock()
	defer u.rvsMux.RUnlock()
	return u.regValidationSig
}

// Loads the Registration Validation Signature if it exists in the ekv
func (u *User) loadRegistrationValidationSignature() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(regValidationSigKey)
	if err == nil {
		u.regValidationSig = obj.Data
	}
	u.rvsMux.Unlock()
}

// Sets the Registration Validation Signature if it is not set and stores it in
// the ekv
func (u *User) SetRegistrationValidationSignature(b []byte) error {
	u.rvsMux.Lock()
	defer u.rvsMux.Unlock()

	//check if the signature already exists
	if u.regValidationSig != nil {
		return errors.New("cannot overwrite existing Registration Validation Signature")
	}

	obj := &versioned.Object{
		Version:   currentRegValidationSigVersion,
		Timestamp: time.Now(),
		Data:      b,
	}

	err := u.kv.Set(regValidationSigKey, obj)
	if err != nil {
		return errors.WithMessage(err, "Failed to store the "+
			"Registration Validation Signature")
	}

	u.regValidationSig = b

	return nil
}
