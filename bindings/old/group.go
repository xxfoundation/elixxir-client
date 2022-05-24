///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	gc "gitlab.com/elixxir/client/groupChat"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// GroupChat object contains the group chat manager.
type GroupChat struct {
	m *gc.Manager
}

// GroupRequestFunc contains a function callback that is called when a group
// request is received.
type GroupRequestFunc interface {
	GroupRequestCallback(g *Group)
}

// GroupReceiveFunc contains a function callback that is called when a group
// message is received.
type GroupReceiveFunc interface {
	GroupReceiveCallback(msg *GroupMessageReceive)
}

// NewGroupManager creates a new group chat manager.
func NewGroupManager(client *Client, requestFunc GroupRequestFunc,
	receiveFunc GroupReceiveFunc) (*GroupChat, error) {

	requestCallback := func(g gs.Group) {
		requestFunc.GroupRequestCallback(&Group{g})
	}
	receiveCallback := func(msg gc.MessageReceive) {
		receiveFunc.GroupReceiveCallback(&GroupMessageReceive{msg})
	}

	// Create a new group chat manager
	// TODO: Need things from storage, services, etc?
	m, err := gc.NewManager(&client.api, requestCallback, receiveCallback)
	if err != nil {
		return nil, err
	}

	return &GroupChat{m}, nil
}

// MakeGroup creates a new group and sends a group request to all members in the
// group. The ID of the new group, the rounds the requests were sent on, and the
// status of the sends are contained in NewGroupReport.
func (g *GroupChat) MakeGroup(membership *IdList, name, message []byte) *NewGroupReport {
	grp, rounds, status, err := g.m.MakeGroup(membership.list, name, message)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return &NewGroupReport{&Group{grp}, rounds, status, errStr}
}

// ResendRequest resends a group request to all members in the group. The rounds
// they were sent on and the status of the send are contained in NewGroupReport.
func (g *GroupChat) ResendRequest(groupIdBytes []byte) (*NewGroupReport, error) {
	groupID, err := id.Unmarshal(groupIdBytes)
	if err != nil {
		return nil,
			errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	grp, exists := g.m.GetGroup(groupID)
	if !exists {
		return nil, errors.Errorf("Failed to find group %s", groupID)
	}

	rounds, status, err := g.m.ResendRequest(groupID)

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return &NewGroupReport{&Group{grp}, rounds, status, errStr}, nil
}

// JoinGroup allows a user to join a group when they receive a request. The
// caller must pass in the serialized bytes of a Group.
func (g *GroupChat) JoinGroup(serializedGroupData []byte) error {
	grp, err := gs.DeserializeGroup(serializedGroupData)
	if err != nil {
		return err
	}
	return g.m.JoinGroup(grp)
}

// LeaveGroup deletes a group so a user no longer has access.
func (g *GroupChat) LeaveGroup(groupIdBytes []byte) error {
	groupID, err := id.Unmarshal(groupIdBytes)
	if err != nil {
		return errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	return g.m.LeaveGroup(groupID)
}

// Send sends the message to the specified group. Returns the round the messages
// were sent on.
func (g *GroupChat) Send(groupIdBytes, message []byte) (*GroupSendReport, error) {
	groupID, err := id.Unmarshal(groupIdBytes)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	round, timestamp, msgID, err := g.m.Send(groupID, message)
	return &GroupSendReport{round, timestamp, msgID}, err
}

// GetGroups returns an IdList containing a list of group IDs that the user is a
// part of.
func (g *GroupChat) GetGroups() *IdList {
	return &IdList{g.m.GetGroups()}
}

// GetGroup returns the group with the group ID. If no group exists, then the
// error "failed to find group" is returned.
func (g *GroupChat) GetGroup(groupIdBytes []byte) (*Group, error) {
	groupID, err := id.Unmarshal(groupIdBytes)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal group ID: %+v", err)
	}

	grp, exists := g.m.GetGroup(groupID)
	if !exists {
		return nil, errors.New("failed to find group")
	}

	return &Group{grp}, nil
}

// NumGroups returns the number of groups the user is a part of.
func (g *GroupChat) NumGroups() int {
	return g.m.NumGroups()
}

////
// NewGroupReport Structure
////

// NewGroupReport is returned when creating a new group and contains the ID of
// the group, a list of rounds that the group requests were sent on, and the
// status of the send.
type NewGroupReport struct {
	group  *Group
	rounds []id.Round
	status gc.RequestStatus
	err    string
}

type GroupReportDisk struct {
	List   []id.Round
	GrpId  []byte
	Status int
}

// GetGroup returns the Group.
func (ngr *NewGroupReport) GetGroup() *Group {
	return ngr.group
}

// GetRoundList returns the RoundList containing a list of rounds requests were
// sent on.
func (ngr *NewGroupReport) GetRoundList() *RoundList {
	return &RoundList{ngr.rounds}
}

// GetStatus returns the status of the requests sent when creating a new group.
// status = 0   an error occurred before any requests could be sent
//          1   all requests failed to send (call Resend Group)
//          2   some request failed and some succeeded (call Resend Group)
//          3,  all requests sent successfully (call Resend Group)
func (ngr *NewGroupReport) GetStatus() int {
	return int(ngr.status)
}

// GetError returns the string of an error.
// Will be an empty string if no error occured
func (ngr *NewGroupReport) GetError() string {
	return ngr.err
}

func (ngr *NewGroupReport) Marshal() ([]byte, error) {
	grpReportDisk := GroupReportDisk{
		List:   ngr.rounds,
		GrpId:  ngr.group.GetID()[:],
		Status: ngr.GetStatus(),
	}
	return json.Marshal(&grpReportDisk)
}

func (ngr *NewGroupReport) Unmarshal(b []byte) error {
	grpReportDisk := GroupReportDisk{}
	if err := json.Unmarshal(b, &grpReportDisk); err != nil {
		return errors.New(fmt.Sprintf("Failed to unmarshal group "+
			"report: %s", err.Error()))
	}

	grpId, err := id.Unmarshal(grpReportDisk.GrpId)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to unmarshal group "+
			"id: %s", err.Error()))
	}

	ngr.group.g.ID = grpId
	ngr.rounds = grpReportDisk.List
	ngr.status = gc.RequestStatus(grpReportDisk.Status)

	return nil
}

////
// NewGroupReport Structure
////

// GroupSendReport is returned when sending a group message. It contains the
// round ID sent on and the timestamp of the send.
type GroupSendReport struct {
	roundID   id.Round
	timestamp time.Time
	messageID group.MessageID
}

// GetRoundID returns the ID of the round that the send occurred on.
func (gsr *GroupSendReport) GetRoundID() int64 {
	return int64(gsr.roundID)
}

// GetTimestampNano returns the timestamp of the send in nanoseconds.
func (gsr *GroupSendReport) GetTimestampNano() int64 {
	return gsr.timestamp.UnixNano()
}

// GetTimestampMS returns the timestamp of the send in milliseconds.
func (gsr *GroupSendReport) GetTimestampMS() int64 {
	ts := uint64(gsr.timestamp.UnixNano()) / uint64(time.Millisecond)
	return int64(ts)
}

// GetMessageID returns the ID of the round that the send occurred on.
func (gsr *GroupSendReport) GetMessageID() []byte {
	return gsr.messageID[:]
}

// GetRoundURL returns the URL of the round that the send occurred on.
func (gsr *GroupSendReport) GetRoundURL() string {
	return getRoundURL(gsr.roundID)
}

////
// Group Structure
////

// Group structure contains the identifying and membership information of a
// group chat.
type Group struct {
	g gs.Group
}

// GetName returns the name set by the user for the group.
func (g *Group) GetName() []byte {
	return g.g.Name
}

// GetID return the 33-byte unique group ID.
func (g *Group) GetID() []byte {
	return g.g.ID.Bytes()
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

// GetMembership returns a list of contacts, one for each member in the group.
// The list is in order; the first contact is the leader/creator of the group.
// All subsequent members are ordered by their ID.
func (g *Group) GetMembership() *GroupMembership {
	return &GroupMembership{g.g.Members}
}

// Serialize serializes the Group.
func (g *Group) Serialize() []byte {
	return g.g.Serialize()
}

////
// Membership Structure
////

// GroupMembership structure contains a list of members that are part of a
// group. The first member is the group leader.
type GroupMembership struct {
	m group.Membership
}

// Len returns the number of members in the group membership.
func (gm *GroupMembership) Len() int {
	return len(gm.m)
}

// Get returns the member at the index. The member at index 0 is always the
// group leader. An error is returned if the index is out of range.
func (gm *GroupMembership) Get(i int) (*GroupMember, error) {
	if i < 0 || i >= gm.Len() {
		return nil, errors.Errorf("ID list index must be between %d "+
			"and the last element %d.", 0, gm.Len())
	}
	return &GroupMember{gm.m[i]}, nil
}

////
// Member Structure
////

// GroupMember represents a member in the group membership list.
type GroupMember struct {
	group.Member
}

// GetID returns the 33-byte user ID of the member.
func (gm GroupMember) GetID() []byte {
	return gm.ID.Bytes()
}

// GetDhKey returns the byte representation of the public Diffie–Hellman key of
// the member.
func (gm *GroupMember) GetDhKey() []byte {
	return gm.DhKey.Bytes()
}

////
// Message Receive Structure
////

// GroupMessageReceive contains a group message, its ID, and its data that a
// user receives.
type GroupMessageReceive struct {
	gc.MessageReceive
}

// GetGroupID returns the 33-byte group ID.
func (gmr *GroupMessageReceive) GetGroupID() []byte {
	return gmr.GroupID.Bytes()
}

// GetMessageID returns the message ID.
func (gmr *GroupMessageReceive) GetMessageID() []byte {
	return gmr.ID.Bytes()
}

// GetPayload returns the message payload.
func (gmr *GroupMessageReceive) GetPayload() []byte {
	return gmr.Payload
}

// GetSenderID returns the 33-byte user ID of the sender.
func (gmr *GroupMessageReceive) GetSenderID() []byte {
	return gmr.SenderID.Bytes()
}

// GetRecipientID returns the 33-byte user ID of the recipient.
func (gmr *GroupMessageReceive) GetRecipientID() []byte {
	return gmr.RecipientID.Bytes()
}

// GetEphemeralID returns the address ID of the recipient.
func (gmr *GroupMessageReceive) GetEphemeralID() int64 {
	return gmr.EphemeralID.Int64()
}

// GetTimestampNano returns the message timestamp in nanoseconds.
func (gmr *GroupMessageReceive) GetTimestampNano() int64 {
	return gmr.Timestamp.UnixNano()
}

// GetTimestampMS returns the message timestamp in milliseconds.
func (gmr *GroupMessageReceive) GetTimestampMS() int64 {
	ts := uint64(gmr.Timestamp.UnixNano()) / uint64(time.Millisecond)
	return int64(ts)
}

// GetRoundID returns the ID of the round the message was sent on.
func (gmr *GroupMessageReceive) GetRoundID() int64 {
	return int64(gmr.RoundID)
}

// GetRoundURL returns the ID of the round the message was sent on.
func (gmr *GroupMessageReceive) GetRoundURL() string {
	return getRoundURL(gmr.RoundID)
}

// GetRoundTimestampNano returns the timestamp, in nanoseconds, of the round the
// message was sent on.
func (gmr *GroupMessageReceive) GetRoundTimestampNano() int64 {
	return gmr.RoundTimestamp.UnixNano()
}

// GetRoundTimestampMS returns the timestamp, in milliseconds, of the round the
// message was sent on.
func (gmr *GroupMessageReceive) GetRoundTimestampMS() int64 {
	ts := uint64(gmr.RoundTimestamp.UnixNano()) / uint64(time.Millisecond)
	return int64(ts)
}
