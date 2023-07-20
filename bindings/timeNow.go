///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// TimeSource is a copy of [netTime.TimeSource]. For some reason, Go bindings
// only allows this interface, not the one found in netTime.
type TimeSource interface {
	NowMs() int64
}

// SetTimeSource will set the time source that will be used when retrieving the
// current time using [netTime.Now]. This should be called BEFORE Login()
// and only be called once. Using this after Login is undefined behavior that
// may result in a crash.
//
// Parameters:
//   - timeNow is an object which adheres to [netTime.TimeSource]. Specifically,
//     this object should a NowMs() method which return a 64-bit integer value.
func SetTimeSource(timeNow TimeSource) {
	netTime.SetTimeSource(timeNow)
}

// SetOffset will set an internal offset variable. All calls to [netTime.Now]
// will have this offset applied to this value.
//
// Parameters:
//   - offset is a time by which netTime.Now will be offset. This value may be
//     negative or positive. This expects a 64-bit integer value which will
//     represent the number in microseconds this offset will be.
func SetOffset(offset int64) {
	netTime.SetOffset(time.Duration(offset) * time.Microsecond)
}
