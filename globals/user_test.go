////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserRegistry(t *testing.T) {
	// Test if CountUsers correctly counts the hard-coded demo users
	if Users.CountUsers() != NUM_DEMO_USERS {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}
	// Test the integration of the LookupUser, UserHash and GetUser functions
	for i := 0; i < len(DEMO_USER_NICKS); i++ {
		uid := UserID(i + 1)
		reg, _ := Users.LookupUser(string(UserHash(uid)))
		usr, _ := Users.GetUser(reg)
		if usr.Nick != DEMO_USER_NICKS[i] {
			t.Errorf("Nickname incorrectly set. Expected: %v Actual: %v",
				DEMO_USER_NICKS[i], usr.Nick)
		}
	}
	// Test the NewUser function
	id := UserID(2002)
	usr := Users.NewUser(id, "Will I am")

	if usr.Nick != "Will I am" {
		t.Errorf("User name should be 'Will I am', but is %v instead", usr.Nick)
	}

	// Test that UpsertUser successfully adds a user to the usermap
	userCount := Users.CountUsers()
	Users.UpsertUser(usr)
	if Users.CountUsers() != userCount+1 {
		t.Errorf("Upsert did not add a new user. User count is incorrect")
	}
	newUsr, suc := Users.GetUser(id)
	if !suc {
		t.Errorf("Upsert did not add the test user correctly. " +
			"The UserID was not found by GetUser.")
	}
	if newUsr.Nick != "Will I am" {
		t.Errorf("Upsert did not add the test user correctly. "+
			"The set nickname was incorrect. Expected: Will I am, "+
			"Actual: %v", newUsr.Nick)
	}

	// Test LookupKeys
	keys, suc := Users.LookupKeys(UserID(1))
	if !suc {
		t.Errorf("LookupKeys failed to find a valid user.")
	}
	h := sha256.New()
	h.Write([]byte(string(20001)))
	key := cyclic.NewIntFromBytes(h.Sum(nil))
	if keys.TransmissionKeys.Base.Text(16) != key.Text(16) {
		t.Errorf("LookupKeys returned an incorrect key. "+
			"Expected:%v \nActual%v", key.Text(16),
			keys.TransmissionKeys.Base.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(30001)))
	key = cyclic.NewIntFromBytes(h.Sum(nil))
	if keys.TransmissionKeys.Recursive.Text(16) != key.Text(16) {
		t.Errorf("LookupKeys returned an incorrect key. "+
			"Expected:%v \nActual%v", key.Text(16),
			keys.TransmissionKeys.Recursive.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(40001)))
	key = cyclic.NewIntFromBytes(h.Sum(nil))
	if keys.ReceptionKeys.Base.Text(16) != key.Text(16) {
		t.Errorf("LookupKeys returned an incorrect key. "+
			"Expected:%v \nActual%v", key.Text(16),
			keys.ReceptionKeys.Base.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(50001)))
	key = cyclic.NewIntFromBytes(h.Sum(nil))
	if keys.ReceptionKeys.Recursive.Text(16) != key.Text(16) {
		t.Errorf("LookupKeys returned an incorrect key. "+
			"Expected:%v \nActual%v", key.Text(16),
			keys.ReceptionKeys.Recursive.Text(16))
	}
	// Test delete user
	Users.DeleteUser(2)

	_, ok := Users.GetUser(2)
	if ok {
		t.Errorf("User %v has not been deleted succesfully!", usr.Nick)
	}
}
