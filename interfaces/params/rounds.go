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
	// maximum number of times to attempt to retrieve a round from a gateway
	// before giving up on it
	MaxAttemptsCheckingARound uint
	// number of historical rounds required to automatically send a historical
	// rounds query
	MaxHistoricalRounds uint
	// maximum period of time a pending historical round query will wait before
	// it si transmitted
	HistoricalRoundsPeriod time.Duration
	// number of worker threads for retreiving messages from gateways
	NumMessageRetrievalWorkers uint

	//Length of historical rounds channel buffer
	HistoricalRoundsBufferLen uint
	//Length of round lookup channel buffer
	LookupRoundsBufferLen uint
}

func GetDefaultRounds() Rounds {
	return Rounds{
		MaxAttemptsCheckingARound:  5,
		MaxHistoricalRounds:        100,
		HistoricalRoundsPeriod:     100 * time.Millisecond,
		NumMessageRetrievalWorkers: 8,

		HistoricalRoundsBufferLen: 1000,
		LookupRoundsBufferLen:     2000,
	}
}
