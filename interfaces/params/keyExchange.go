///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import "time"

type Rekey struct {
	RoundTimeout time.Duration
}

func GetDefaultRekey() Rekey {
	return Rekey{
		RoundTimeout: time.Minute,
	}
}
