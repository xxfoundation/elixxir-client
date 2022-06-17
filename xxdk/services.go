package xxdk

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/stoppable"
	"sync"
	"time"
)

// a service process starts itself in a new thread, returning from the
// originator a stopable to control it
type Service func() (stoppable.Stoppable, error)

type services struct {
	services  []Service
	stoppable *stoppable.Multi
	state     Status
	mux       sync.Mutex
}

// newServiceProcessiesList creates a new services list which will add its
// services to the passed mux
func newServices() *services {
	return &services{
		services:  make([]Service, 0),
		stoppable: stoppable.NewMulti("services"),
		state:     Stopped,
	}
}

// Add adds the service process to the list and adds it to the multi-stopable.
// Start running it if services are running
func (s *services) add(sp Service) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	//append the process to the list
	s.services = append(s.services, sp)

	//if services are running, start the process
	if s.state == Running {
		stop, err := sp()
		if err != nil {
			return errors.WithMessage(err, "Failed to start added service")
		}
		s.stoppable.Add(stop)
	}
	return nil
}

// Runs all services. If they are in the process of stopping,
// it will wait for the stop to complete or the timeout to ellapse
// Will error if already running
func (s *services) start(timeout time.Duration) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	//handle various states
	switch s.state {
	case Stopped:
		break
	case Running:
		return errors.New("Cannot start services when already Running")
	case Stopping:
		err := stoppable.WaitForStopped(s.stoppable, timeout)
		if err != nil {
			return errors.Errorf("Procesies did not all stop within %s, "+
				"unable to start services: %+v", timeout, err)
		}
	}

	//create a new stopable
	s.stoppable = stoppable.NewMulti(followerStoppableName)

	//start all services and register with the stoppable
	for _, sp := range s.services {
		stop, err := sp()
		if err != nil {
			return errors.WithMessage(err, "Failed to start added service")
		}
		s.stoppable.Add(stop)
	}

	s.state = Running

	return nil
}

// Stops all currently running services. Will return an
// error if the state is not "running"
func (s *services) stop() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.state != Running {
		return errors.Errorf("cannot stop services when they "+
			"are not Running, services are: %s", s.state)
	}

	s.state = Stopping

	if err := s.stoppable.Close(); err != nil {
		return errors.WithMessage(err, "Failed to stop services")
	}

	s.state = Stopped

	return nil
}

// returns the current state of services
func (s *services) status() Status {
	s.mux.Lock()
	defer s.mux.Unlock()

	return s.state
}
