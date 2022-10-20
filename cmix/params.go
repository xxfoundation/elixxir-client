////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/pickup"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
)

type Params struct {
	TrackNetworkPeriod time.Duration
	// MaxCheckedRounds is the maximum number of rounds to check in a single
	// iterations network updates.
	MaxCheckedRounds uint

	// RegNodesBufferLen is the size of the buffer of nodes to register.
	RegNodesBufferLen uint

	// NetworkHealthTimeout is the longest delay between network events for
	// health tracker to denote that the network is in a bad state.
	NetworkHealthTimeout time.Duration

	// ParallelNodeRegistrations is the number of parallel node registrations
	// that the client is capable of.
	ParallelNodeRegistrations uint

	// KnownRoundsThreshold dictates how far back in rounds the network should
	// actually check.
	KnownRoundsThreshold uint

	// FastPolling determines verbosity of network updates while polling. If
	// true, client receives a filtered set of updates. If false, client
	// receives the full list of network updates.
	FastPolling bool

	// VerboseRoundTracking determines if the state of every round processed is
	// tracked in memory. This is very memory intensive and is primarily used
	// for debugging.
	VerboseRoundTracking bool

	// RealtimeOnly disables all attempts to pick up dropped or missed messages.
	RealtimeOnly bool

	// ReplayRequests Resends auth requests up the stack if received multiple
	// times.
	ReplayRequests bool

	// MaxParallelIdentityTracks is the maximum number of parallel identities
	// the system will poll in one iteration of the follower
	MaxParallelIdentityTracks uint

	// ClockSkewClamp is the window (+/-) in which clock skew is
	// ignored and local time is used
	ClockSkewClamp time.Duration

	Rounds     rounds.Params
	Pickup     pickup.Params
	Message    message.Params
	Historical rounds.Params
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	TrackNetworkPeriod        time.Duration
	MaxCheckedRounds          uint
	RegNodesBufferLen         uint
	NetworkHealthTimeout      time.Duration
	ParallelNodeRegistrations uint
	KnownRoundsThreshold      uint
	FastPolling               bool
	VerboseRoundTracking      bool
	RealtimeOnly              bool
	ReplayRequests            bool
	Rounds                    rounds.Params
	Pickup                    pickup.Params
	Message                   message.Params
	Historical                rounds.Params
	MaxParallelIdentityTracks uint
}

// GetDefaultParams returns a Params object containing the
// default parameters.
func GetDefaultParams() Params {
	n := Params{
		TrackNetworkPeriod:        100 * time.Millisecond,
		MaxCheckedRounds:          500,
		RegNodesBufferLen:         1000,
		NetworkHealthTimeout:      15 * time.Second,
		ParallelNodeRegistrations: 5,
		KnownRoundsThreshold:      1500, // 5 rounds/sec * 60 sec/min * 5 min
		FastPolling:               true,
		VerboseRoundTracking:      false,
		RealtimeOnly:              false,
		ReplayRequests:            true,
		MaxParallelIdentityTracks: 20,
		ClockSkewClamp:            50 * time.Millisecond,
	}
	n.Rounds = rounds.GetDefaultParams()
	n.Pickup = pickup.GetDefaultParams()
	n.Message = message.GetDefaultParams()
	n.Historical = rounds.GetDefaultParams()

	return n
}

// GetParameters returns the default Params, or override with given
// parameters, if set.
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

// MarshalJSON adheres to the json.Marshaler interface.
func (p Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		TrackNetworkPeriod:        p.TrackNetworkPeriod,
		MaxCheckedRounds:          p.MaxCheckedRounds,
		RegNodesBufferLen:         p.RegNodesBufferLen,
		NetworkHealthTimeout:      p.NetworkHealthTimeout,
		ParallelNodeRegistrations: p.ParallelNodeRegistrations,
		KnownRoundsThreshold:      p.KnownRoundsThreshold,
		FastPolling:               p.FastPolling,
		VerboseRoundTracking:      p.VerboseRoundTracking,
		RealtimeOnly:              p.RealtimeOnly,
		ReplayRequests:            p.ReplayRequests,
		Rounds:                    p.Rounds,
		Pickup:                    p.Pickup,
		Message:                   p.Message,
		Historical:                p.Historical,
		MaxParallelIdentityTracks: p.MaxParallelIdentityTracks,
	}

	return json.Marshal(&pDisk)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (p *Params) UnmarshalJSON(data []byte) error {
	pDisk := paramsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*p = Params{
		TrackNetworkPeriod:        pDisk.TrackNetworkPeriod,
		MaxCheckedRounds:          pDisk.MaxCheckedRounds,
		RegNodesBufferLen:         pDisk.RegNodesBufferLen,
		NetworkHealthTimeout:      pDisk.NetworkHealthTimeout,
		ParallelNodeRegistrations: pDisk.ParallelNodeRegistrations,
		KnownRoundsThreshold:      pDisk.KnownRoundsThreshold,
		FastPolling:               pDisk.FastPolling,
		VerboseRoundTracking:      pDisk.VerboseRoundTracking,
		RealtimeOnly:              pDisk.RealtimeOnly,
		ReplayRequests:            pDisk.ReplayRequests,
		Rounds:                    pDisk.Rounds,
		Pickup:                    pDisk.Pickup,
		Message:                   pDisk.Message,
		Historical:                pDisk.Historical,
		MaxParallelIdentityTracks: pDisk.MaxParallelIdentityTracks,
	}

	return nil
}

func (p Params) SetRealtimeOnlyAll() Params {
	p.RealtimeOnly = true
	p.Pickup.RealtimeOnly = true
	p.Message.RealtimeOnly = true
	return p
}

const DefaultDebugTag = "External"

type CMIXParams struct {
	// RoundTries is the maximum number of rounds to try to send on
	RoundTries     uint
	Timeout        time.Duration
	RetryDelay     time.Duration
	ExcludedRounds excludedRounds.ExcludedRounds `json:"-"`

	// SendTimeout is the duration to wait before sending on a round times out
	// and a new round is tried.
	SendTimeout time.Duration

	// DebugTag is a tag that is printed with sending logs to help localize the
	// source. All internal sends are tagged, so the default tag is "External".
	DebugTag string

	// Stop can be used to stop the send early.
	Stop *stoppable.Single `json:"-"`

	// BlacklistedNodes is a list of nodes to not send to; will skip a round
	// with these nodes in it.
	BlacklistedNodes NodeMap

	// Critical indicates if the message is critical. The system will track that
	// the round it sends on completes and will auto resend in the event the
	// round fails or completion cannot be determined. The sent data will be
	// byte identical, so this has a high chance of metadata leak. This system
	// should only be used in cases where repeats cannot be different. Only used
	// in sendCmix, not sendManyCmix.
	Critical bool
}

// cMixParamsDisk will be the marshal-able and umarshal-able object.
type cMixParamsDisk struct {
	RoundTries       uint
	Timeout          time.Duration
	RetryDelay       time.Duration
	SendTimeout      time.Duration
	DebugTag         string
	BlacklistedNodes NodeMap
	Critical         bool
}

func GetDefaultCMIXParams() CMIXParams {
	return CMIXParams{
		RoundTries:  10,
		Timeout:     45 * time.Second,
		RetryDelay:  1 * time.Second,
		SendTimeout: 3 * time.Second,
		DebugTag:    DefaultDebugTag,
		// Unused stoppable so components that require one have a channel to
		// wait on
		Stop: stoppable.NewSingle("cmixParamsDefault"),
	}
}

// GetCMIXParameters obtains default CMIX parameters, or overrides with given
// parameters if set.
func GetCMIXParameters(params string) (CMIXParams, error) {
	p := GetDefaultCMIXParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return CMIXParams{}, err
		}
	}
	return p, nil
}

// MarshalJSON adheres to the json.Marshaler interface.
func (p CMIXParams) MarshalJSON() ([]byte, error) {
	pDisk := cMixParamsDisk{
		RoundTries:       p.RoundTries,
		Timeout:          p.Timeout,
		RetryDelay:       p.RetryDelay,
		SendTimeout:      p.SendTimeout,
		DebugTag:         p.DebugTag,
		Critical:         p.Critical,
		BlacklistedNodes: p.BlacklistedNodes,
	}

	return json.Marshal(&pDisk)

}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (p *CMIXParams) UnmarshalJSON(data []byte) error {
	pDisk := cMixParamsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*p = CMIXParams{
		RoundTries:       pDisk.RoundTries,
		Timeout:          pDisk.Timeout,
		RetryDelay:       pDisk.RetryDelay,
		SendTimeout:      pDisk.SendTimeout,
		DebugTag:         pDisk.DebugTag,
		Critical:         pDisk.Critical,
		BlacklistedNodes: pDisk.BlacklistedNodes,
	}

	return nil
}

// NodeMap represents a map of nodes and whether they have been
// blacklisted. This is designed for use with CMIXParams.BlacklistedNodes
type NodeMap map[id.ID]bool

// MarshalJSON adheres to the json.Marshaler interface.
func (nm NodeMap) MarshalJSON() ([]byte, error) {
	stringMap := make(map[string]bool, len(nm))
	for nid, b := range nm {
		stringMap[base64.StdEncoding.EncodeToString(nid.Marshal())] = b
	}

	return json.Marshal(stringMap)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (nm *NodeMap) UnmarshalJSON(data []byte) error {
	stringMap := make(map[string]bool)
	err := json.Unmarshal(data, &stringMap)
	if err != nil {
		return err
	}

	newNM := make(NodeMap)
	for nidString, bString := range stringMap {
		nidBytes, err := base64.StdEncoding.DecodeString(nidString)
		if err != nil {
			return err
		}
		nid, err := id.Unmarshal(nidBytes)
		if err != nil {
			return err
		}

		newNM[*nid] = bString
	}

	*nm = newNM

	return nil
}
