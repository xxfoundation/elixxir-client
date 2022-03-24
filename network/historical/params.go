package historical

import "time"

type Params struct {
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

func GetDefaultParams() Params {
	return Params{
		MaxHistoricalRounds:    100,
		HistoricalRoundsPeriod: 100 * time.Millisecond,

		HistoricalRoundsBufferLen:  1000,
		MaxHistoricalRoundsRetries: 3,
	}
}
