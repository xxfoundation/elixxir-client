///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"time"
)

type Historical struct {
	// Number of historical rounds required to automatically send a historical
	// rounds query
	MaxHistoricalRounds uint
	// Maximum period of time a pending historical round query will wait before
	// it is transmitted
	HistoricalRoundsPeriod time.Duration

	// Length of historical rounds channel buffer
	HistoricalRoundsBufferLen uint

	// Maximum number of times a historical round lookup will be attempted
	MaxHistoricalRoundsRetries uint
}

func GetDefaultHistorical() Historical {
	return Historical{
		MaxHistoricalRounds:    100,
		HistoricalRoundsPeriod: 100 * time.Millisecond,

		HistoricalRoundsBufferLen:  1000,
		MaxHistoricalRoundsRetries: 3,
	}
}
