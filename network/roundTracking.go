////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"sort"
	"sync"
)

type RoundState uint8

const (
	Unchecked = iota
	Unknown
	Checked
	Abandoned
)

func (rs RoundState) String() string {
	switch rs {
	case Unchecked:
		return "Unchecked"
	case Unknown:
		return "Unknown"
	case Checked:
		return "Checked"
	case Abandoned:
		return "Abandoned"
	default:
		return fmt.Sprintf("Unregistered Round State: %d", rs)
	}
}

type RoundTracker struct {
	state map[id.Round]RoundState
	mux   sync.Mutex
}

func NewRoundTracker() *RoundTracker {
	return &RoundTracker{
		state: make(map[id.Round]RoundState),
	}
}

func (rt *RoundTracker) denote(rid id.Round, state RoundState) {
	rt.mux.Lock()
	defer rt.mux.Unlock()
	rt.state[rid] = state
}

func (rt *RoundTracker) String() string {
	rt.mux.Lock()
	defer rt.mux.Unlock()
	keys := make([]int, 0, len(rt.state))
	for key := range rt.state {
		keys = append(keys, int(key))
	}

	sort.Ints(keys)

	stringification := ""
	for _, key := range keys {
		stringification += fmt.Sprintf("Round: %d, state:%s \n", key, rt.state[id.Round(key)])
	}

	return stringification
}
