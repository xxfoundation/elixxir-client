////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"os"
	"time"
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

// stringer for Registration Status
func (rs RegistrationStatus) String() string {
	switch rs {
	case NotStarted:
		return "Not Started"
	case KeyGenComplete:
		return "Key Generation Complete"
	case PermissioningComplete:
		return "Permissioning Registration Complete"
	case UDBComplete:
		return "User Discovery Registration Complete"
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

// loads the registration status from disk. If the status cannot be found, it
// defaults to Not Started
func (s *Session) loadOrCreateRegStatus() error {
	obj, err := s.Get(registrationStatusKey)
	if err != nil {
		if os.IsNotExist(err) {
			// set at not started but do not save until it is updated
			s.regStatus = NotStarted
			return nil
		} else {
			return errors.WithMessagef(err, "Failed to load registration status")
		}
	}
	s.regStatus = regStatusUnmarshalBinary(obj.Data)
	return nil
}

// sets the registration status to the passed status if it is greater than the
// current stats, otherwise returns an error
func (s *Session) ForwardRegistrationStatus(regStatus RegistrationStatus) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if regStatus <= s.regStatus {
		return errors.Errorf("Cannot set registration status to a "+
			"status before the current stats: Current: %s, New: %s",
			s.regStatus, regStatus)
	}

	now := time.Now()

	obj := versioned.Object{
		Version:   currentRegistrationStatusVersion,
		Timestamp: now,
		Data:      regStatus.marshalBinary(),
	}

	err := s.Set(registrationStatusKey, &obj)
	if err != nil {
		return errors.WithMessagef(err, "Failed to store registration status")
	}

	s.regStatus = regStatus
	return nil
}

// sets the registration status to the passed status if it is greater than the
// current stats, otherwise returns an error
func (s *Session) GetRegistrationStatus() RegistrationStatus {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.regStatus
}
