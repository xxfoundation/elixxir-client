////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/auth"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/e2e/rekey"
)

// params.go define the high level parameters structures (which embed E2E and
// CMIX params respectively) that are passed down into the core xxdk modules.

// CMIXParams contains the parameters for Network tracking and for specific CMIX
// messaging settings.
//
// Example JSON:
//
//	{
//	  "Network": {
//	    "TrackNetworkPeriod": 1000000000,
//	    "MaxCheckedRounds": 500,
//	    "RegNodesBufferLen": 1000,
//	    "NetworkHealthTimeout": 15000000000,
//	    "ParallelNodeRegistrations": 20,
//	    "KnownRoundsThreshold": 1500,
//	    "FastPolling": true,
//	    "VerboseRoundTracking": false,
//	    "RealtimeOnly": false,
//	    "ReplayRequests": true,
//	    "Rounds": {
//	      "MaxHistoricalRounds": 100,
//	      "HistoricalRoundsPeriod": 100000000,
//	      "HistoricalRoundsBufferLen": 1000,
//	      "MaxHistoricalRoundsRetries": 3
//	    },
//	    "Pickup": {
//	      "NumMessageRetrievalWorkers": 2,
//	      "LookupRoundsBufferLen": 100000,
//	      "MaxHistoricalRoundsRetries": 2,
//	      "UncheckRoundPeriod": 120000000000,
//	      "ForceMessagePickupRetry": false,
//	      "SendTimeout": 3000000000,
//	      "RealtimeOnly": false,
//	      "ForceHistoricalRounds": false
//	    },
//	    "Message": {
//	      "MessageReceptionBuffLen": 500,
//	      "MessageReceptionWorkerPoolSize": 2,
//	      "MaxChecksInProcessMessage": 10,
//	      "InProcessMessageWait": 900000000000,
//	      "RealtimeOnly": false
//	    },
//	    "Historical": {
//	      "MaxHistoricalRounds": 100,
//	      "HistoricalRoundsPeriod": 100000000,
//	      "HistoricalRoundsBufferLen": 1000,
//	      "MaxHistoricalRoundsRetries": 3
//	    },
//	    "MaxParallelIdentityTracks": 5,
//	    "EnableImmediateSending": false
//	  },
//	  "CMIX": {
//	    "RoundTries": 10,
//	    "Timeout": 45000000000,
//	    "RetryDelay": 1000000000,
//	    "SendTimeout": 3000000000,
//	    "DebugTag": "External",
//	    "BlacklistedNodes": {},
//	    "Critical": false
//	  }
//	}
//
// FIXME: this breakdown could be cleaner and is an unfortunate side effect of
//
//	several refactors of the codebase.
type CMIXParams struct {
	Network cmix.Params
	CMIX    cmix.CMIXParams
}

// E2EParams holds all the settings for e2e and it's various submodules.
//
// Note that Base wraps cmix.CMIXParams to control message send params, so that
// xxdk library users should copy the desired settings to both.
// FIXME: this should not wrap a copy of cmix.CMIXParams.
type E2EParams struct {
	Session        session.Params
	Base           e2e.Params
	Rekey          rekey.Params
	EphemeralRekey rekey.Params
	Auth           auth.Params
}

////////////////////////////////////////////////////////////////////////////////
// CMix Params Helper Functions                                               //
////////////////////////////////////////////////////////////////////////////////

// GetDefaultCMixParams returns a new CMIXParams with the default parameters.
func GetDefaultCMixParams() CMIXParams {
	return CMIXParams{
		Network: cmix.GetDefaultParams(),
		CMIX:    cmix.GetDefaultCMIXParams(),
	}
}

// Unmarshal fills an empty object with the deserialized contents of the JSON
// data.
func (p *CMIXParams) Unmarshal(jsonData []byte) error {
	return json.Unmarshal(jsonData, p)
}

// Marshal creates JSON data of the object.
func (p *CMIXParams) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

////////////////////////////////////////////////////////////////////////////////
// E2E Params Helper Functions                                                //
////////////////////////////////////////////////////////////////////////////////

// GetDefaultE2EParams returns a new E2EParams with the default parameters.
func GetDefaultE2EParams() E2EParams {
	return E2EParams{
		Session:        session.GetDefaultParams(),
		Base:           e2e.GetDefaultParams(),
		Rekey:          rekey.GetDefaultParams(),
		EphemeralRekey: rekey.GetDefaultEphemeralParams(),
		Auth:           auth.GetDefaultParams(),
	}
}

// Unmarshal fills an empty object with the deserialized contents of the JSON
// data.
func (p *E2EParams) Unmarshal(jsonData []byte) error {
	return json.Unmarshal(jsonData, p)
}

// Marshal creates JSON data of the object.
func (p *E2EParams) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
