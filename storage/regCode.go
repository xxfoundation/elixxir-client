////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const regCodeKey = "regCode"
const regCodeVersion = 0

// SetNDF stores a network definition json file
func (s *session) SetRegCode(regCode string) {
	if err := s.syncKV.Set(regCodeKey,
		&versioned.Object{
			Version:   regCodeVersion,
			Timestamp: netTime.Now(),
			Data:      []byte(regCode),
		}); err != nil {
		jww.FATAL.Panicf("Failed to set the registration code: %s", err)
	}
}

// Returns the stored network definition json file
func (s *session) GetRegCode() (string, error) {
	regCode, err := s.syncKV.Get(regCodeKey, regCodeVersion)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to load the regcode")
	}
	return string(regCode.Data), nil
}
