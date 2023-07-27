////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"encoding/json"
	"gitlab.com/xx_network/comms/connect"
	"time"
)

// Params allows configuration of HostPool parameters.
type Params struct {
	// MaxPoolSize is the maximum number of Hosts in the HostPool.
	MaxPoolSize uint32

	// PoolSize allows override of HostPool size. Set to zero for dynamic size
	// calculation.
	PoolSize uint32

	// ProxyAttempts dictates how many proxies will be used in event of send
	// failure.
	ProxyAttempts uint32

	// MaxPings is the number of gateways to concurrently test when selecting
	// a new member of HostPool. Must be at least 1.
	MaxPings uint32

	// NumConnectionsWorkers is the number of workers connecting to gateways
	NumConnectionsWorkers int

	// MinBufferLength is the minimum length of input buffers
	// to the hostpool runner
	MinBufferLength uint32

	// EnableRotation enables the system which auto rotates/ gateways regularly.
	// This system will auto disable if the network size is less than 20.
	EnableRotation bool

	// RotationPeriod is how long until a single host is rotated.
	RotationPeriod time.Duration

	// RotationPeriodVariability is the max that the rotation
	// period can randomly deviate from the stated amount
	RotationPeriodVariability time.Duration

	// HostParams is the parameters for the creation of new Host objects.
	HostParams connect.HostParams

	// DebugPrintPeriod is an OPTIONAL field that may be set by the caller.
	// This may be set by the caller when passing Params into NewSender, as an
	// example.
	//
	// This field indicates how often debug prints are printed by the hostPool's
	// long-running thread. These debug prints will contain data on the
	// hostPool's state (refer to debug.go). If this value is not set by the user,
	// then by default debug prints will be disabled.
	DebugPrintPeriod time.Duration

	// GatewayFilter is the function which will be used to filter gateways
	// before connecting.  This must be set before initializing a HostPool and
	// cannot be changed.  If no filter is set, the defaultFilter will be used.
	GatewayFilter Filter
}

// DefaultParams returns a default set of PoolParams.
func DefaultParams() Params {
	p := Params{
		MaxPoolSize:               MaxPoolSize,
		ProxyAttempts:             5,
		PoolSize:                  0,
		MaxPings:                  5,
		NumConnectionsWorkers:     5,
		MinBufferLength:           100,
		EnableRotation:            true,
		RotationPeriod:            7 * time.Minute,
		RotationPeriodVariability: 4 * time.Minute,
		DebugPrintPeriod:          defaultPrintInterval,

		HostParams: GetDefaultHostPoolHostParams(),
	}

	return p
}

// DefaultPoolParams is a deprecated version of DefaultParams
// it does the same thing, just under a different function name
// Use DefaultParams.
func DefaultPoolParams() Params {
	return DefaultParams()
}

// GetParameters returns the default PoolParams, or
// override with given parameters, if set.
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
func (pp *Params) MarshalJSON() ([]byte, error) {
	return json.Marshal(pp)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (pp *Params) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, pp)
}

// GetDefaultHostPoolHostParams returns the default parameters used for
// hosts in the host pool
func GetDefaultHostPoolHostParams() connect.HostParams {
	hp := connect.GetDefaultHostParams()
	hp.MaxRetries = 1
	hp.MaxSendRetries = 1
	hp.AuthEnabled = false
	hp.EnableCoolOff = false
	hp.NumSendsBeforeCoolOff = 1
	hp.CoolOffTimeout = 5 * time.Minute
	hp.SendTimeout = 1000 * time.Millisecond
	hp.PingTimeout = 1000 * time.Millisecond
	hp.DisableAutoConnect = true
	return hp
}
