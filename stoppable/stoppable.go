///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import "time"

const (
	stopped = 0
	running = 1
)

// Stoppable interface for stopping a goroutine.
type Stoppable interface {
	Close(timeout time.Duration) error
	IsRunning() bool
	Name() string
}
