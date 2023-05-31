////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"encoding/binary"
	"fmt"

	jww "github.com/spf13/jwalterweatherman"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentRegistrationStatusVersion = 0
const registrationStatusKey = "regStatusKey"

type RegistrationStatus uint32

const (
	NotStarted            RegistrationStatus = 0     // Set on session creation
	KeyGenComplete        RegistrationStatus = 10000 // Set upon generation of session information
	PermissioningComplete RegistrationStatus = 20000 // Set upon completion of RegisterWithPermissioning
	UDBComplete           RegistrationStatus = 30000 // Set upon completion of RegisterWithUdb
)

// stringer for Identity Status
func (rs RegistrationStatus) String() string {
	switch rs {
	case NotStarted:
		return "Not Started"
	case KeyGenComplete:
		return "Key Generation Complete"
	case PermissioningComplete:
		return "Permissioning Identity Complete"
	case UDBComplete:
		return "User Discovery Identity Complete"
	default:
		return fmt.Sprintf("Unknown registration state %v", uint32(rs))
	}
}

// creates a registration status from binary data
func regStatusUnmarshalBinary(b []byte) RegistrationStatus {
	return RegistrationStatus(binary.BigEndian.Uint32(b))
}

// returns the binary representation of the registration status
func (rs RegistrationStatus) marshalBinary() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(rs))
	return b
}

// RegStatus returns the registration status as stored in the
// kv
func (s *session) RegStatus() (RegistrationStatus, error) {
	var status RegistrationStatus
	obj, err := s.syncKV.Get(registrationStatusKey,
		currentRegistrationStatusVersion)
	if err == nil {
		status = regStatusUnmarshalBinary(obj.Data)
	} else {
		err = errors.Wrapf(err, "could not load RegStatus")
	}
	return status, err
}

// creates a new registration status and stores it
func (s *session) newRegStatus() error {
	regStatus := NotStarted

	curStatus, err := s.RegStatus()
	if err == nil {
		jww.WARN.Printf("RegStatus is already created: %s",
			curStatus)
		return nil
	}

	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentRegistrationStatusVersion,
		Timestamp: now,
		Data:      regStatus.marshalBinary(),
	}

	err = s.syncKV.Set(registrationStatusKey, &obj)
	if err != nil {
		return errors.WithMessagef(err, "Failed to store new "+
			"registration status")
	}

	return nil
}

// sets the registration status to the passed status if it is greater than the
// current stats, otherwise returns an error
func (s *session) ForwardRegistrationStatus(
	regStatus RegistrationStatus) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	oldStatus, err := s.RegStatus()
	if err != nil {
		return err
	}

	if regStatus <= oldStatus {
		return errors.Errorf("Cannot set registration status to a "+
			"status before the current stats: Current: %s, New: %s",
			oldStatus, regStatus)
	}

	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentRegistrationStatusVersion,
		Timestamp: now,
		Data:      regStatus.marshalBinary(),
	}
	err = s.syncKV.Set(registrationStatusKey, &obj)
	if err != nil {
		return errors.Wrapf(err, "Failed to store registration status")
	}

	return nil
}
