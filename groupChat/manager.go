///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	crypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
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

// GroupCmix is a subset of the cmix.Client interface containing only the
// methods needed by GroupChat
type GroupCmix interface {
	SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (
		id.Round, []ephemeral.Id, error)
	AddService(
		clientID *id.ID, newService message.Service, response message.Processor)
	DeleteService(
		clientID *id.ID, toDelete message.Service, processor message.Processor)
	GetMaxMessageLength() int
}

// GroupE2e is a subset of the e2e.Handler interface containing only the methods
// needed by GroupChat
type GroupE2e interface {
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params e2e.Params) ([]id.Round, crypto.MessageID, time.Time, error)
	RegisterListener(senderID *id.ID, messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
	AddService(tag string, processor message.Processor) error
	AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams session.Params) (partner.Manager, error)
	GetPartner(partnerID *id.ID) (partner.Manager, error)
	GetHistoricalDHPubkey() *cyclic.Int
	GetHistoricalDHPrivkey() *cyclic.Int
}

// manager handles the list of groups a user is a part of.
type manager struct {
	// Group storage
	gs *gs.Store

	// List of registered processors
	services    map[string]Processor
	servicesMux sync.Mutex

	// Callback that is called when a new group request is received
	requestFunc RequestCallback

	receptionId *id.ID
	net         GroupCmix
	e2e         GroupE2e
	grp         *cyclic.Group
	rng         *fastRNG.StreamGenerator
}

// NewManager creates a new group chat manager
func NewManager(services GroupCmix, e2e GroupE2e, receptionId *id.ID,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, kv *versioned.KV,
	requestFunc RequestCallback, receiveFunc Processor) (GroupChat, error) {

	// Load the group chat storage or create one if one does not exist
	gStore, err := gs.NewOrLoadStore(
		kv, group.Member{ID: receptionId, DhKey: e2e.GetHistoricalDHPubkey()})
	if err != nil {
		return nil, errors.Errorf(newGroupStoreErr, err)
	}

	// Define the manager object
	m := &manager{
		gs:          gStore,
		services:    make(map[string]Processor),
		requestFunc: requestFunc,
		receptionId: receptionId,
		net:         services,
		e2e:         e2e,
		grp:         grp,
		rng:         rng,
	}

	// Register listener for incoming e2e group chat requests
	e2e.RegisterListener(
		&id.ZeroUser, catalog.GroupCreationRequest, &requestListener{m})

	// Register notifications listener for incoming e2e group chat requests
	err = e2e.AddService(catalog.GroupRq, nil)
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
