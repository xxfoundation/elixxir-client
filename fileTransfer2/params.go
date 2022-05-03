////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import "time"

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
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		MaxThroughput: defaultMaxThroughput,
		SendTimeout:   defaultSendTimeout,
	}
}
