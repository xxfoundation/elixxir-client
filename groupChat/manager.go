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
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// Error messages.
const (
	// NewManager
	newGroupStoreErr     = "failed to create new group store: %+v"
	errAddDefaultService = "could not add default service: %+v"

	// manager.JoinGroup
	joinGroupErr = "failed to join new group %s: %+v"

	// manager.LeaveGroup
	leaveGroupErr = "failed to leave group %s: %+v"
)

const defaultServiceTag = "default"

// manager handles the list of groups a user is a part of.
type manager struct {
	// Group storage
	gs *gs.Store

	// List of registered processors
	services    map[string]Processor
	servicesMux sync.Mutex

	// Callback that is called when a new group request is received
	requestFunc RequestCallback

	user groupE2e
}

// NewManager creates a new group chat manager
func NewManager(user groupE2e,
	requestFunc RequestCallback, receiveFunc Processor) (GroupChat, error) {

	// Initialize a member object
	handler := user.GetE2E()
	member := group.Member{
		ID:    user.GetReceptionIdentity().ID,
		DhKey: handler.GetHistoricalDHPubkey(),
	}

	// Load the group chat storage or create one if one does not exist
	kv := user.GetStorage().GetKV()
	gStore, err := gs.NewOrLoadStore(kv, member)
	if err != nil {
		return nil, errors.Errorf(newGroupStoreErr, err)
	}

	// Define the manager object
	m := &manager{
		gs:          gStore,
		services:    make(map[string]Processor),
		requestFunc: requestFunc,
		user:        user,
	}

	// Register listener for incoming e2e group chat requests
	handler.RegisterListener(
		&id.ZeroUser, catalog.GroupCreationRequest, &requestListener{m})

	// Register notifications listener for incoming e2e group chat requests
	err = handler.AddService(catalog.GroupRq, nil)
	if err != nil {
		return nil, err
	}

	err = m.AddService(defaultServiceTag, receiveFunc)
	if err != nil {
		return nil, errors.Errorf(errAddDefaultService, err)
	}

	return m, nil
}

// JoinGroup adds the group to storage, and enables requisite services.
// An error is returned if the user is already part of the group or if the
// maximum number of groups have already been joined.
func (m *manager) JoinGroup(g gs.Group) error {
	if err := m.gs.Add(g); err != nil {
		return errors.Errorf(joinGroupErr, g.ID, err)
	}

	// Add all services for this group
	m.addAllServices(g)

	jww.INFO.Printf("[GC] Joined group %q with ID %s.", g.Name, g.ID)
	return nil
}

// LeaveGroup removes a group from a list of groups the user is a part of.
func (m *manager) LeaveGroup(groupID *id.ID) error {
	if err := m.gs.Remove(groupID); err != nil {
		return errors.Errorf(leaveGroupErr, groupID, err)
	}

	m.deleteAllServices(groupID)

	jww.INFO.Printf("[GC] Left group with ID %s.", groupID)
	return nil
}

// GetGroups returns a list of all registered groupChat IDs.
func (m *manager) GetGroups() []*id.ID {
	jww.DEBUG.Print("[GC] Getting list of all groups.")
	return m.gs.GroupIDs()
}

// GetGroup returns the group with the matching ID or returns false if none
// exist.
func (m *manager) GetGroup(groupID *id.ID) (gs.Group, bool) {
	jww.DEBUG.Printf("[GC] Getting group with ID %s.", groupID)
	return m.gs.Get(groupID)
}

// NumGroups returns the number of groups the user is a part of.
func (m *manager) NumGroups() int {
	return m.gs.Len()
}

/////////////////////////////////////////////////////////////////////////////////////////
// Internal getters /////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////

func (m *manager) getCMix() groupCmix {
	return m.user.GetCmix()
}

func (m *manager) getE2eHandler() groupE2eHandler {
	return m.user.GetE2E()
}

func (m *manager) getReceptionIdentity() xxdk.ReceptionIdentity {
	return m.user.GetReceptionIdentity()
}

func (m *manager) getRng() *fastRNG.StreamGenerator {
	return m.user.GetRng()
}

func (m *manager) getE2eGroup() *cyclic.Group {
	return m.user.GetStorage().GetE2EGroup()
}
