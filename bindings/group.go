///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	gc "gitlab.com/elixxir/client/groupChat"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Group Singleton Tracker                                               //
////////////////////////////////////////////////////////////////////////////////

// groupTrackerSingleton is used to track Group objects so that they can be
// referenced by ID back over the bindings.
var groupTrackerSingleton = &groupTracker{
	tracked: make(map[int]*Group),
	count:   0,
}

// groupTracker is a singleton used to keep track of extant Group objects,
// preventing race conditions created by passing it over the bindings.
type groupTracker struct {
	tracked map[int]*Group
	count   int
	mux     sync.RWMutex
}

// make create a Group from a groupStore.Group, assigns it a unique ID, and
// adds it to the groupChatTracker.
func (ut *groupTracker) make(g gs.Group) *Group {
	ut.mux.Lock()
	defer ut.mux.Unlock()

	id := ut.count
	ut.count++

	ut.tracked[id] = &Group{
		g:  g,
		id: id,
	}

	return ut.tracked[id]
}

// get a Group from the groupChatTracker given its ID.
func (ut *groupTracker) get(id int) (*Group, error) {
	ut.mux.RLock()
	defer ut.mux.RUnlock()

	g, exist := ut.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get Group for ID %d, does not exist", id)
	}

	return g, nil
}

// delete removes a Group from the groupTracker.
func (ut *groupTracker) delete(id int) {
	ut.mux.Lock()
	defer ut.mux.Unlock()

	delete(ut.tracked, id)
}

////////////////////////////////////////////////////////////////////////////////
// Group Chat                                                                 //
////////////////////////////////////////////////////////////////////////////////

// GroupChat is a binding-layer group chat manager.
type GroupChat struct {
	m *gc.Wrapper
}

// NewGroupChat creates a bindings-layer group chat manager.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker.
//  - requestFunc - a callback to handle group chat requests.
//  - processor - the group chat message processor.
func NewGroupChat(e2eID int,
	requestFunc GroupRequest, processor GroupChatProcessor) (*GroupChat, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Construct a wrapper for the request callback
	requestCb := func(g gs.Group) {
		newGroup := groupTrackerSingleton.make(g)
		requestFunc.Callback(newGroup)
	}

	// Construct a group chat manager
	gcInt, err := gc.NewManager(user.api, requestCb,
		&groupChatProcessor{bindingsCb: processor})
	if err != nil {
		return nil, err
	}

	// Construct wrapper
	wrapper := gc.NewWrapper(gcInt)
	return &GroupChat{m: wrapper}, nil
}

// MakeGroup creates a new Group and sends a group request to all members in the
// group.
//
// Parameters:
//  - membership - IDs of members the user wants in the group.
//  - message - the initial message sent to all members in the group. This is an
//    optional parameter and may be nil.
//  - tag - the name of the group decided by the creator. This is an optional parameter
//    and may be nil. If nil the group will be assigned the default name.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the GroupReport object, which can be
//    passed into WaitForRoundResult to see if the group request message send
//    succeeded.
func (g *GroupChat) MakeGroup(membership IdList, message, name []byte) (
	[]byte, error) {

	// Construct membership list into a list of []*id.Id
	members, err := deserializeIdList(membership)
	if err != nil {
		return nil, err
	}

	// Construct group
	grp, rounds, status, err := g.m.MakeGroup(members, name, message)
	if err != nil {
		return nil, err
	}

	// Construct the group report
	report := GroupReport{
		Id:         grp.ID.Bytes(),
		RoundsList: makeRoundsList(rounds...),
		Status:     int(status),
	}

	// Marshal the report
	return json.Marshal(report)
}

// ResendRequest resends a group request to all members in the group.
//
// Parameters:
//  - groupId - a byte representation of a group's ID.
//    This can be found in the report returned by GroupChat.MakeGroup.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the GroupReport object, which can be
//    passed into WaitForRoundResult to see if the group request message send
//    succeeded.
func (g *GroupChat) ResendRequest(groupId []byte) ([]byte, error) {

	// Unmarshal the group ID
	groupID, err := id.Unmarshal(groupId)
	if err != nil {
		return nil,
			errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	// Retrieve group from manager
	grp, exists := g.m.GetGroup(groupID)
	if !exists {
		return nil, errors.Errorf("Failed to find group %s", groupID)
	}

	// Resend request
	rnds, status, err := g.m.ResendRequest(groupID)
	if err != nil {
		return nil, err
	}

	// Construct the group report on resent request
	report := &GroupReport{
		Id:         grp.ID.Bytes(),
		RoundsList: makeRoundsList(rnds...),
		Status:     int(status),
	}

	// Marshal the report
	return json.Marshal(report)
}

// JoinGroup allows a user to join a group when a request is received.
// If an error is returned, handle it properly first; you may then retry later
// with the same trackedGroupId.
//
// Parameters:
//  - trackedGroupId - the ID to retrieve the Group object within the backend's
//    tracking system. This is received by GroupRequest.Callback.
func (g *GroupChat) JoinGroup(trackedGroupId int) error {
	// Retrieve group from singleton
	grp, err := groupTrackerSingleton.get(trackedGroupId)
	if err != nil {
		return err
	}

	// Join group
	err = g.m.JoinGroup(grp.g)
	if err != nil {
		return err
	}

	return nil
}

// LeaveGroup deletes a group so a user no longer has access.
//
// Parameters:
//  - groupId - the byte data representing a group ID.
//    This can be pulled from a marshalled GroupReport.
func (g *GroupChat) LeaveGroup(groupId []byte) error {
	grpId, err := id.Unmarshal(groupId)
	if err != nil {
		return errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	return g.m.LeaveGroup(grpId)
}

// Send is the bindings-level function for sending to a group.
//
// Parameters:
//  - groupId - the byte data representing a group ID.
//    This can be pulled from a marshalled GroupReport.
//  - message - the message that the user wishes to send to the group.
//  - tag - the tag associated with the message. This tag may be empty.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the GroupSendReport object, which
//    can be passed into WaitForRoundResult to see if the group message send
//    succeeded.
func (g *GroupChat) Send(groupId,
	message []byte, tag string) ([]byte, error) {
	groupID, err := id.Unmarshal(groupId)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	// Send group message
	round, timestamp, msgID, err := g.m.Send(groupID, message, tag)
	if err != nil {
		return nil, err
	}

	// Construct send report
	sendReport := &GroupSendReport{
		RoundsList: makeRoundsList(round),
		Timestamp:  timestamp.UnixNano(),
		MessageID:  msgID.Bytes(),
	}

	return json.Marshal(sendReport)
}

// GetGroups returns an IdList containing a list of group IDs that the user is a member of.
//
// Returns:
//  - []byte - a JSON marshalled IdList representing all group ID's.
func (g *GroupChat) GetGroups() ([]byte, error) {
	groupIds := makeIdList(g.m.GetGroups())

	return json.Marshal(groupIds)
}

// GetGroup returns the group with the group ID. If no group exists, then the
// error "failed to find group" is returned.
//
// Parameters:
//  - groupId - The byte data representing a group ID (a byte marshalled id.ID).
//              This can be pulled from a marshalled GroupReport.
// Returns:
//  - Group - The bindings-layer representation of a group.
func (g *GroupChat) GetGroup(groupId []byte) (*Group, error) {
	// Unmarshal group ID
	groupID, err := id.Unmarshal(groupId)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	// Retrieve group from manager
	grp, exists := g.m.GetGroup(groupID)
	if !exists {
		return nil, errors.New("failed to find group")
	}

	// Add to tracker and return Group object
	return groupTrackerSingleton.make(grp), nil
}

// NumGroups returns the number of groups the user is a part of.
func (g *GroupChat) NumGroups() int {
	return g.m.NumGroups()
}

////////////////////////////////////////////////////////////////////////////////
// Group Structure
////////////////////////////////////////////////////////////////////////////////

// Group structure contains the identifying and membership information of a
// group chat.
type Group struct {
	g  gs.Group
	id int
}

// GetName returns the name set by the user for the group.
func (g *Group) GetName() []byte {
	return g.g.Name
}

// GetID return the 33-byte unique group ID. This represents the id.ID object
func (g *Group) GetID() []byte {
	return g.g.ID.Bytes()
}

// GetTrackedID returns the tracked ID of the Group object. This is used by the backend tracker.
func (g *Group) GetTrackedID() int {
	return g.id
}

// GetInitMessage returns initial message sent with the group request.
func (g *Group) GetInitMessage() []byte {
	return g.g.InitMessage
}

// GetCreatedNano returns the time the group was created in nanoseconds. This is
// also the time the group requests were sent.
func (g *Group) GetCreatedNano() int64 {
	return g.g.Created.UnixNano()
}

// GetCreatedMS returns the time the group was created in milliseconds. This is
// also the time the group requests were sent.
func (g *Group) GetCreatedMS() int64 {
	ts := uint64(g.g.Created.UnixNano()) / uint64(time.Millisecond)
	return int64(ts)
}

// GetMembership retrieves a list of group members. The list is in order;
// the first contact is the leader/creator of the group.
// All subsequent members are ordered by their ID.
//
// Returns:
//  - []byte - a JSON marshalled version of the member list.
func (g *Group) GetMembership() ([]byte, error) {
	return json.Marshal(g.g.Members)
}

// Serialize serializes the Group.
func (g *Group) Serialize() []byte {
	return g.g.Serialize()
}

//////////////////////////////////////////////////////////////////////////////////
// Callbacks
//////////////////////////////////////////////////////////////////////////////////

// GroupRequest is a bindings-layer interface that handles a group reception.
//
// Parameters:
//  - trackedGroupId - a bindings layer Group object.
type GroupRequest interface {
	Callback(g *Group)
}

// GroupChatProcessor manages the handling of received group chat messages.
type GroupChatProcessor interface {
	Process(decryptedMessage, msg, receptionId []byte, ephemeralId,
		roundId int64, err error)
	fmt.Stringer
}

// groupChatProcessor implements GroupChatProcessor as a way of obtaining a
// groupChat.Processor over the bindings.
type groupChatProcessor struct {
	bindingsCb GroupChatProcessor
}

// convertProcessor turns the input of a groupChat.Processor to the
// binding-layer primitives equivalents within the GroupChatProcessor.Process.
func convertGroupChatProcessor(decryptedMsg gc.MessageReceive, msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) (
	decryptedMessage, message, receptionId []byte, ephemeralId, roundId int64, err error) {

	decryptedMessage, err = json.Marshal(decryptedMsg)
	message = msg.Marshal()
	receptionId = receptionID.Source.Marshal()
	ephemeralId = receptionID.EphId.Int64()
	roundId = int64(round.ID)
	return
}

// Process handles incoming group chat messages.
func (gcp *groupChatProcessor) Process(decryptedMsg gc.MessageReceive, msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	gcp.bindingsCb.Process(convertGroupChatProcessor(decryptedMsg, msg, receptionID, round))
}

// String prints a name for debugging.
func (gcp *groupChatProcessor) String() string {
	return gcp.bindingsCb.String()
}

/////////////////////////////////////////////////////////////////////////////////
// Report Structures
////////////////////////////////////////////////////////////////////////////////

// GroupReport is returned when creating a new group and contains the ID of
// the group, a list of rounds that the group requests were sent on, and the
// status of the send operation.
type GroupReport struct {
	Id []byte
	RoundsList
	Status int
}

// GroupSendReport is returned when sending a group message. It contains the
// round ID sent on and the timestamp of the send operation.
type GroupSendReport struct {
	RoundsList
	Timestamp int64
	MessageID []byte
}
