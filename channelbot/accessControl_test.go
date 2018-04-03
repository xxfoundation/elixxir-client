package channelbot

import (
	"testing"
)

func TestAddUser(t *testing.T) {
	users = make(map[uint64]AccessControl)
	ownerId := uint64(55)
	newUserId := uint64(25)
	// set up an owner
	users[ownerId] = &OwnerAccess{}
	// the owner adds a user
	err := AddUser(newUserId, ownerId)
	if err != nil {
		t.Errorf("Owner couldn't add new user: %v", err.Error())
	}

	// someone outside of the channel removes someone in the channel
	err = AddUser(ownerId, 1)
	if err == nil {
		t.Error("No error from a user not in the channel adding a user in" +
			" the channel")
	} else {
		t.Log("A user not in the channel couldn't add a user in the" +
			" channel:", err.Error())
	}
}

func TestRemoveUser(t *testing.T) {
	ownerId := uint64(88)
	normalUserId := uint64(76)
	users = map[uint64]AccessControl{
		ownerId:              &OwnerAccess{},
		normalUserId:         &OwnerAccess{},
	}

	// ownerId bans normalUserId
	err := RemoveUser(normalUserId, ownerId)
	if err != nil {
		t.Errorf("Owner couldn't remove user: %v", err.Error())
	}
	_, ok := users[normalUserId]
	if ok {
		t.Errorf("User %v wasn't removed", normalUserId)
	}

	// owner mistakenly bans someone who's not in the channel
	err = RemoveUser(1, ownerId)
	// this should fail silently
	if err != nil {
		t.Error("Removing a user not in the channel: %v", err.Error())
	}

	// someone outside of the channel bans someone in the channel
	err = RemoveUser(ownerId, 1)
	if err == nil {
		t.Error("No error from a user not in the channel removing a user in" +
			" the channel")
	} else {
		t.Log("A user not in the channel couldn't remove a user in the" +
			" channel:", err.Error())
	}
}
