package params

import (
	"time"
)

type Network struct {
	TrackNetworkPeriod time.Duration
	NumWorkers         uint
	// maximum number of rounds to check in a single iterations network updates
	MaxCheckCheckedRounds uint
	Rounds
}

func GetDefaultNetwork() Network {
	n := Network{
		TrackNetworkPeriod:    100 * time.Millisecond,
		NumWorkers:            4,
		MaxCheckCheckedRounds: 500,
	}
	n.Rounds = GetDefaultRounds()
	return n
}
