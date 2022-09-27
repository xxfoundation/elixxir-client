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
	"sync"
	"sync/atomic"
)

// Error message.
const toStoppingErr = "failed to set the status of single stoppable %q to " +
	"stopped when status is %s instead of %s"

// Single allows stopping a single goroutine using a channel. It adheres to the
// Stoppable interface.
type Single struct {
	name   string
	quit   chan struct{}
	status Status
	once   sync.Once
}

// NewSingle returns a new single Stoppable.
func NewSingle(name string) *Single {
	return &Single{
		name:   name,
		quit:   make(chan struct{}, 1),
		status: Running,
	}
}

// Name returns the name of the Single Stoppable.
func (s *Single) Name() string {
	return s.name
}

// GetStatus returns the status of the Stoppable.
func (s *Single) GetStatus() Status {
	return Status(atomic.LoadUint32((*uint32)(&s.status)))
}

// IsRunning returns true if Stoppable is marked as running.
func (s *Single) IsRunning() bool {
	return s.GetStatus() == Running
}

// IsStopping returns true if Stoppable is marked as stopping.
func (s *Single) IsStopping() bool {
	return s.GetStatus() == Stopping
}

// IsStopped returns true if Stoppable is marked as stopped.
func (s *Single) IsStopped() bool {
	return s.GetStatus() == Stopped
}

// toStopping changes the status from running to stopping. An error is returned
// if the status is not already set to running.
func (s *Single) toStopping() error {
	if !atomic.CompareAndSwapUint32((*uint32)(&s.status), uint32(Running), uint32(Stopping)) {
		return errors.Errorf(toStoppingErr, s.Name(), s.GetStatus(), Running)
	}

	jww.INFO.Printf("Switched status of single stoppable %q from %s to %s.",
		s.Name(), Running, Stopping)

	return nil
}

// ToStopped changes the status from stopping to stopped. Panics if the status
// is not already set to stopping.
func (s *Single) ToStopped() {
	if !atomic.CompareAndSwapUint32((*uint32)(&s.status), uint32(Stopping), uint32(Stopped)) {
		jww.FATAL.Panicf("Failed to set the status of single stoppable %q to "+
			"stopped when status is %s instead of %s.",
			s.Name(), s.GetStatus(), Stopping)
	}

	jww.INFO.Printf("Switched status of single stoppable %q from %s to %s.",
		s.Name(), Stopping, Stopped)
}

// Quit returns a receive-only channel that will be triggered when the Stoppable
// quits.
func (s *Single) Quit() <-chan struct{} {
	return s.quit
}

// Close signals the Single to close via the quit channel. Returns an error if
// the status of the Single is not Running.
func (s *Single) Close() error {
	var err error

	s.once.Do(func() {
		// Attempt to set status to stopping or return an error if unable
		err = s.toStopping()
		if err != nil {
			return
		}

		jww.TRACE.Printf("Sending on quit channel to single stoppable %q.",
			s.Name())

		// Send on quit channel
		s.quit <- struct{}{}
	})

	if err != nil {
		jww.ERROR.Print(err.Error())
	}

	return err
}
