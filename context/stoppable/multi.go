package stoppable

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"sync/atomic"
	"time"
)

type Multi struct {
	stoppables []Stoppable
	name       string
	running    uint32
	mux        sync.RWMutex
}

//returns a new multi stoppable
func NewMulti(name string) *Multi {
	return &Multi{
		name:    name,
		running: 1,
	}
}

// returns true if the thread is still running
func (m *Multi) IsRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

// adds the given stoppable to the list of stoppables
func (m *Multi) Add(stoppable Stoppable) {
	m.mux.Lock()
	m.stoppables = append(m.stoppables, stoppable)
	m.mux.Unlock()
}

// returns the name of the multi stoppable and the names of all stoppables it
// contains
func (m *Multi) Name() string {
	m.mux.RLock()
	names := m.name + ": {"
	for _, s := range m.stoppables {
		names += s.Name() + ", "
	}
	if len(m.stoppables) > 0 {
		names = names[:len(names)-2]
	}
	names += "}"
	m.mux.RUnlock()

	return names
}

// closes all child stoppers. It does not return their errors and assumes they
// print them to the log
func (m *Multi) Close(timeout time.Duration) error {
	if !m.IsRunning() {
		return nil
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	numErrors := uint32(0)

	wg := &sync.WaitGroup{}

	for _, stoppable := range m.stoppables {
		wg.Add(1)
		go func() {
			if stoppable.Close(timeout) != nil {
				atomic.AddUint32(&numErrors, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()

	atomic.StoreUint32(&m.running, 0)

	if numErrors > 0 {
		errStr := fmt.Sprintf("MultiStopper %s failed to close "+
			"%v/%v stoppers", m.name, numErrors, len(m.stoppables))
		jww.ERROR.Println(errStr)
		return errors.New(errStr)
	}

	return nil
}
