package historical

import "time"

type Params struct {
	// MaxHistoricalRounds is the number of historical rounds required to
	// automatically send a historical rounds query.
	MaxHistoricalRounds uint

	// HistoricalRoundsPeriod is the maximum period of time a pending historical
	// round query will wait before it is transmitted.
	HistoricalRoundsPeriod time.Duration

	// HistoricalRoundsBufferLen is the length of historical rounds channel
	// buffer.
	HistoricalRoundsBufferLen uint

	// MaxHistoricalRoundsRetries is the maximum number of times a historical
	// round lookup will be attempted.
	MaxHistoricalRoundsRetries uint
}

func GetDefaultParams() Params {
	return Params{
		MaxHistoricalRounds:        100,
		HistoricalRoundsPeriod:     100 * time.Millisecond,
		HistoricalRoundsBufferLen:  1000,
		MaxHistoricalRoundsRetries: 3,
	}
}
