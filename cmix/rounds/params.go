////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/json"
	"time"
)

// Params contains the parameters for the rounds package.
type Params struct {
	// MaxHistoricalRounds is the number of historical rounds required to
	// automatically send a historical rounds query.
	MaxHistoricalRounds uint

	// HistoricalRoundsPeriod is the maximum period of time a pending historical
	// round query will wait before it is transmitted.
	HistoricalRoundsPeriod time.Duration

	// HistoricalRoundsBufferLen is the length of historical rounds channel
	// buffer.
	HistoricalRoundsBufferLen uint

	// MaxHistoricalRoundsRetries is the maximum number of times a historical
	// round lookup will be attempted.
	MaxHistoricalRoundsRetries uint
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	MaxHistoricalRounds        uint
	HistoricalRoundsPeriod     time.Duration
	HistoricalRoundsBufferLen  uint
	MaxHistoricalRoundsRetries uint
}

// GetDefaultParams returns a default set of Params.
func GetDefaultParams() Params {
	return Params{
		MaxHistoricalRounds:        100,
		HistoricalRoundsPeriod:     100 * time.Millisecond,
		HistoricalRoundsBufferLen:  1000,
		MaxHistoricalRoundsRetries: 3,
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
func (r Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		MaxHistoricalRounds:        r.MaxHistoricalRounds,
		HistoricalRoundsPeriod:     r.HistoricalRoundsPeriod,
		HistoricalRoundsBufferLen:  r.HistoricalRoundsBufferLen,
		MaxHistoricalRoundsRetries: r.MaxHistoricalRoundsRetries,
	}

	return json.Marshal(&pDisk)

}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (r *Params) UnmarshalJSON(data []byte) error {
	pDisk := paramsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*r = Params{
		MaxHistoricalRounds:        pDisk.MaxHistoricalRounds,
		HistoricalRoundsPeriod:     pDisk.HistoricalRoundsPeriod,
		HistoricalRoundsBufferLen:  pDisk.HistoricalRoundsBufferLen,
		MaxHistoricalRoundsRetries: pDisk.MaxHistoricalRoundsRetries,
	}

	return nil
}
