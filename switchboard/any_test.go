package switchboard

import (
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

//tests that AnyUser returns the correct user
func TestAnyUser(t *testing.T) {
	au := AnyUser()
	if !au.Cmp(&id.ZeroUser) {
		t.Errorf("Wrong user returned from AnyUser")
	}
}
