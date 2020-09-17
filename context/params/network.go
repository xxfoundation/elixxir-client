package params

import (
	"time"
)

type Network struct {
	TrackNetworkPeriod  time.Duration
	NumWorkers          int
	MaxHistoricalRounds int
	MaxCheckCnt         int
}

func GetDefaultNetwork() Network {
	return Network{
		TrackNetworkPeriod:  100 * time.Millisecond,
		NumWorkers:          4,
		MaxHistoricalRounds: 100,
		MaxCheckCnt:         100,
	}
}
