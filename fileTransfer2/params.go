////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmix"
	"time"
)

const (
	defaultMaxThroughput = 150_000 // 150 kB per second
	defaultSendTimeout   = 500 * time.Millisecond
)

// Params contains parameters used for file transfer.
type Params struct {
	// MaxThroughput is the maximum data transfer speed to send file parts (in
	// bytes per second)
	MaxThroughput int

	// SendTimeout is the duration, in nanoseconds, before sending on a round
	// times out. It is recommended that SendTimeout is not changed from its
	// default.
	SendTimeout time.Duration

	// Cmix are the parameters used when sending a cMix message.
	Cmix cmix.CMIXParams
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	MaxThroughput int
	SendTimeout   time.Duration
	Cmix          cmix.CMIXParams
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		MaxThroughput: defaultMaxThroughput,
		SendTimeout:   defaultSendTimeout,
		Cmix:          cmix.GetDefaultCMIXParams(),
	}
}

// GetParameters returns the default network parameters, or override with given
// parameters, if set.
func GetParameters(params string) (Params, error) {
	p := DefaultParams()
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
		MaxThroughput: p.MaxThroughput,
		SendTimeout:   p.SendTimeout,
		Cmix:          p.Cmix,
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
		MaxThroughput: pDisk.MaxThroughput,
		SendTimeout:   pDisk.SendTimeout,
		Cmix:          pDisk.Cmix,
	}

	return nil
}
