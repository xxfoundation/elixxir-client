////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"gitlab.com/elixxir/client/cmix/rounds"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Wrapper handles the sending and receiving of group chat using E2E
// messages to inform the recipient of incoming group chat messages.
type Wrapper struct {
	gc GroupChat
}

// NewWrapper constructs a wrapper around the GroupChat interface.
func NewWrapper(manager GroupChat) *Wrapper {
	return &Wrapper{gc: manager}
}

// MakeGroup calls GroupChat.MakeGroup.
func (w *Wrapper) MakeGroup(membership []*id.ID, name, message []byte) (
	gs.Group, []id.Round, RequestStatus, error) {
	return w.gc.MakeGroup(membership, name, message)
}

// GetGroup calls GroupChat.GetGroup.
func (w *Wrapper) GetGroup(groupID *id.ID) (gs.Group, bool) {
	return w.gc.GetGroup(groupID)
}

// ResendRequest calls GroupChat.ResendRequest.
func (w *Wrapper) ResendRequest(groupID *id.ID) ([]id.Round, RequestStatus, error) {
	return w.gc.ResendRequest(groupID)
}

// JoinGroup calls GroupChat.JoinGroup.
func (w *Wrapper) JoinGroup(grp gs.Group) error {
	return w.gc.JoinGroup(grp)
}

// LeaveGroup calls GroupChat.LeaveGroup.
func (w *Wrapper) LeaveGroup(groupID *id.ID) error {
	return w.gc.LeaveGroup(groupID)
}

// Send calls GroupChat.Send.
func (w *Wrapper) Send(groupID *id.ID, message []byte, tag string) (
	rounds.Round, time.Time, group.MessageID, error) {
	return w.gc.Send(groupID, tag, message)
}

// GetGroups calls GroupChat.GetGroups.
func (w *Wrapper) GetGroups() []*id.ID {
	return w.gc.GetGroups()
}

// NumGroups calls GroupChat.NumGroups.
func (w *Wrapper) NumGroups() int {
	return w.gc.NumGroups()
}
