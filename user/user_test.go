////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"crypto/sha256"
	"gitlab.com/elixxir/crypto/cyclic"
	"testing"
	"gitlab.com/elixxir/primitives/userid"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the Registry interface
func TestUserRegistry(t *testing.T) {
	// Test if CountUsers correctly counts the hard-coded demo users
	if Users.CountUsers() != NUM_DEMO_USERS {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}
	// Test the integration of the LookupUser, UserHash and GetUser functions
	for i := 0; i < len(DemoUserNicks); i++ {
		currentID := id.NewUserIDFromUint(uint64(i+1), t)
		reg, ok := Users.LookupUser(currentID.RegistrationCode())
		if !ok {
			t.Errorf("Couldn't lookup user %q with code %v", *currentID,
				currentID.RegistrationCode())
		}
		usr, ok := Users.GetUser(reg)
		if !ok {
			t.Logf("Reg codes of both: %v, %v", reg.RegistrationCode(),
				currentID.RegistrationCode())
			t.Errorf("Couldn't get user %q corresponding to user %q",
				*reg, *currentID)
		}
		if usr.Nick != DemoUserNicks[i] {
			t.Errorf("Nickname incorrectly set. Expected: %v Actual: %v",
				DemoUserNicks[i], usr.Nick)
		}
	}
	// Test the NewUser function
	newID := id.NewUserIDFromUint(2002, t)
	usr := Users.NewUser(newID, "Will I am")

	if usr.Nick != "Will I am" {
		t.Errorf("User name should be 'Will I am', but is %v instead", usr.Nick)
	}

	// Test that UpsertUser successfully adds a user to the usermap
	userCount := Users.CountUsers()
	Users.UpsertUser(usr)
	if Users.CountUsers() != userCount+1 {
		t.Errorf("Upsert did not add a new user. User count is incorrect")
	}
	newUsr, suc := Users.GetUser(newID)
	if !suc {
		t.Errorf("Upsert did not add the test user correctly. " +
			"The ID was not found by GetUser.")
	}
	if newUsr.Nick != "Will I am" {
		t.Errorf("Upsert did not add the test user correctly. "+
			"The set nickname was incorrect. Expected: Will I am, "+
			"Actual: %v", newUsr.Nick)
	}

	// Test LookupKeys
	keys, suc := Users.LookupKeys(id.NewUserIDFromUint(1, t))
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
	Users.DeleteUser(id.NewUserIDFromUint(2, t))

	_, ok := Users.GetUser(id.NewUserIDFromUint(2, t))
	if ok {
		t.Errorf("User %v has not been deleted succesfully!", usr.Nick)
	}
}

// Doesn't actually do any testing, but can print the registration codes for
// the first several users
func TestPrintRegCodes(t *testing.T) {
	for i := 1; i <= NUM_DEMO_USERS; i++ {
		currentID := id.NewUserIDFromUint(uint64(i), t)
		t.Logf("%v:\t%v", i, currentID.RegistrationCode())
	}
}
