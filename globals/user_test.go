////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserRegistry(t *testing.T) {

	if Users.CountUsers() != NUM_DEMO_USERS {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}

	for i := 0; i < len(DEMO_NICKS); i++ {
		reg, _ := Users.LookupUser(UserHash(uint64(i+1)))
		usr, _ := Users.GetUser(reg)
		if usr.Nick != DEMO_NICKS[i] {
			t.Errorf("Nickname incorrectly set. Expected: %v Actual: %v",
				DEMO_NICKS[i], usr.Nick)
		}
	}

	usr := Users.NewUser(2002, "Will I am")

	if usr.Nick != "Will I am" {
		t.Errorf("User name should be 'Will I am', but is %v instead", usr.Nick)
	}

	Users.UpsertUser(usr)

	if Users.CountUsers() == NUM_DEMO_USERS {
		t.Errorf("Upsert did not work properly")
	}


	Users.DeleteUser(2)

	_, ok := Users.GetUser(2)
	if ok {
		t.Errorf("User %v has not been deleted succesfully!", usr.Nick)
	}
}

