///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const regCodeKey = "regCode"
const regCodeVersion = 0

// SetNDF stores a network definition json file
func (s *Session) SetRegCode(regCode string) {
	if err := s.Set(regCodeKey,
		&versioned.Object{
			Version:   regCodeVersion,
			Data:      []byte(regCode),
			Timestamp: time.Now(),
		}); err != nil {
		jww.FATAL.Panicf("Failed to set the registration code: %s", err)
	}
}

// Returns the stored network definition json file
func (s *Session) GetRegCode() (string, error) {
	regCode, err := s.Get(regCodeKey)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to load the regcode")
	}
	return string(regCode.Data), nil
}
