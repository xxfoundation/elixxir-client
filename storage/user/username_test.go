package user

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Test normal function and errors for User's SetUsername function
func TestUser_SetUsername(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	u, err := NewUser(kv, uid, []byte("salt"), &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	u1 := "zezima"
	u2 := "dunkey"
	err = u.SetUsername(u1)
	if err != nil {
		t.Errorf("Failed to set username: %+v", err)
	}

	err = u.SetUsername(u2)
	if err == nil {
		t.Error("Did not error when attempting to set a new username")
	}

	o, err := u.kv.Get(usernameKey)
	if err != nil {
		t.Errorf("Didn't get username from user kv store: %+v", err)
	}

	if string(o.Data) != u1 {
		t.Errorf("Expected username was not stored.\nExpected: %s\tReceived: %s", u1, string(o.Data))
	}
}

// Test functionality of User's GetUsername function
func TestUser_GetUsername(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	u, err := NewUser(kv, uid, []byte("salt"), &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	_, err = u.GetUsername()
	if err == nil {
		t.Error("GetUsername should return an error if username is not set")
	}

	u1 := "zezima"
	u.username = u1
	username, err := u.GetUsername()
	if err != nil {
		t.Errorf("Failed to get username when set: %+v", err)
	}
	if username != u1 {
		t.Errorf("Somehow got the wrong username")
	}
}

// Test the loadUsername helper function
func TestUser_loadUsername(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	u, err := NewUser(kv, uid, []byte("salt"), &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	u1 := "zezima"

	err = u.kv.Set(usernameKey, &versioned.Object{
		Version:   currentUsernameVersion,
		Timestamp: time.Now(),
		Data:      []byte(u1),
	})
	u.loadUsername()
	if u.username != u1 {
		t.Errorf("Username was not properly loaded from kv.\nExpected: %s, Received: %s", u1, u.username)
	}
}
