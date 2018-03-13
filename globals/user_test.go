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

	reg, _ := Users.LookupUser(UserHash(1))
	usr, _ := Users.GetUser(reg)
	if usr.Nick != "David" {
		t.Errorf("User 10001 is not 'David'")
	}

	reg, _ = Users.LookupUser(UserHash(2))
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Jim" {
		t.Errorf("User 10002 is not 'Jim'")
	}

	reg, _ = Users.LookupUser(UserHash(3))
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Ben" {
		t.Errorf("User 10003 is not 'Ben'")
	}

	reg, _ = Users.LookupUser(UserHash(4))
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Rick" {
		t.Errorf("User 10004 is not 'Rick'")
	}

	reg, _ = Users.LookupUser(UserHash(5))
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Spencer" {
		t.Errorf("User 10005 is not 'Spencer'")
	}

}