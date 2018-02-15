package globals

import (
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserSession(t *testing.T) {

	if Session.GetCurrentUser() != nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	}

	if !Session.Login(1) {
		t.Errorf("Error: Unable to login with valid user!")
	}

	if Session.Login(1) {
		t.Errorf("Error: Able to login with invalid user!")
	}

	if Session.GetCurrentUser() == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	}

}
