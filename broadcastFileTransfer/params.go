////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmix"
	"time"
)

const (
	defaultMaxThroughput = 150_000 // 150 kB per second
	defaultSendTimeout   = 500 * time.Millisecond
	defaultResendWait    = 30 * time.Second
)

// Params contains parameters used for file transfer.
type Params struct {
	// MaxThroughput is the maximum data transfer speed to send file parts (in
	// bytes per second). If set to 0, rate limiting will be disabled.
	MaxThroughput int

	// SendTimeout is the duration, in nanoseconds, before sending on a round
	// times out. It is recommended that SendTimeout is not changed from its
	// default.
	SendTimeout time.Duration

	// ResendWait is the duration, in nanoseconds, to wait to receive a file
	// part before resending it.
	ResendWait time.Duration

	// Cmix are the parameters used when sending a cMix message.
	Cmix cmix.CMIXParams
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		MaxThroughput: defaultMaxThroughput,
		SendTimeout:   defaultSendTimeout,
		ResendWait:    defaultResendWait,
		Cmix:          cmix.GetDefaultCMIXParams(),
	}
}

// GetParameters returns the default network parameters, or override with given
// parameters, if set. Returns an error if provided invalid JSON. If the JSON is
// valid but does not match the Params structure, the default parameters will be
// returned.
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
