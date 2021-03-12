package api

import (
	"gitlab.com/elixxir/client/stoppable"
	"sync"
)

// a service process starts itself in a new thread, returning from the
// originator a stopable to control it
type ServiceProcess func() stoppable.Stoppable

type serviceProcessiesList struct {
	serviceProcessies []ServiceProcess
	multiStopable     *stoppable.Multi
	mux               sync.Mutex
}

// newServiceProcessiesList creates a new processies list which will add its
// processies to the passed mux
func newServiceProcessiesList(m *stoppable.Multi) *serviceProcessiesList {
	return &serviceProcessiesList{
		serviceProcessies: make([]ServiceProcess, 0),
		multiStopable:     m,
	}
}

// Add adds the service process to the list and adds it to the multi-stopable
func (spl serviceProcessiesList) Add(sp ServiceProcess) {
	spl.mux.Lock()
	defer spl.mux.Unlock()

	spl.serviceProcessies = append(spl.serviceProcessies, sp)
	// starts the process and adds it to the stopable
	// there can be a race condition between the execution of the process and
	// the stopable.
	spl.multiStopable.Add(sp())
}

// Runs all processies, to be used after a stop. Must use a new stopable
func (spl serviceProcessiesList) run(m *stoppable.Multi) {
	spl.mux.Lock()
	defer spl.mux.Unlock()

	spl.multiStopable = m
	for _, sp := range spl.serviceProcessies {
		spl.multiStopable.Add(sp())
	}
}
