////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"strings"
	"time"
)

// Error message returned after a comms operations ends and finds that its
// parent thread is stopping or stopped.
const (
	errKey     = "[StoppableNotRunning]"
	ErrMsg     = "stoppable %q is not running, exiting %s early " + errKey
	timeoutErr = "timed out after %s waiting for the stoppable to stop for %q"
)

// pollPeriod is the duration to wait between polls to see of stoppables are
// stopped.
const pollPeriod = 100 * time.Millisecond

// Stoppable interface for stopping a goroutine. All functions are thread safe.
type Stoppable interface {
	// Name returns the name of the Stoppable.
	Name() string

	// GetStatus returns the status of the Stoppable.
	GetStatus() Status

	// IsRunning returns true if the Stoppable is running.
	IsRunning() bool

	// IsStopping returns true if Stoppable is marked as stopping.
	IsStopping() bool

	// IsStopped returns true if Stoppable is marked as stopped.
	IsStopped() bool

	// Close marks the Stoppable as stopping and issues a close signal to the
	// Stoppable or any children it may have.
	Close() error
}

// WaitForStopped polls the stoppable and all its children to see if they are
// stopped. Returns an error if its times out waiting for all children to stop.
func WaitForStopped(s Stoppable, timeout time.Duration) error {
	done := make(chan struct{})

	// Launch the processes to check if all stoppables are stopped in separate
	// goroutine so that when the timeout is reached, no time is wasted exiting
	go func() {
		for !s.IsStopped() {
			time.Sleep(pollPeriod)
		}

		select {
		case done <- struct{}{}:
		case <-time.NewTimer(50 * time.Millisecond).C:
		}
	}()

	select {
	case <-done:
		jww.INFO.Printf("All stoppables have stopped for %q.", s.Name())
		return nil
	case <-time.NewTimer(timeout).C:
		return errors.Errorf(timeoutErr, timeout, s.Name())
	}
}

// CheckErr returns true if the error contains a stoppable error message. This
// function is used by callers to determine if a sub function quit due to a
// stoppable closing and tells the caller to exit.
func CheckErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), errKey)
}
