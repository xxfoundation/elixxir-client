////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/message"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
)

func (m *manager) AddService(g gs.Group, tag string, p Processor) {
	newService := message.Service{
		Identifier: g.ID[:],
		Tag:        makeServiceTag(tag),
		Metadata:   g.ID[:],
	}
	m.services.AddService(m.receptionId, newService, &receptionProcessor{m, g, p})
}

func (m *manager) RemoveService(g gs.Group, tag string, p Processor) {
	toDelete := message.Service{
		Identifier: g.ID[:],
		Tag:        makeServiceTag(tag),
	}
	m.services.DeleteService(g.ID, toDelete, &receptionProcessor{m, g, p})
}

func makeServiceTag(tag string) string {
	return catalog.Group + "-" + tag
}
