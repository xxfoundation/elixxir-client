package ratchet

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/xx_network/primitives/id"
)

// Services is a subsection of the network.Manager interface used for services
type Services interface {
	AddService(AddService *id.ID, newService message.Service,
		response message.Processor)
	DeleteService(clientID *id.ID, toDelete message.Service,
		processor message.Processor)
}

func (s *Store) add(m *partner.Manager) {
	s.servicesmux.RLock()
	defer s.servicesmux.RUnlock()
	for tag, process := range s.services {
		s.sInteface.AddService(s.myID, m.MakeService(tag), process)
	}
}

func (s *Store) delete(m *partner.Manager) {
	s.servicesmux.RLock()
	defer s.servicesmux.RUnlock()
	for tag, process := range s.services {
		s.sInteface.DeleteService(s.myID, m.MakeService(tag), process)
	}
}

func (s *Store) AddService(tag string, processor message.Processor) error {
	s.servicesmux.Lock()
	defer s.servicesmux.Unlock()
	//add the services to the list
	if _, exists := s.services[tag]; exists {
		return errors.Errorf("Cannot add more than one service '%s'", tag)
	}
	s.services[tag] = processor

	//add a service for every manager
	for _, m := range s.managers {
		s.sInteface.AddService(s.myID, m.MakeService(tag), processor)
	}

	return nil
}

func (s *Store) RemoveService(tag string) error {
	s.servicesmux.Lock()
	defer s.servicesmux.Unlock()

	oldServiceProcess, exists := s.services[tag]
	if !exists {
		return errors.Errorf("Cannot remove a service that doesnt "+
			"exist: '%s'", tag)
	}

	delete(s.services, tag)

	for _, m := range s.managers {
		s.sInteface.DeleteService(s.myID, m.MakeService(tag), oldServiceProcess)
	}

	return nil
}
