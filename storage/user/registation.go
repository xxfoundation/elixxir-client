///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

const currentRegValidationSigVersion = 0
const registrationTimestampVersion = 0
const transmissionRegValidationSigKey = "transmissionRegistrationValidationSignature"
const receptionRegValidationSigKey = "receptionRegistrationValidationSignature"
const registrationTimestampKey = "registrationTimestamp"

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

// Returns the registration timestamp stored in RAM as
func (u *User) GetRegistrationTimestamp() time.Time {
	u.rvsMux.RLock()
	defer u.rvsMux.RUnlock()
	return u.registrationTimestamp
}

// Loads the transmission Identity Validation Signature if it exists in the ekv
func (u *User) loadTransmissionRegistrationValidationSignature() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(transmissionRegValidationSigKey,
		currentRegValidationSigVersion)
	if err == nil {
		u.transmissionRegValidationSig = obj.Data
	}
	u.rvsMux.Unlock()
}

// Loads the reception Identity Validation Signature if it exists in the ekv
func (u *User) loadReceptionRegistrationValidationSignature() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(receptionRegValidationSigKey,
		currentRegValidationSigVersion)
	if err == nil {
		u.receptionRegValidationSig = obj.Data
	}
	u.rvsMux.Unlock()
}

// Loads the registration timestamp if it exists in the ekv
func (u *User) loadRegistrationTimestamp() {
	u.rvsMux.Lock()
	obj, err := u.kv.Get(registrationTimestampKey,
		registrationTimestampVersion)
	if err == nil {
		tsNano := binary.BigEndian.Uint64(obj.Data)
		u.registrationTimestamp = time.Unix(0, int64(tsNano))
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
		Timestamp: netTime.Now(),
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
		Timestamp: netTime.Now(),
		Data:      b,
	}

	err := u.kv.Set(receptionRegValidationSigKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store the reception Identity Validation "+
			"Signature: %s", err)
	}

	u.receptionRegValidationSig = b
}

// Sets the Registration Timestamp if it is not set and stores it in
// the ekv
func (u *User) SetRegistrationTimestamp(tsNano int64) {
	u.rvsMux.Lock()
	defer u.rvsMux.Unlock()

	//check if the signature already exists
	if !u.registrationTimestamp.IsZero() {
		jww.FATAL.Panicf("cannot overwrite existing registration timestamp")
	}

	// Serialize the timestamp
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(tsNano))

	obj := &versioned.Object{
		Version:   registrationTimestampVersion,
		Timestamp: netTime.Now(),
		Data:      tsBytes,
	}

	err := u.kv.Set(registrationTimestampKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Failed to store the reception timestamp: %s", err)
	}

	u.registrationTimestamp = time.Unix(0, tsNano)

}
