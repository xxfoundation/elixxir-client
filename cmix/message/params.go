////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
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

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksInProcessMessage      uint
	InProcessMessageWait           time.Duration
	RealtimeOnly                   bool
}

// GetDefaultParams returns a Params object containing the
// default parameters.
func GetDefaultParams() Params {
	return Params{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
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
func (r Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		MessageReceptionBuffLen:        r.MessageReceptionBuffLen,
		MessageReceptionWorkerPoolSize: r.MessageReceptionWorkerPoolSize,
		MaxChecksInProcessMessage:      r.MaxChecksInProcessMessage,
		InProcessMessageWait:           r.InProcessMessageWait,
		RealtimeOnly:                   r.RealtimeOnly,
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
		MessageReceptionBuffLen:        pDisk.MessageReceptionBuffLen,
		MessageReceptionWorkerPoolSize: pDisk.MessageReceptionWorkerPoolSize,
		MaxChecksInProcessMessage:      pDisk.MaxChecksInProcessMessage,
		InProcessMessageWait:           pDisk.InProcessMessageWait,
		RealtimeOnly:                   pDisk.RealtimeOnly,
	}

	return nil
}
