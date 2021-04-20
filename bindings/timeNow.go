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

type TimeSource interface {
	NowMs() int
}

// SetTimeSource sets the network time to a custom source.
func SetTimeSource(timeNow TimeSource) {
	netTime.Now = func() time.Time {
		return time.Unix(0, int64(timeNow.NowMs()*int(time.Millisecond)))
	}
}
