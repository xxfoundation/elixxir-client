////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
)

type AccessControl interface {
	CanReceive() bool
	CanSend() bool
	CanSetTopic() bool
	CanAddUser() bool
	CanRemoveUser() bool
	CanSetPrivs() bool
}

// The person who creates the channel has OwnerAccess
type OwnerAccess struct{}

func (o *OwnerAccess) CanReceive() bool    { return true }
func (o *OwnerAccess) CanSend() bool       { return true }
func (o *OwnerAccess) CanSetTopic() bool   { return true }
func (o *OwnerAccess) CanRemoveUser() bool { return true }
func (o *OwnerAccess) CanAddUser() bool    { return true }
func (o *OwnerAccess) CanSetPrivs() bool   { return true }

var users map[uint64]AccessControl = map[uint64]AccessControl{
	1: &OwnerAccess{},
	2: &OwnerAccess{},
	3: &OwnerAccess{},
	4: &OwnerAccess{},
	5: &OwnerAccess{},
	6: &OwnerAccess{},
	7: &OwnerAccess{},
	8: &OwnerAccess{},
	9: &OwnerAccess{},
}

func AddUser(newUser uint64, commandSender uint64) error {
	sender, ok := users[commandSender]
	if !ok {
		return errors.New("Couldn't add user to channel: You aren't in this" +
			" channel")
	}
	if sender.CanAddUser() {
		users[newUser] = &OwnerAccess{}
		jww.INFO.Printf("Adding user %v to channel", newUser)
		return nil
	} else {
		return errors.New("Couldn't add user to channel: Access denied")
	}
}

func RemoveUser(user, commandSender uint64) error {
	sender, ok := users[commandSender]
	if !ok {
		return errors.New("Couldn't remove user from channel: You aren't in this channel")
	}
	if sender.CanRemoveUser() {
		delete(users, user)
		jww.INFO.Printf("Removing user %v from channel", user)
		return nil
	} else {
		return errors.New("Couldn't remove user from channel: Access denied")
	}
}
