package api

import (
	"fmt"
	"github.com/pkg/errors"
	"sync/atomic"
)

type Status int

const (
	Stopped  Status = 0
	Starting Status = 1000
	Running  Status = 2000
	Stopping Status = 3000
)

func (s Status) String() string {
	switch s {
	case Stopped:
		return "Stopped"
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case Stopping:
		return "Stopping"
	default:
		return fmt.Sprintf("Unknown status %d", s)
	}
}

type statusTracker struct {
	s *uint32
}

func newStatusTracker() *statusTracker {
	s := uint32(Stopped)
	return &statusTracker{s: &s}
}

func (s *statusTracker) toStarting() error {
	if !atomic.CompareAndSwapUint32(s.s, uint32(Stopped), uint32(Starting)) {
		return errors.Errorf("Failed to move to '%s' status, at '%s', "+
			"must be at '%s' for transition", Starting, atomic.LoadUint32(s.s),
			Stopped)
	}
	return nil
}

func (s *statusTracker) toRunning() error {
	if !atomic.CompareAndSwapUint32(s.s, uint32(Starting), uint32(Running)) {
		return errors.Errorf("Failed to move to '%s' status, at '%s', "+
			"must be at '%s' for transition", Running, atomic.LoadUint32(s.s),
			Starting)
	}
	return nil
}

func (s *statusTracker) toStopping() error {
	if !atomic.CompareAndSwapUint32(s.s, uint32(Running), uint32(Stopping)) {
		return errors.Errorf("Failed to move to '%s' status, at '%s',"+
			" must be at '%s' for transition", Stopping, atomic.LoadUint32(s.s),
			Running)
	}
	return nil
}

func (s *statusTracker) toStopped() error {
	if !atomic.CompareAndSwapUint32(s.s, uint32(Stopping), uint32(Stopped)) {
		return errors.Errorf("Failed to move to '%s' status, at '%s',"+
			" must be at '%s' for transition", Stopped, atomic.LoadUint32(s.s),
			Stopping)
	}
	return nil
}

func (s *statusTracker) get() Status {
	return Status(atomic.LoadUint32(s.s))
}
