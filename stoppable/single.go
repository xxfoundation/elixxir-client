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
	"sync"
	"sync/atomic"
	"time"
)

// Error message.
const closeTimeoutErr = "stopper for %s failed to stop after timeout of %s"

// Single allows stopping a single goroutine using a channel. It adheres to the
// Stoppable interface.
type Single struct {
	name    string
	quit    chan struct{}
	running uint32
	once    sync.Once
}

// NewSingle returns a new single Stoppable.
func NewSingle(name string) *Single {
	return &Single{
		name:    name,
		quit:    make(chan struct{}),
		running: running,
	}
}

// IsRunning returns true if stoppable is marked as running.
func (s *Single) IsRunning() bool {
	return atomic.LoadUint32(&s.running) == running
}

// Quit returns a receive-only channel that will be triggered when the Stoppable
// quits.
func (s *Single) Quit() <-chan struct{} {
	return s.quit
}

// Name returns the name of the Single Stoppable.
func (s *Single) Name() string {
	return s.name
}

// Close signals the Single to close via the quit channel. Returns an error if
// sending on the quit channel times out.
func (s *Single) Close(timeout time.Duration) error {
	var err error
	s.once.Do(func() {
		select {
		case <-time.NewTimer(timeout).C:
			err = errors.Errorf(closeTimeoutErr, s.name, timeout)
			jww.ERROR.Print(err.Error())
		case s.quit <- struct{}{}:
		}
		atomic.StoreUint32(&s.running, stopped)
	})

	return err
}
