package channelbot

import "testing"

// Test RunAdd on correct inputs
func TestRunAdd(t *testing.T) {
	addingUser := uint64(45)
	users = map[uint64]AccessControl{
		addingUser: &OwnerAccess{},
	}
	// 45 adds one user to the channel
	err := RunAdd([]string{"79"}, addingUser)
	if err != nil {
		t.Error(err.Error())
	}
	_, ok := users[79]
	if !ok {
		t.Error("Couldn't add user to the channel with RunAdd")
	}
}

// Test RunRemove on correct inputs
func TestRunRemove(t *testing.T) {
	removingUser := uint64(45)
	removedUser := uint64(79)
	users = map[uint64]AccessControl{
		removingUser: &OwnerAccess{},
		removedUser:  &OwnerAccess{},
	}
	// 45 removes one user to the channel
	err := RunRemove([]string{"79"}, removingUser)
	if err != nil {
		t.Error(err.Error())
	}
	_, ok := users[79]
	if ok {
		t.Error("Couldn't remove user to the channel with RunRemove")
	}
}

// Add and remove a user with ParseCommand
func TestParseCommand(t *testing.T) {
	owner := uint64(28)
	users = map[uint64]AccessControl{
		owner: &OwnerAccess{},
	}

	err := ParseCommand("/add 71", owner)
	if err != nil {
		t.Error("Couldn't add user with ParseCommand: " + err.Error())
	}
	_, ok := users[71]
	if !ok {
		t.Error("Didn't add user with ParseCommand")
	}

	err = ParseCommand("/remove 71", owner)
	if err != nil {
		t.Error("Couldn't remove user with ParseCommand: " + err.Error())
	}
	_, ok = users[71]
	if ok {
		t.Error("Didn't remove user with ParseCommand")
	}
}

// Test error cases with ParseCommand
func TestParseCommand2(t *testing.T) {
	err := ParseCommand("/brobdingnan",1)
	if err == nil {
		t.Error("ParseCommand: No error from brobdingnan")
	}

	err = ParseCommand("/add",1 )
	if err == nil {
		t.Error("ParseCommand: No error from adding no users")
	}

	err = ParseCommand("/add 21 59 82", 1)
	if err == nil {
		t.Error("ParseCommand: No error from adding multiple users")
	}

	err = ParseCommand("/add ggGGGgg", 1)
	if err == nil {
		t.Error("ParseCommand: No error from adding a rubbish user")
	}

	err = ParseCommand("", 1)
	if err == nil {
		t.Error("ParseCommand: No error from empty command string")
	}

	err = ParseCommand("/", 1)
	if err == nil {
		t.Error("ParseCommand: No error from single forward slash")
	}
}
