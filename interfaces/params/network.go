///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"encoding/json"
	"time"
)

type Network struct {
	TrackNetworkPeriod time.Duration
	// maximum number of rounds to check in a single iterations network updates
	MaxCheckedRounds uint
	// Size of the buffer of nodes to register
	RegNodesBufferLen uint
	// Longest delay between network events for Health tracker to denote that
	// the network is in a bad state
	NetworkHealthTimeout time.Duration
	//Number of parallel node registration the client is capable of
	ParallelNodeRegistrations uint
	//How far back in rounds the network should actually check
	KnownRoundsThreshold uint
	// Determines verbosity of network updates while polling
	// If true, client receives a filtered set of updates
	// If false, client receives the full list of network updates
	FastPolling bool
	// Messages will not be sent to Rounds containing these Nodes
	BlacklistedNodes []string
	// Determines if the state of every round processed is tracked in ram.
	// This is very memory intensive and is primarily used for debugging
	VerboseRoundTracking bool
	//disables all attempts to pick up dropped or missed messages
	RealtimeOnly bool
	// Resends auth requests up the stack if received multiple times
	ReplayRequests bool

	Rounds
	Messages
	Rekey

	E2EParams E2ESessionParams
}

func GetDefaultNetwork() Network {
	n := Network{
		TrackNetworkPeriod:        100 * time.Millisecond,
		MaxCheckedRounds:          500,
		RegNodesBufferLen:         1000,
		NetworkHealthTimeout:      30 * time.Second,
		E2EParams:                 GetDefaultE2ESessionParams(),
		ParallelNodeRegistrations: 20,
		KnownRoundsThreshold:      1500, //5 rounds/sec * 60 sec/min * 5 min
		FastPolling:               true,
		BlacklistedNodes:          make([]string, 0),
		VerboseRoundTracking:      false,
		RealtimeOnly:              false,
		ReplayRequests:            true,
	}
	n.Rounds = GetDefaultRounds()
	n.Messages = GetDefaultMessage()
	n.Rekey = GetDefaultRekey()
	return n
}

func (n Network) Marshal() ([]byte, error) {
	return json.Marshal(n)
}

func (n Network) SetRealtimeOnlyAll() Network {
	n.RealtimeOnly = true
	n.Rounds.RealtimeOnly = true
	n.Messages.RealtimeOnly = true
	return n
}

// Obtain default Network parameters, or override with given parameters if set
func GetNetworkParameters(params string) (Network, error) {
	p := GetDefaultNetwork()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Network{}, err
		}
	}
	return p, nil
}
