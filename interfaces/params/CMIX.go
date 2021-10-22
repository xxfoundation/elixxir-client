///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"encoding/json"
	"time"
)

type CMIX struct {
	//maximum number of rounds to try and send on
	RoundTries uint
	Timeout    time.Duration
	RetryDelay time.Duration
	// an alternate identity preimage to use on send. If not set, the default
	// for the sending identity will be used
	IdentityPreimage []byte
}

func GetDefaultCMIX() CMIX {
	return CMIX{
		RoundTries: 10,
		Timeout:    25 * time.Second,
		RetryDelay: 1 * time.Second,
	}
}

func (c CMIX) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Obtain default CMIX parameters, or override with given parameters if set
func GetCMIXParameters(params string) (CMIX, error) {
	p := GetDefaultCMIX()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return CMIX{}, err
		}
	}
	return p, nil
}
