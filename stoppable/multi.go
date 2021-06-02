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
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Error message.
const closeMultiErr = "MultiStopper %s failed to close %d/%d stoppers"

type Multi struct {
	stoppables []Stoppable
	name       string
	running    uint32
	mux        sync.RWMutex
	once       sync.Once
}

// NewMulti returns a new multi Stoppable.
func NewMulti(name string) *Multi {
	return &Multi{
		name:    name,
		running: running,
	}
}

// IsRunning returns true if stoppable is marked as running.
func (m *Multi) IsRunning() bool {
	return atomic.LoadUint32(&m.running) == running
}

// Add adds the given stoppable to the list of stoppables.
func (m *Multi) Add(stoppable Stoppable) {
	m.mux.Lock()
	m.stoppables = append(m.stoppables, stoppable)
	m.mux.Unlock()
}

// Name returns the name of the Multi Stoppable and the names of all stoppables
// it contains.
func (m *Multi) Name() string {
	m.mux.RLock()
	defer m.mux.RUnlock()

	names := make([]string, len(m.stoppables))
	for i, s := range m.stoppables {
		names[i] = s.Name()
	}

	return m.name + ": {" + strings.Join(names, ", ") + "}"
}

// Close closes all child stoppers. It does not return their errors and assumes
// they print them to the log.
func (m *Multi) Close(timeout time.Duration) error {
	var err error
	m.once.Do(
		func() {
			atomic.StoreUint32(&m.running, stopped)

			var numErrors uint32
			var wg sync.WaitGroup

			m.mux.Lock()
			for _, stoppable := range m.stoppables {
				wg.Add(1)
				go func(stoppable Stoppable) {
					if stoppable.Close(timeout) != nil {
						atomic.AddUint32(&numErrors, 1)
					}
					wg.Done()
				}(stoppable)
			}
			m.mux.Unlock()

			wg.Wait()

			if numErrors > 0 {
				err = errors.Errorf(
					closeMultiErr, m.name, numErrors, len(m.stoppables))
				jww.ERROR.Print(err.Error())
			}
		})

	return err
}
