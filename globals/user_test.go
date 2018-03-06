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

	if Users.CountUsers() != 1000 {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}

	reg, _ := Users.LookupUser(10001)
	usr, _ := Users.GetUser(reg)
	if usr.Nick != "David" {
		t.Errorf("User 10001 is not 'David'")
	}

	reg, _ = Users.LookupUser(10002)
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Jim" {
		t.Errorf("User 10002 is not 'Jim'")
	}

	reg, _ = Users.LookupUser(10003)
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Ben" {
		t.Errorf("User 10008 is not 'Will'")
	}

	reg, _ = Users.LookupUser(10004)
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Rick" {
		t.Errorf("User 10003 is not 'Ben'")
	}

	reg, _ = Users.LookupUser(10005)
	usr, _ = Users.GetUser(reg)
	if usr.Nick != "Spencer" {
		t.Errorf("User 10004 is not 'Rick'")
	}

}
