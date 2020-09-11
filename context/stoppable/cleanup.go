package stoppable

import (
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

// Wraps any stoppable and runs a callback after to stop for cleanup behavior
// the cleanup is run under the remainder of the timeout but will not be canceled
// if the timeout runs out
// the cleanup function does not run if the thread does not stop
type Cleanup struct {
	stop Stoppable
	// the clean function receives how long it has to run before the timeout,
	// this is nto expected to be used in most cases
	clean   func(duration time.Duration) error
	running uint32
	once    sync.Once
}

// Creates a new cleanup from the passed stoppable and the function
func NewCleanup(stop Stoppable, clean func(duration time.Duration) error) *Cleanup {
	return &Cleanup{
		stop:    stop,
		clean:   clean,
		running: 0,
	}
}

// returns true if the thread is still running and its cleanup has completed
func (c *Cleanup) IsRunning() bool {
	return atomic.LoadUint32(&c.running) == 1
}

// returns the name of the stoppable denoting it has cleanup
func (c *Cleanup) Name() string {
	return c.stop.Name() + " with cleanup"
}

// stops the contained stoppable and runs the cleanup function after.
// the cleanup function does not run if the thread does not stop
func (c *Cleanup) Close(timeout time.Duration) error {
	var err error

	c.once.Do(
		func() {
			defer atomic.StoreUint32(&c.running, 0)
			start := time.Now()

			//run the stopable
			if err := c.stop.Close(timeout); err != nil {
				err = errors.WithMessagef(err, "Cleanup for %s not executed",
					c.stop.Name())
				return
			}

			//run the cleanup function with the remaining time as a timeout
			elapsed := time.Since(start)

			complete := make(chan error, 1)
			go func() {
				complete <- c.clean(elapsed)
			}()

			timer := time.NewTimer(elapsed)

			select {
			case err := <-complete:
				if err != nil {
					err = errors.WithMessagef(err, "Cleanup for %s "+
						"failed", c.stop.Name())
				}
			case <-timer.C:
				err = errors.Errorf("Clean up for %s timedout", c.stop.Name())
			}
		})

	return err
}
