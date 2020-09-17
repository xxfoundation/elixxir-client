package stoppable

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"sync/atomic"
	"time"
)

// Single allows stopping a single goroutine using a channel.
// It adheres to the stoppable interface.
type Single struct {
	name    string
	quit    chan struct{}
	running uint32
	once    sync.Once
}

// NewSingle returns a new single stoppable.
func NewSingle(name string) *Single {
	return &Single{
		name:    name,
		quit:    make(chan struct{}),
		running: 1,
	}
}

// IsRunning returns true if the thread is still running.
func (s *Single) IsRunning() bool {
	return atomic.LoadUint32(&s.running) == 1
}

// Quit returns the read only channel it will send the stop signal on.
func (s *Single) Quit() <-chan struct{} {
	return s.quit
}

// Name returns the name of the thread. This is designed to be
func (s *Single) Name() string {
	return s.name
}

// Close signals the thread to time out and closes if it is still running.
func (s *Single) Close(timeout time.Duration) error {
	var err error
	s.once.Do(func() {
		atomic.StoreUint32(&s.running, 0)
		timer := time.NewTimer(timeout)
		select {
		case <-timer.C:
			jww.ERROR.Printf("Stopper for %s failed to stop after "+
				"timeout of %s", s.name, timeout)
			err = errors.Errorf("%s failed to close", s.name)
		case s.quit <- struct{}{}:
		}
	})
	return err
}
