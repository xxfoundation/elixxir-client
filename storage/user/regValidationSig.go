///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const currentRegValidationSigVersion = 0
const transmissionRegValidationSigKey = "transmissionRegistrationValidationSignature"
const receptionRegValidationSigKey = "receptionRegistrationValidationSignature"

// Returns the transmission Identity Validation Signature stored in RAM. May return
// nil of no signature is stored
func (u *User) GetTransmissionRegistrationValidationSignature() []byte {
	u.rvsMux.RLock()
	defer u.rvsMux.RUnlock()
	return u.transmissionRegValidationSig
}

// Returns the reception Identity Validation Signature stored in RAM. May return
// nil of no signature is stored
func (u *User) GetReceptionRegistrationValidationSignature() []byte {
	u.rvsMux.RLock()
	defer u.rvsMux.RUnlock()
	return u.receptionRegValidationSig
}

// Loads the transmission Identity Validation Signature if it exists in the ekv
func (u *User) loadTransmissionRegistrationValidationSignature() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(transmissionRegValidationSigKey)
	if err == nil {
		u.transmissionRegValidationSig = obj.Data
	}
	u.rvsMux.Unlock()
}

// Loads the reception Identity Validation Signature if it exists in the ekv
func (u *User) loadReceptionRegistrationValidationSignature() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(receptionRegValidationSigKey)
	if err == nil {
		u.receptionRegValidationSig = obj.Data
	}
	u.rvsMux.Unlock()
}

// Sets the Identity Validation Signature if it is not set and stores it in
// the ekv
func (u *User) SetTransmissionRegistrationValidationSignature(b []byte) {
	u.rvsMux.Lock()
	defer u.rvsMux.Unlock()

	//check if the signature already exists
	if u.transmissionRegValidationSig != nil {
		jww.FATAL.Panicf("cannot overwrite existing transmission Identity Validation Signature")
	}

	obj := &versioned.Object{
		Version:   currentRegValidationSigVersion,
		Timestamp: time.Now(),
		Data:      b,
	}

	err := u.kv.Set(transmissionRegValidationSigKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store the transmission Identity Validation "+
			"Signature: %s", err)
	}

	u.transmissionRegValidationSig = b
}

// Sets the Identity Validation Signature if it is not set and stores it in
// the ekv
func (u *User) SetReceptionRegistrationValidationSignature(b []byte) {
	u.rvsMux.Lock()
	defer u.rvsMux.Unlock()

	//check if the signature already exists
	if u.receptionRegValidationSig != nil {
		jww.FATAL.Panicf("cannot overwrite existing reception Identity Validation Signature")
	}

	obj := &versioned.Object{
		Version:   currentRegValidationSigVersion,
		Timestamp: time.Now(),
		Data:      b,
	}

	err := u.kv.Set(receptionRegValidationSigKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store the reception Identity Validation "+
			"Signature: %s", err)
	}

	u.receptionRegValidationSig = b
}
