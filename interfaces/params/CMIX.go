///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import "time"

type CMIX struct {
	//maximum number of rounds to try and send on
	RoundTries uint
	Timeout    time.Duration
	RetryDelay time.Duration
}

func GetDefaultCMIX() CMIX {
	return CMIX{
		RoundTries: 10,
		Timeout:    25 * time.Second,
		RetryDelay: 1 * time.Second,
	}
}
