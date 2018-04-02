////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import "errors"

type AccessControl interface {
	CanReceive() bool
	CanSend() bool
	CanSetTopic() bool
	CanAddUser() bool
	CanRemoveUser() bool
	CanSetPrivs() bool
}

type PrivilegeSetter interface {
	SetCanSend(bool)
	SetCanReceive(bool)
}

// Users who are added to the channel have UserAccess
type UserAccess struct {
	canSend    bool
	canReceive bool
}

// make sure that UserAccess implements AccessControl
//var _ AccessControl = (UserAccess)(nil)

func (u UserAccess) CanReceive() bool    { return u.canReceive }
func (u UserAccess) CanSend() bool       { return u.canSend }
func (u UserAccess) CanSetTopic() bool   { return false }
func (u UserAccess) CanRemoveUser() bool { return false }
func (u UserAccess) CanAddUser() bool    { return false }
func (u UserAccess) CanSetPrivs() bool   { return false }

// The person who creates the channel has OwnerAccess
type OwnerAccess struct{}

// make sure that OwnerAccess implements AccessControl
//var _ AccessControl = (OwnerAccess)(nil)

func (o OwnerAccess) CanReceive() bool    { return true }
func (o OwnerAccess) CanSend() bool       { return true }
func (o OwnerAccess) CanSetTopic() bool   { return true }
func (o OwnerAccess) CanRemoveUser() bool { return true }
func (o OwnerAccess) CanAddUser() bool    { return true }
func (o OwnerAccess) CanSetPrivs() bool   { return true }

var users map[uint64]AccessControl = map[uint64]AccessControl{
	1: &OwnerAccess{},
	2: &OwnerAccess{},
	3: &OwnerAccess{},
	4: &OwnerAccess{},
	5: &UserAccess{true, true},
	6: &UserAccess{true, true},
	7: &UserAccess{true, true},
	8: &UserAccess{true, true},
	9: &UserAccess{true, true},
}

func AddUser(newUser uint64, commandSender uint64) error {
	if users[commandSender].CanAddUser() {
		users[newUser] = UserAccess{true, true}
		return nil
	} else {
		return errors.New("Couldn't add user to channel: Access denied")
	}
}

func RemoveUser(user, commandSender uint64) error {
	if users[commandSender].CanRemoveUser() {
		delete(users, user)
		return nil
	} else {
		return errors.New("Couldn't remove user from channel: Access denied")
	}
}

func SetCanSend(user, commandSender uint64, newCanSend bool) error {
	if users[commandSender].CanSetPrivs() {
		ps, ok := users[user].(PrivilegeSetter)
		if ok {
			ps.SetCanSend(newCanSend)
			return nil
		} else {
			return errors.New("Couldn't mute/unmute user: That user's" +
				" privileges can't be set")
		}
	} else {
		return errors.New("Couldn't mute/unmute user: Access denied")
	}
}

func SetCanReceive(user, commandSender uint64, newCanReceive bool) error {
	if users[commandSender].CanSetPrivs() {
		ps, ok := users[user].(PrivilegeSetter)
		if ok {
			ps.SetCanReceive(newCanReceive)
			return nil
		} else {
			return errors.New("Couldn't deafen/undeafen user: That user's" +
				" privileges can't be set")
		}
	} else {
		return errors.New("Couldn't deafen/undeafen user: Access denied")
	}
}
