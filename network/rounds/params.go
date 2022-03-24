package rounds

import "time"

type Params struct {
	// Number of worker threads for retrieving messages from gateways
	NumMessageRetrievalWorkers uint

	// Length of round lookup channel buffer
	LookupRoundsBufferLen uint

	// Maximum number of times a historical round lookup will be attempted
	MaxHistoricalRoundsRetries uint

	// Interval between checking for rounds in UncheckedRoundStore
	// due for a message retrieval retry
	UncheckRoundPeriod time.Duration

	// Toggles if message pickup retrying mechanism if forced
	// by intentionally not looking up messages
	ForceMessagePickupRetry bool

	// Duration to wait before sending on a round times out and a new round is
	// tried
	SendTimeout time.Duration

	//disables all attempts to pick up dropped or missed messages
	RealtimeOnly bool

	// Toggles if historical rounds should always be used
	ForceHistoricalRounds bool
}

func GetDefaultParams() Params {
	return Params{
		NumMessageRetrievalWorkers: 8,
		LookupRoundsBufferLen:      2000,
		MaxHistoricalRoundsRetries: 3,
		UncheckRoundPeriod:         20 * time.Second,
		ForceMessagePickupRetry:    false,
		SendTimeout:                3 * time.Second,
		RealtimeOnly:               false,
	}
}
