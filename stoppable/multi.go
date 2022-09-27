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
	"sync"
	"sync/atomic"
)

// Error message.
const closeMultiErr = "multi stoppable %q failed to close %d/%d stoppables"

type Multi struct {
	stoppables []Stoppable
	name       string
	mux        sync.RWMutex
	once       sync.Once
}

// NewMulti returns a new multi Stoppable.
func NewMulti(name string) *Multi {
	return &Multi{
		name: name,
	}
}

// Add adds the given Stoppable to the list of stoppables.
func (m *Multi) Add(stoppable Stoppable) {
	m.mux.Lock()
	m.stoppables = append(m.stoppables, stoppable)
	m.mux.Unlock()
}

// Name returns the name of the Multi Stoppable and the names of all stoppables
// it contains.
func (m *Multi) Name() string {
	m.mux.RLock()

	names := make([]string, len(m.stoppables))
	for i, s := range m.stoppables {
		names[i] = s.Name()
	}

	m.mux.RUnlock()

	return m.name + "{" + strings.Join(names, ", ") + "}"
}

// GetStatus returns the lowest status of all of the Stoppable children. The
// status is not the status of all Stoppables, but the status of the Stoppable
// with the lowest status.
func (m *Multi) GetStatus() Status {
	lowestStatus := Stopped
	m.mux.RLock()

	for i := range m.stoppables {
		s := m.stoppables[i]
		status := s.GetStatus()
		if status < lowestStatus {
			lowestStatus = status
			jww.INFO.Printf("Stoppable %s has status %s",
				s.Name(), status.String())
		}
	}

	m.mux.RUnlock()

	return lowestStatus
}

// GetRunningProcesses returns the names of all running processes at the time
// of this call. Note that this list may change and is subject to race
// conditions if multiple threads are in the process of starting or stopping.
func (m *Multi) GetRunningProcesses() []string {
	m.mux.RLock()

	runningProcesses := make([]string, 0)
	for i := range m.stoppables {
		s := m.stoppables[i]
		status := s.GetStatus()
		if status < Stopped {
			runningProcesses = append(runningProcesses, s.Name())
		}
	}

	m.mux.RUnlock()

	return runningProcesses
}

// IsRunning returns true if Stoppable is marked as running.
func (m *Multi) IsRunning() bool {
	return m.GetStatus() == Running
}

// IsStopping returns true if Stoppable is marked as stopping.
func (m *Multi) IsStopping() bool {
	return m.GetStatus() == Stopping
}

// IsStopped returns true if Stoppable is marked as stopped.
func (m *Multi) IsStopped() bool {
	return m.GetStatus() == Stopped
}

// Close issues a close signal to all child stoppables and marks the status of
// the Multi Stoppable as stopping. Returns an error if one or more child
// stoppables failed to close, but it does not return their specific errors and
// assumes they print them to the log.
func (m *Multi) Close() error {
	var numErrors uint32

	m.once.Do(func() {
		var wg sync.WaitGroup

		jww.INFO.Printf("Sending on quit channel to multi stoppable %q with processes: %v.",
			m.Name(), m.GetRunningProcesses())

		m.mux.Lock()
		// Attempt to stop each stoppable in its own goroutine
		for _, stoppable := range m.stoppables {
			wg.Add(1)
			go func(stoppable Stoppable) {
				if stoppable.Close() != nil {
					atomic.AddUint32(&numErrors, 1)
				}
				wg.Done()
			}(stoppable)
		}
		m.mux.Unlock()

		wg.Wait()
	})

	if numErrors > 0 {
		err := errors.Errorf(closeMultiErr, m.name, numErrors, len(m.stoppables))
		jww.ERROR.Print(err.Error())
		return err
	}

	return nil
}
