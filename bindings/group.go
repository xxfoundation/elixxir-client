////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	gc "gitlab.com/elixxir/client/v4/groupChat"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

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
//   - e2eID - e2e object ID in the tracker.
//   - requestFunc - a callback to handle group chat requests.
//   - processor - the group chat message processor.
func NewGroupChat(e2eID int,
	requestFunc GroupRequest, processor GroupChatProcessor) (*GroupChat, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Construct a wrapper for the request callback
	requestCb := func(g gs.Group) {
		requestFunc.Callback(&Group{g: g})
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
//   - membershipBytes - the JSON marshalled list of []*id.ID; it contains the
//     IDs of members the user wants to add to the group.
//   - message - the initial message sent to all members in the group. This is an
//     optional parameter and may be nil.
//   - name - the name of the group decided by the creator. This is an optional
//     parameter and may be nil. If nil the group will be assigned the default
//     name.
//
// Returns:
//   - []byte - the JSON marshalled bytes of the GroupReport object, which can be
//     passed into Cmix.WaitForRoundResult to see if the group request message
//     send succeeded.
func (g *GroupChat) MakeGroup(
	membershipBytes, message, name []byte) ([]byte, error) {

	// Unmarshal membership list into a list of []*id.Id
	var members []*id.ID
	err := json.Unmarshal(membershipBytes, &members)
	if err != nil {
		return nil, err
	}

	// Construct group
	grp, roundIDs, status, err := g.m.MakeGroup(members, name, message)
	if err != nil {
		return nil, err
	}

	// Construct the group report
	report := GroupReport{
		Id:         grp.ID.Bytes(),
		RoundsList: makeRoundsList(roundIDs...),
		RoundURL:   getRoundURL(roundIDs[0]),
		Status:     int(status),
	}

	// Marshal the report
	return json.Marshal(report)
}

// ResendRequest resends a group request to all members in the group.
//
// Parameters:
//   - groupId - a byte representation of a group's ID.
//     This can be found in the report returned by GroupChat.MakeGroup.
//
// Returns:
//   - []byte - the JSON marshalled bytes of the GroupReport object, which can be
//     passed into WaitForRoundResult to see if the group request message send
//     succeeded.
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
		RoundURL:   getRoundURL(rnds[0]),
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
//   - serializedGroupData - the result of calling Group.Serialize() on
//     any Group object returned over the bindings
func (g *GroupChat) JoinGroup(serializedGroupData []byte) error {
	grp, err := DeserializeGroup(serializedGroupData)
	if err != nil {
		return err
	}
	return g.m.JoinGroup(grp.g)
}

// LeaveGroup deletes a group so a user no longer has access.
//
// Parameters:
//   - groupId - the byte data representing a group ID.
//     This can be pulled from a marshalled GroupReport.
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
//   - groupId - the byte data representing a group ID. This can be pulled from
//     marshalled GroupReport.
//   - message - the message that the user wishes to send to the group.
//   - tag - the tag associated with the message. This tag may be empty.
//
// Returns:
//   - []byte - the JSON marshalled bytes of the GroupSendReport object, which
//     can be passed into Cmix.WaitForRoundResult to see if the group message
//     send succeeded.
func (g *GroupChat) Send(groupId, message []byte, tag string) ([]byte, error) {
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
		RoundURL:   getRoundURL(round.ID),
		RoundsList: makeRoundsList(round.ID),
		Timestamp:  timestamp.UnixNano(),
		MessageID:  msgID.Bytes(),
	}

	return json.Marshal(sendReport)
}

// GetGroups returns a list of group IDs that the user is a member of.
//
// Returns:
//   - []byte - a JSON marshalled []*id.ID representing all group ID's.
func (g *GroupChat) GetGroups() ([]byte, error) {
	return json.Marshal(g.m.GetGroups())
}

// GetGroup returns the group with the group ID. If no group exists, then the
// error "failed to find group" is returned.
//
// Parameters:
//   - groupId - The byte data representing a group ID (a byte marshalled id.ID).
//     This can be pulled from a marshalled GroupReport.
//
// Returns:
//   - Group - The bindings-layer representation of a group.
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
	return &Group{g: grp}, nil
}

// NumGroups returns the number of groups the user is a part of.
func (g *GroupChat) NumGroups() int {
	return g.m.NumGroups()
}

////////////////////////////////////////////////////////////////////////////////
// Group Structure                                                            //
////////////////////////////////////////////////////////////////////////////////

// Group structure contains the identifying and membership information of a
// group chat.
type Group struct {
	g gs.Group
}

// GetName returns the name set by the user for the group.
func (g *Group) GetName() []byte {
	return g.g.Name
}

// GetID return the 33-byte unique group ID. This represents the id.ID object.
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

// GetMembership retrieves a list of group members. The list is in order;
// the first contact is the leader/creator of the group.
// All subsequent members are ordered by their ID.
//
// Returns:
//   - []byte - JSON marshalled [group.Membership], which is an array of
//     [group.Member].
//
// Example JSON [group.Membership] return:
//
//	[
//	  {
//	    "ID": "U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
//	    "DhKey": {
//	      "Value": 3534334367214237261,
//	      "Fingerprint": 16801541511233098363
//	    }
//	  },
//	  {
//	    "ID": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//	    "DhKey": {
//	      "Value": 7497468244883513247,
//	      "Fingerprint": 16801541511233098363
//	    }
//	  }
//	]
func (g *Group) GetMembership() ([]byte, error) {
	return json.Marshal(g.g.Members)
}

// Serialize serializes the Group.
func (g *Group) Serialize() []byte {
	return g.g.Serialize()
}

// DeserializeGroup converts the results of Group.Serialize into a Group
// so that its methods can be called.
func DeserializeGroup(serializedGroupData []byte) (*Group, error) {
	grp, err := gs.DeserializeGroup(serializedGroupData)
	if err != nil {
		return nil, err
	}
	return &Group{g: grp}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Callbacks                                                                  //
////////////////////////////////////////////////////////////////////////////////

// GroupRequest is a bindings-layer interface that handles a group reception.
//
// Parameters:
//   - g - a bindings layer Group object.
type GroupRequest interface {
	Callback(g *Group)
}

// GroupChatProcessor manages the handling of received group chat messages.
// The decryptedMessage field will be a JSON marshalled GroupChatMessage.
type GroupChatProcessor interface {
	Process(decryptedMessage, msg, receptionId []byte, ephemeralId,
		roundId int64, roundUrl string, err error)
	fmt.Stringer
}

// groupChatProcessor implements GroupChatProcessor as a way of obtaining a
// groupChat.Processor over the bindings.
type groupChatProcessor struct {
	bindingsCb GroupChatProcessor
}

// GroupChatMessage is the bindings layer representation of the
// [groupChat.MessageReceive].
//
// GroupChatMessage Example JSON:
//
//	{
//	  "GroupId": "AAAAAAAJlasAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE",
//	  "SenderId": "AAAAAAAAB8gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//	  "MessageId": "Zm9ydHkgZml2ZQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
//	  "Payload": "Zm9ydHkgZml2ZQ==",
//	  "Timestamp": 1663009269474079000
//	}
type GroupChatMessage struct {
	// GroupId is the ID of the group that this message was sent on.
	GroupId []byte

	// SenderId is the ID of the sender of this message.
	SenderId []byte

	// MessageId is the ID of this group message.
	MessageId []byte

	// Payload is the content of the message.
	Payload []byte

	// Timestamp is the time this message was sent on.
	Timestamp int64
}

// convertMessageReceive is a helper function which converts a
// [groupChat.MessageReceive] to the bindings-layer representation GroupChatMessage.
func convertMessageReceive(decryptedMsg gc.MessageReceive) GroupChatMessage {
	return GroupChatMessage{
		GroupId:   decryptedMsg.GroupID.Bytes(),
		SenderId:  decryptedMsg.SenderID.Bytes(),
		MessageId: decryptedMsg.ID.Bytes(),
		Payload:   decryptedMsg.Payload,
		Timestamp: decryptedMsg.Timestamp.UnixNano(),
	}
}

// convertProcessor turns the input of a groupChat.Processor to the
// binding-layer primitives equivalents within the GroupChatProcessor.Process.
func convertGroupChatProcessor(decryptedMsg gc.MessageReceive, msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) (
	decryptedMessage, message, receptionId []byte, ephemeralId, roundId int64, roundUrl string, err error) {

	decryptedMessage, err = json.Marshal(convertMessageReceive(decryptedMsg))
	message = msg.Marshal()
	receptionId = receptionID.Source.Marshal()
	ephemeralId = receptionID.EphId.Int64()
	roundId = int64(round.ID)
	roundUrl = getRoundURL(round.ID)
	return
}

// Process handles incoming group chat messages.
func (gcp *groupChatProcessor) Process(decryptedMsg gc.MessageReceive, msg format.Message,
	_ []string, _ []byte, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
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
//
// Example GroupReport JSON:
//
//	{
//		"Id": "AAAAAAAAAM0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE",
//		"Rounds": [25, 64],
//		"RoundURL": "https://dashboard.xx.network/rounds/25?xxmessenger=true",
//		"Status": 1
//	}
type GroupReport struct {
	Id []byte
	RoundsList
	RoundURL string
	Status   int
}

// GroupSendReport is returned when sending a group message. It contains the
// round ID sent on and the timestamp of the send operation.
//
// Example GroupSendReport JSON:
//
//	     {
//	 	"Rounds": [25,	64],
//	 	"RoundURL": "https://dashboard.xx.network/rounds/25?xxmessenger=true",
//	 	"Timestamp": 1662577352813112000,
//	 	"MessageID": "69ug6FA50UT2q6MWH3hne9PkHQ+H9DnEDsBhc0m0Aww="
//		    }
type GroupSendReport struct {
	RoundsList
	RoundURL  string
	Timestamp int64
	MessageID []byte
}
