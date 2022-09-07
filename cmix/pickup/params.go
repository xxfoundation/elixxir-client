////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"encoding/json"
	"time"
)

// Params contains the parameters for the pickup package.
type Params struct {
	// Number of worker threads for retrieving messages from gateways
	NumMessageRetrievalWorkers uint

	// Length of round lookup channel buffer
	LookupRoundsBufferLen uint

	// Maximum number of times a historical round lookup will be attempted
	MaxHistoricalRoundsRetries uint

	// Interval between checking for rounds in UncheckedRoundStore due for a
	// message retrieval retry
	UncheckRoundPeriod time.Duration

	// Toggles if message pickup retrying mechanism if forced
	// by intentionally not looking up messages
	ForceMessagePickupRetry bool

	// Duration to wait before sending on a round times out and a new round is
	// tried
	SendTimeout time.Duration

	// Disables all attempts to pick up dropped or missed messages
	RealtimeOnly bool

	// Toggles if historical rounds should always be used
	ForceHistoricalRounds bool
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	NumMessageRetrievalWorkers uint
	LookupRoundsBufferLen      uint
	MaxHistoricalRoundsRetries uint
	UncheckRoundPeriod         time.Duration
	ForceMessagePickupRetry    bool
	SendTimeout                time.Duration
	RealtimeOnly               bool
	ForceHistoricalRounds      bool
}

// GetDefaultParams returns a default set of Params.
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
		NumMessageRetrievalWorkers: p.NumMessageRetrievalWorkers,
		LookupRoundsBufferLen:      p.LookupRoundsBufferLen,
		MaxHistoricalRoundsRetries: p.MaxHistoricalRoundsRetries,
		UncheckRoundPeriod:         p.UncheckRoundPeriod,
		ForceMessagePickupRetry:    p.ForceMessagePickupRetry,
		SendTimeout:                p.SendTimeout,
		RealtimeOnly:               p.RealtimeOnly,
		ForceHistoricalRounds:      p.ForceHistoricalRounds,
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
		NumMessageRetrievalWorkers: pDisk.NumMessageRetrievalWorkers,
		LookupRoundsBufferLen:      pDisk.LookupRoundsBufferLen,
		MaxHistoricalRoundsRetries: pDisk.MaxHistoricalRoundsRetries,
		UncheckRoundPeriod:         pDisk.UncheckRoundPeriod,
		ForceMessagePickupRetry:    pDisk.ForceMessagePickupRetry,
		SendTimeout:                pDisk.SendTimeout,
		RealtimeOnly:               pDisk.RealtimeOnly,
		ForceHistoricalRounds:      pDisk.ForceHistoricalRounds,
	}

	return nil
}
