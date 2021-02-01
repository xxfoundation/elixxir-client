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

	Rounds
	Messages
	Rekey
}

func GetDefaultNetwork() Network {
	n := Network{
		TrackNetworkPeriod:   100 * time.Millisecond,
		MaxCheckedRounds:     500,
		RegNodesBufferLen:    500,
		NetworkHealthTimeout: 30 * time.Second,
	}
	n.Rounds = GetDefaultRounds()
	n.Messages = GetDefaultMessage()
	return n
}

func (n Network) Marshal() ([]byte, error) {
	return json.Marshal(n)
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
