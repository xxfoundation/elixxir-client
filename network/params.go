package network

import (
	"encoding/json"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type Params struct {
	TrackNetworkPeriod time.Duration
	// maximum number of rounds to check in a single iterations network updates
	MaxCheckedRounds uint
	// Size of the buffer of nodes to register
	RegNodesBufferLen uint
	// Longest delay between network events for health tracker to denote that
	// the network is in a bad state
	NetworkHealthTimeout time.Duration
	//Number of parallel nodes registration the client is capable of
	ParallelNodeRegistrations uint
	//How far back in rounds the network should actually check
	KnownRoundsThreshold uint
	// Determines verbosity of network updates while polling
	// If true, client receives a filtered set of updates
	// If false, client receives the full list of network updates
	FastPolling bool
	// Determines if the state of every round processed is tracked in ram.
	// This is very memory intensive and is primarily used for debugging
	VerboseRoundTracking bool
	//disables all attempts to pick up dropped or missed messages
	RealtimeOnly bool
	// Resends auth requests up the stack if received multiple times
	ReplayRequests bool

	Rounds     rounds.Params
	Message    message.Params
	Historical historical.Params
}

func GetDefaultParams() Params {
	n := Params{
		TrackNetworkPeriod:        100 * time.Millisecond,
		MaxCheckedRounds:          500,
		RegNodesBufferLen:         1000,
		NetworkHealthTimeout:      30 * time.Second,
		ParallelNodeRegistrations: 20,
		KnownRoundsThreshold:      1500, //5 rounds/sec * 60 sec/min * 5 min
		FastPolling:               true,
		VerboseRoundTracking:      false,
		RealtimeOnly:              false,
		ReplayRequests:            true,
	}
	n.Rounds = rounds.GetDefaultParams()
	n.Message = message.GetDefaultParams()
	n.Historical = historical.GetDefaultParams()

	return n
}

func (n Params) Marshal() ([]byte, error) {
	return json.Marshal(n)
}

func (n Params) SetRealtimeOnlyAll() Params {
	n.RealtimeOnly = true
	n.Rounds.RealtimeOnly = true
	n.Message.RealtimeOnly = true
	return n
}

// Obtain default Network parameters, or override with given parameters if set
func GetParameters(params string) (Params, error) {
	p := GetDefaultParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Params{}, err
		}
	}
	return p, nil
}

type CMIXParams struct {
	// maximum number of rounds to try and send on
	RoundTries     uint
	Timeout        time.Duration
	RetryDelay     time.Duration
	ExcludedRounds excludedRounds.ExcludedRounds

	// Duration to wait before sending on a round times out and a new round is
	// tried
	SendTimeout time.Duration

	// an alternate identity preimage to use on send. If not set, the default
	// for the sending identity will be used
	IdentityPreimage []byte

	// Tag which prints with sending logs to help localize the source
	// All internal sends are tagged, so the default tag is "External"
	DebugTag string

	//Threading interface, can be used to stop the send early
	Stop *stoppable.Single

	//List of nodes to not send to, will skip a round with these
	//nodes in it
	BlacklistedNodes map[id.ID]interface{}
}

func GetDefaultCMIX() CMIXParams {
	return CMIXParams{
		RoundTries:  10,
		Timeout:     25 * time.Second,
		RetryDelay:  1 * time.Second,
		SendTimeout: 3 * time.Second,
		DebugTag:    "External",
	}
}

func (c CMIXParams) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// GetCMIXParameters func obtains default CMIX parameters, or overrides with given parameters if set
func GetCMIXParameters(params string) (CMIXParams, error) {
	p := GetDefaultCMIX()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return CMIXParams{}, err
		}
	}
	return p, nil
}
