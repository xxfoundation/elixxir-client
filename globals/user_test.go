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

	if Users.CountUsers() != 5 {
		t.Errorf("CountUsers: Start size of userRegistry not zero!")
	}

	usr, _ := Users.GetUser(1)

	if usr.Nick != "Phineas Flynn" {
		t.Errorf("User 1 is not 'Phineas Flynn'")
	}

	usr, _ = Users.GetUser(2)

	if usr.Nick != "Ferb Flynn" {
		t.Errorf("User 2 is not 'Ferb Flynn'")
	}

	usr, _ = Users.GetUser(3)

	if usr.Nick != "Cadance Flynn" {
		t.Errorf("User 3 is not 'Cadance Flynn'")
	}

	usr, _ = Users.GetUser(4)

	if usr.Nick != "Perry the Platypus" {
		t.Errorf("User 4 is not 'Perry the Platypus'")
	}

	usr, _ = Users.GetUser(5)

	if usr.Nick != "Heinz Doofenshmirtz" {
		t.Errorf("User 5 is not 'Heinz Doofenshmirtz'")
	}

}
