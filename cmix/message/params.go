////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/json"
	"time"
)

// Params contains the parameters for the message package.
type Params struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksInProcessMessage      uint
	InProcessMessageWait           time.Duration
	RealtimeOnly                   bool
}

// paramsDisk will be the marshal-able and unmarshal-able object.
type paramsDisk struct {
	MessageReceptionBuffLen        uint          `json:"messageReceptionBuffLen"`
	MessageReceptionWorkerPoolSize uint          `json:"messageReceptionWorkerPoolSize"`
	MaxChecksInProcessMessage      uint          `json:"maxChecksInProcessMessage"`
	InProcessMessageWait           time.Duration `json:"inProcessMessageWait"`
	RealtimeOnly                   bool          `json:"realtimeOnly"`
}

// GetDefaultParams returns a Params object containing the
// default parameters.
func GetDefaultParams() Params {
	return Params{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 2,
		MaxChecksInProcessMessage:      10,
		InProcessMessageWait:           15 * time.Minute,
		RealtimeOnly:                   false,
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
		MessageReceptionBuffLen:        p.MessageReceptionBuffLen,
		MessageReceptionWorkerPoolSize: p.MessageReceptionWorkerPoolSize,
		MaxChecksInProcessMessage:      p.MaxChecksInProcessMessage,
		InProcessMessageWait:           p.InProcessMessageWait,
		RealtimeOnly:                   p.RealtimeOnly,
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
		MessageReceptionBuffLen:        pDisk.MessageReceptionBuffLen,
		MessageReceptionWorkerPoolSize: pDisk.MessageReceptionWorkerPoolSize,
		MaxChecksInProcessMessage:      pDisk.MaxChecksInProcessMessage,
		InProcessMessageWait:           pDisk.InProcessMessageWait,
		RealtimeOnly:                   pDisk.RealtimeOnly,
	}

	return nil
}
