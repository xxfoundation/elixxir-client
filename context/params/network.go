package params

import (
	"time"
)

type Network struct {
	TrackNetworkPeriod time.Duration
	// maximum number of rounds to check in a single iterations network updates
	MaxCheckedRounds uint
	//Size of the buffer of nodes to register
	RegNodesBufferLen uint

	Rounds
	Messages
}

func GetDefaultNetwork() Network {
	n := Network{
		TrackNetworkPeriod: 100 * time.Millisecond,
		MaxCheckedRounds:   500,
		RegNodesBufferLen:  500,
	}
	n.Rounds = GetDefaultRounds()
	n.Messages = GetDefaultMessage()
	return n
}
