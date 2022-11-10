////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/xx_network/primitives/id"
)

// Services is a subsection of the cmix.Manager interface used for services
type Services interface {
	AddService(
		clientID *id.ID, newService message.Service, response message.Processor)
	DeleteService(
		clientID *id.ID, toDelete message.Service, processor message.Processor)
}

func (r *Ratchet) add(m partner.Manager) {
	r.servicesMux.RLock()
	defer r.servicesMux.RUnlock()
	for tag, process := range r.services {
		r.sInterface.AddService(r.myID, m.MakeService(tag), process)
	}
}

func (r *Ratchet) delete(m partner.Manager) {
	r.servicesMux.RLock()
	defer r.servicesMux.RUnlock()
	for tag, process := range r.services {
		r.sInterface.DeleteService(r.myID, m.MakeService(tag), process)
	}
}

func (r *Ratchet) AddService(tag string, processor message.Processor) error {
	r.servicesMux.Lock()
	defer r.servicesMux.Unlock()
	//add the services to the list
	if _, exists := r.services[tag]; exists {
		return errors.Errorf("Cannot add more than one service '%s'", tag)
	}
	r.services[tag] = processor

	//add a service for every manager
	for _, m := range r.managers {
		r.sInterface.AddService(r.myID, m.MakeService(tag), processor)
	}

	return nil
}

func (r *Ratchet) RemoveService(tag string) error {
	r.servicesMux.Lock()
	defer r.servicesMux.Unlock()

	oldServiceProcess, exists := r.services[tag]
	if !exists {
		return errors.Errorf("Cannot remove a service that doesnt "+
			"exist: '%s'", tag)
	}

	delete(r.services, tag)

	for _, m := range r.managers {
		r.sInterface.DeleteService(r.myID, m.MakeService(tag), oldServiceProcess)
	}

	return nil
}
