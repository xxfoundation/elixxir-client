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
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/catalog"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
)

const (
	rawMessageBuffSize   = 100
	receiveStoppableName = "GroupChatReceive"
	receiveListenerName  = "GroupChatReceiveListener"
	requestStoppableName = "GroupChatRequest"
	requestListenerName  = "GroupChatRequestListener"
	groupStoppableName   = "GroupChat"
)

// Error messages.
const (
	newGroupStoreErr = "failed to create new group store: %+v"
	joinGroupErr     = "failed to join new group %s: %+v"
	leaveGroupErr    = "failed to leave group %s: %+v"
)

// Manager handles the list of groups a user is a part of.
type Manager struct {
	client *api.Client
	store  *storage.Session
	swb    interfaces.Switchboard
	net    interfaces.NetworkManager
	rng    *fastRNG.StreamGenerator
	gs     *gs.Store

	requestFunc RequestCallback
	receiveFunc ReceiveCallback
}

// NewManager generates a new group chat manager. This functions satisfies the
// GroupChat interface.
func NewManager(client *api.Client, requestFunc RequestCallback,
	receiveFunc ReceiveCallback) (*Manager, error) {
	return newManager(
		client,
		client.GetUser().ReceptionID.DeepCopy(),
		client.GetStorage().E2e().GetDHPublicKey(),
		client.GetStorage(),
		client.GetSwitchboard(),
		client.GetNetworkInterface(),
		client.GetRng(),
		client.GetStorage().GetKV(),
		requestFunc,
		receiveFunc,
	)
}

// newManager creates a new group chat manager from api.Client parts for easier
// testing.
func newManager(client *api.Client, userID *id.ID, userDhKey *cyclic.Int,
	store *storage.Session, swb interfaces.Switchboard,
	net interfaces.NetworkManager, rng *fastRNG.StreamGenerator,
	kv *versioned.KV, requestFunc RequestCallback,
	receiveFunc ReceiveCallback) (*Manager, error) {

	// Load the group chat storage or create one if one does not exist
	gStore, err := gs.NewOrLoadStore(
		kv, group.Member{ID: userID, DhKey: userDhKey})
	if err != nil {
		return nil, errors.Errorf(newGroupStoreErr, err)
	}

	return &Manager{
		client:      client,
		store:       store,
		swb:         swb,
		net:         net,
		rng:         rng,
		gs:          gStore,
		requestFunc: requestFunc,
		receiveFunc: receiveFunc,
	}, nil
}

// StartProcesses starts the reception worker.
func (m *Manager) StartProcesses() (stoppable.Stoppable, error) {
	// Start group reception worker
	receiveStop := stoppable.NewSingle(receiveStoppableName)
	receiveChan := make(chan message.Receive, rawMessageBuffSize)
	m.swb.RegisterChannel(receiveListenerName, &id.ID{},
		message.Raw, receiveChan)
	go m.receive(receiveChan, receiveStop)

	// Start group request worker
	requestStop := stoppable.NewSingle(requestStoppableName)
	requestChan := make(chan message.Receive, rawMessageBuffSize)
	m.swb.RegisterChannel(requestListenerName, &id.ID{},
		message.GroupCreationRequest, requestChan)
	go m.receiveRequest(requestChan, requestStop)

	// Create a multi stoppable
	multiStoppable := stoppable.NewMulti(groupStoppableName)
	multiStoppable.Add(receiveStop)
	multiStoppable.Add(requestStop)

	return multiStoppable, nil
}

// JoinGroup adds the group to the list of group chats the user is a part of.
// An error is returned if the user is already part of the group or if the
// maximum number of groups have already been joined.
func (m Manager) JoinGroup(g gs.Group) error {
	if err := m.gs.Add(g); err != nil {
		return errors.Errorf(joinGroupErr, g.ID, err)
	}

	edgeStore := m.store.GetEdge()
	edgeStore.Add(edge.Preimage{
		Data:   g.ID[:],
		Type:   catalog.Group,
		Source: g.ID[:],
	}, m.store.GetUser().ReceptionID)

	jww.DEBUG.Printf("Joined group %q with ID %s.", g.Name, g.ID)

	return nil
}

// LeaveGroup removes a group from a list of groups the user is a part of.
func (m Manager) LeaveGroup(groupID *id.ID) error {
	if err := m.gs.Remove(groupID); err != nil {
		return errors.Errorf(leaveGroupErr, groupID, err)
	}

	edgeStore := m.store.GetEdge()
	err := edgeStore.Remove(edge.Preimage{
		Data:   groupID[:],
		Type:   catalog.Group,
		Source: groupID[:],
	}, m.store.GetUser().ReceptionID)

	jww.DEBUG.Printf("Left group with ID %s.", groupID)

	return err
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
