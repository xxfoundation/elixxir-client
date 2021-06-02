///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"sync/atomic"
	"time"
)

const nameTag = " with cleanup"

type CleanFunc func(duration time.Duration) error

// Cleanup wraps any stoppable and runs a callback after to stop for cleanup
// behavior. The cleanup is run under the remainder of the timeout but will not
// be canceled if the timeout runs out. The cleanup function does not run if the
// thread does not stop.
type Cleanup struct {
	stop Stoppable
	// clean receives how long it has to run before the timeout, this is not
	// expected to be used in most cases
	clean   CleanFunc
	running uint32
	once    sync.Once
}

// NewCleanup creates a new Cleanup from the passed in stoppable and clean
// function.
func NewCleanup(stop Stoppable, clean CleanFunc) *Cleanup {
	return &Cleanup{
		stop:    stop,
		clean:   clean,
		running: stopped,
	}
}

// IsRunning returns true if the thread is still running and its cleanup has
// completed.
func (c *Cleanup) IsRunning() bool {
	return atomic.LoadUint32(&c.running) == running
}

// Name returns the name of the stoppable denoting it has cleanup.
func (c *Cleanup) Name() string {
	return c.stop.Name() + nameTag
}

// Close stops the wrapped stoppable and after, runs the cleanup function. The
// cleanup function does not run if the thread fails to stop.
func (c *Cleanup) Close(timeout time.Duration) error {
	var err error

	c.once.Do(
		func() {
			defer atomic.StoreUint32(&c.running, stopped)
			start := netTime.Now()

			// Close each stoppable
			if err := c.stop.Close(timeout); err != nil {
				err = errors.WithMessagef(err, "Cleanup not executed for %s",
					c.stop.Name())
				return
			}

			// Run the cleanup function with the remaining time as a timeout
			elapsed := netTime.Now().Sub(start)

			complete := make(chan error, 1)
			go func() {
				complete <- c.clean(elapsed)
			}()

			select {
			case err := <-complete:
				if err != nil {
					err = errors.WithMessagef(err, "Cleanup for %s failed",
						c.stop.Name())
				}
			case <-time.NewTimer(elapsed).C:
				err = errors.Errorf("Clean up for %s timed out after %s",
					c.stop.Name(), elapsed)
			}
		})

	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	return err
}
