////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix/message"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// manager.AddService
	errServiceExists = "service with tag %q already exists"

	// manager.RemoveService
	errServiceNotExists = "service with tag %q does not exist"
)

func (m *manager) AddService(tag string, p Processor) error {
	m.servicesMux.Lock()
	defer m.servicesMux.Unlock()

	if _, exists := m.services[tag]; exists {
		// Return an error if the service already exists
		return errors.Errorf(errServiceExists, tag)
	} else {
		// Add the service to the list
		m.services[tag] = p
	}

	// Add a service for every group
	for _, g := range m.gs.Groups() {
		newService := makeService(g.ID, tag)
		m.getCMix().AddService(m.getReceptionIdentity().ID, newService,
			&receptionProcessor{m, g, p})
	}

	return nil
}

func (m *manager) RemoveService(tag string) error {
	m.servicesMux.Lock()
	defer m.servicesMux.Unlock()

	// Delete the service from the list
	oldProcess, exists := m.services[tag]
	if exists {
		return errors.Errorf(errServiceNotExists, tag)
	} else {
		delete(m.services, tag)
	}

	// Delete service for every group
	for _, g := range m.gs.Groups() {
		toDelete := makeService(g.ID, tag)
		m.getCMix().DeleteService(m.getReceptionIdentity().ID, toDelete,
			&receptionProcessor{m, g, oldProcess})
	}

	return nil
}

// addAllServices adds every service for the given group.
func (m *manager) addAllServices(g gs.Group) {
	jww.INFO.Printf("Adding service for group %s", g.ID)
	for tag, p := range m.services {
		newService := makeService(g.ID, tag)
		m.getCMix().AddService(m.getReceptionIdentity().ID, newService,
			&receptionProcessor{m, g, p})
	}
}

// deleteAllServices deletes every service for the given group.
func (m *manager) deleteAllServices(groupID *id.ID) {
	for tag := range m.services {
		toDelete := makeService(groupID, tag)
		m.getCMix().DeleteService(m.getReceptionIdentity().ID, toDelete, nil)
	}
}

func makeService(groupID *id.ID, tag string) message.Service {
	return message.Service{
		Identifier: groupID[:],
		Tag:        makeServiceTag(tag),
		Metadata:   groupID[:],
	}
}

func makeServiceTag(tag string) string {
	return catalog.Group + "-" + tag
}
