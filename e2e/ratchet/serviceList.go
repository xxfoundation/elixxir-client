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

func (r *Ratchet) add(m *partner.Manager) {
	r.servicesmux.RLock()
	defer r.servicesmux.RUnlock()
	for tag, process := range r.services {
		r.sInteface.AddService(r.defaultID, m.MakeService(tag), process)
	}
}

func (r *Ratchet) delete(m *partner.Manager) {
	r.servicesmux.RLock()
	defer r.servicesmux.RUnlock()
	for tag, process := range r.services {
		r.sInteface.DeleteService(r.defaultID, m.MakeService(tag), process)
	}
}

func (r *Ratchet) AddService(tag string, processor message.Processor) error {
	r.servicesmux.Lock()
	defer r.servicesmux.Unlock()
	//add the services to the list
	if _, exists := r.services[tag]; exists {
		return errors.Errorf("Cannot add more than one service '%s'", tag)
	}
	r.services[tag] = processor

	//add a service for every manager
	for _, m := range r.managers {
		r.sInteface.AddService(r.defaultID, m.MakeService(tag), processor)
	}

	return nil
}

func (r *Ratchet) RemoveService(tag string) error {
	r.servicesmux.Lock()
	defer r.servicesmux.Unlock()

	oldServiceProcess, exists := r.services[tag]
	if !exists {
		return errors.Errorf("Cannot remove a service that doesnt "+
			"exist: '%s'", tag)
	}

	delete(r.services, tag)

	for _, m := range r.managers {
		r.sInteface.DeleteService(r.defaultID, m.MakeService(tag), oldServiceProcess)
	}

	return nil
}
