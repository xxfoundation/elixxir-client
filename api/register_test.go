////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package api

import (
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

//Test that a registered session may be stored & recovered
func TestRegistrationGob(t *testing.T) {
	// Get a Client
	testClient, err := NewClient(&globals.RamStorage{}, ".ekv-registergob", "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}
	err = testClient.GenerateKeys(nil, "1234")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	io.SessionV2.SetRegState(user.KeyGenComplete)

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "WTROXJ33")
	if err != nil {
		t.Error(err)
	}

	err = testClient.session.StoreSession()
	if err != nil {
		t.Error(err)
	}

	// get the gob out of there again
	Session, err := user.LoadSession(testClient.storage,
		"1234")
	if err != nil {
		t.Error(err)
	}

	VerifyRegisterGobUser(Session, t)

	disconnectServers()
}

//Happy path for a non precen user
func TestClient_Register(t *testing.T) {
	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, ".ekv-clientregister", "", def)

	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}

	err = testClient.GenerateKeys(nil, "password")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// fixme please (and all other places where this call is above RegisterWithPermissioning in tests)
	io.SessionV2.SetRegState(user.KeyGenComplete)
	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "WTROXJ33")
	if err != nil {
		t.Error(err)
	}

	err = testClient.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	// get the gob out of there again
	Session, err := user.LoadSession(testClient.storage,
		"password")
	if err != nil {
		t.Error(err)
	}

	VerifyRegisterGobUser(Session, t)

	disconnectServers()
}

//Verify the user from the session make in the registration above matches expected user
func VerifyRegisterGobUser(session user.Session, t *testing.T) {

	expectedUser := id.NewIdFromUInt(5, id.User, t)

	if !session.GetCurrentUser().User.Cmp(expectedUser) {
		t.Errorf("Incorrect User ID; \n   expected: %q \n   recieved: %q",
			expectedUser, session.GetCurrentUser().User)
	}
}

// Verify that a valid precanned user can register
func TestRegister_ValidRegParams___(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, ".ekv-validregparams", "", def)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	err = client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	io.SessionV2.SetRegState(user.KeyGenComplete)
	// Register precanned user with all gateways
	regRes, err := client.RegisterWithPermissioning(false, ValidRegCode)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *&id.ZeroUser {
		t.Errorf("Invalid registration number received: %+v", *regRes)
	}
	err = client.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	//Disconnect and shutdown servers
	disconnectServers()
}
