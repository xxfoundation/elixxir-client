///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"sync/atomic"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

// Cleanup wraps any stoppable and runs a callback after to stop for cleanup
// behavior. The cleanup is run under the remainder of the timeout but will not
// be canceled if the timeout runs out. The cleanup function does not run if the
// thread does not stop.
type Cleanup struct {
	stop Stoppable
	// the clean function receives how long it has to run before the timeout,
	// this is nto expected to be used in most cases
	clean   func(duration time.Duration) error
	running uint32
	once    sync.Once
}

// NewCleanup creates a new Cleanup from the passed stoppable and function.
func NewCleanup(stop Stoppable, clean func(duration time.Duration) error) *Cleanup {
	return &Cleanup{
		stop:    stop,
		clean:   clean,
		running: 0,
	}
}

// IsRunning returns true if the thread is still running and its cleanup has
// completed.
func (c *Cleanup) IsRunning() bool {
	return atomic.LoadUint32(&c.running) == 1
}

// Name returns the name of the stoppable denoting it has cleanup.
func (c *Cleanup) Name() string {
	return c.stop.Name() + " with cleanup"
}

// Close stops the contained stoppable and runs the cleanup function after. The
// cleanup function does not run if the thread does not stop.
func (c *Cleanup) Close(timeout time.Duration) error {
	var err error

	c.once.Do(
		func() {
			defer atomic.StoreUint32(&c.running, 0)
			start := netTime.Now()

			// Run the stoppable
			if err := c.stop.Close(timeout); err != nil {
				err = errors.WithMessagef(err, "Cleanup for %s not executed",
					c.stop.Name())
				return
			}

			// Run the cleanup function with the remaining time as a timeout
			elapsed := time.Since(start)

			complete := make(chan error, 1)
			go func() {
				complete <- c.clean(elapsed)
			}()

			timer := time.NewTimer(elapsed)

			select {
			case err := <-complete:
				if err != nil {
					err = errors.WithMessagef(err, "Cleanup for %s failed",
						c.stop.Name())
				}
			case <-timer.C:
				err = errors.Errorf("Clean up for %s timeout", c.stop.Name())
			}
		})

	if err!=nil{
		jww.ERROR.Printf(err.Error())
	}

	return err
}
