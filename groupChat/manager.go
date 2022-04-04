///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	newGroupStoreErr = "failed to create new group store: %+v"
	joinGroupErr     = "failed to join new group %s: %+v"
	leaveGroupErr    = "failed to leave group %s: %+v"
)

// Manager handles the list of groups a user is a part of.
type Manager struct {
	e2e e2e.Handler
	net interfaces.NetworkManager

	receptionId *id.ID
	rng         *fastRNG.StreamGenerator
	grp         *cyclic.Group
	gs          *gs.Store
	services    network.Manager

	requestFunc RequestCallback
	receiveFunc ReceiveCallback
}

// NewManager creates a new group chat manager
func NewManager(services network.Manager, e2e e2e.Handler, net interfaces.NetworkManager, receptionId *id.ID,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, userDhKey *cyclic.Int,
	kv *versioned.KV, requestFunc RequestCallback, receiveFunc ReceiveCallback) (*Manager, error) {

	// Load the group chat storage or create one if one does not exist
	gStore, err := gs.NewOrLoadStore(
		kv, group.Member{ID: receptionId, DhKey: userDhKey})
	if err != nil {
		return nil, errors.Errorf(newGroupStoreErr, err)
	}

	// Define the manager object
	m := &Manager{
		e2e:         e2e,
		net:         net,
		rng:         rng,
		receptionId: receptionId,
		grp:         grp,
		gs:          gStore,
		services:    services,
		requestFunc: requestFunc,
		receiveFunc: receiveFunc,
	}

	// Register listener for incoming e2e group chat requests
	e2e.RegisterListener(&id.ZeroUser, catalog.GroupCreationRequest, &requestListener{m: m})

	// Register notifications listener for incoming e2e group chat requests
	err = e2e.AddService(catalog.GroupRq, nil)
	if err != nil {
		return nil, err
	}

	// Register all groups
	for _, gId := range m.GetGroups() {
		g, exists := m.GetGroup(gId)
		if !exists {
			jww.WARN.Printf("Unexpected failure to locate GroupID %s", gId.String())
			continue
		}

		err = m.JoinGroup(g)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

// JoinGroup adds the group to the list of group chats the user is a part of.
// An error is returned if the user is already part of the group or if the
// maximum number of groups have already been joined.
func (m Manager) JoinGroup(g gs.Group) error {
	if err := m.gs.Add(g); err != nil {
		return errors.Errorf(joinGroupErr, g.ID, err)
	}

	newService := message.Service{
		Identifier: g.ID[:],
		Tag:        catalog.Group,
		Metadata:   g.ID[:],
	}
	m.services.AddService(m.receptionId, newService, &receptionProcessor{m: &m, g: g})

	jww.DEBUG.Printf("Joined group %q with ID %s.", g.Name, g.ID)
	return nil
}

// LeaveGroup removes a group from a list of groups the user is a part of.
func (m Manager) LeaveGroup(groupID *id.ID) error {
	if err := m.gs.Remove(groupID); err != nil {
		return errors.Errorf(leaveGroupErr, groupID, err)
	}

	delService := message.Service{
		Identifier: groupID.Bytes(),
		Tag:        catalog.Group,
	}
	m.services.DeleteService(m.receptionId, delService, nil)

	jww.DEBUG.Printf("Left group with ID %s.", groupID)
	return nil
}

// GetGroups returns a list of all registered groupChat IDs.
func (m Manager) GetGroups() []*id.ID {
	jww.DEBUG.Print("Getting list of all groups.")
	return m.gs.GroupIDs()
}

// GetGroup returns the group with the matching ID or returns false if none
// exist.
func (m Manager) GetGroup(groupID *id.ID) (gs.Group, bool) {
	jww.DEBUG.Printf("Getting group with ID %s.", groupID)
	return m.gs.Get(groupID)
}

// NumGroups returns the number of groups the user is a part of.
func (m Manager) NumGroups() int {
	return m.gs.Len()
}
