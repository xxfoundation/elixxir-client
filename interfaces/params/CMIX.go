///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"encoding/json"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"time"
)

type CMIX struct {
	// maximum number of rounds to try and send on
	RoundTries     uint
	Timeout        time.Duration
	RetryDelay     time.Duration
	ExcludedRounds excludedRounds.ExcludedRounds

	// Duration to wait before sending on a round times out and a new round is
	// tried
	SendTimeout time.Duration

	// an alternate identity preimage to use on send. If not set, the default
	// for the sending identity will be used
	IdentityPreimage []byte

	// Tag which prints with sending logs to help localize the source
	// All internal sends are tagged, so the default tag is "External"
	DebugTag string
}

func GetDefaultCMIX() CMIX {
	return CMIX{
		RoundTries:  10,
		Timeout:     25 * time.Second,
		RetryDelay:  1 * time.Second,
		SendTimeout: 3 * time.Second,
		DebugTag:    "External",
	}
}

func (c CMIX) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// GetCMIXParameters func obtains default CMIX parameters, or overrides with given parameters if set
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
