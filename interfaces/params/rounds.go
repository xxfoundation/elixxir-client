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

type Rounds struct {
	// Number of historical rounds required to automatically send a historical
	// rounds query
	MaxHistoricalRounds uint
	// Maximum period of time a pending historical round query will wait before
	// it is transmitted
	HistoricalRoundsPeriod time.Duration
	// Number of worker threads for retrieving messages from gateways
	NumMessageRetrievalWorkers uint

	// Length of historical rounds channel buffer
	HistoricalRoundsBufferLen uint
	// Length of round lookup channel buffer
	LookupRoundsBufferLen uint

	// Toggles if historical rounds should always be used
	ForceHistoricalRounds bool

	// Maximum number of times a historical round lookup will be attempted
	MaxHistoricalRoundsRetries uint
}

func GetDefaultRounds() Rounds {
	return Rounds{
		MaxHistoricalRounds:        100,
		HistoricalRoundsPeriod:     100 * time.Millisecond,
		NumMessageRetrievalWorkers: 8,

		HistoricalRoundsBufferLen: 1000,
		LookupRoundsBufferLen:     2000,
		ForceHistoricalRounds:     false,
		MaxHistoricalRoundsRetries: 3,
	}
}
