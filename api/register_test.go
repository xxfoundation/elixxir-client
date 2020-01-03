package api

import (
	"crypto/sha256"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
)

func TestRegistrationGob(t *testing.T) {
	// Get a Client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
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

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6")
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
	VerifyRegisterGobKeys(Session, testClient.topology, t)

	disconnectServers()
}

//Happy path for a non precen user
func TestClient_Register(t *testing.T) {
	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)

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

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6")
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

	//Probs can't do this as there is now a sense of randomness??
	//VerifyRegisterGobKeys(Session, testClient.topology, t)
	disconnectServers()
}

func VerifyRegisterGobUser(session user.Session, t *testing.T) {

	expectedUser := id.NewUserFromUint(5, t)

	if reflect.DeepEqual(session.GetCurrentUser().User, &expectedUser) {
		t.Errorf("Incorrect User ID; \n   expected %q \n   recieved: %q",
			expectedUser, session.GetCurrentUser().User)
	}
}

func VerifyRegisterGobKeys(session user.Session, topology *connect.Circuit, t *testing.T) {
	cmixGrp, _ := getGroups()
	h := sha256.New()
	h.Write([]byte(string(40005)))
	expectedTransmissionBaseKey := cmixGrp.NewIntFromBytes(h.Sum(nil))

	if session.GetNodeKeys(topology)[0].TransmissionKey.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			session.GetNodeKeys(topology)[0].TransmissionKey.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}

}

/*
//TODO: Fix this test
//Error path: register with udb, but udb is not set up to return a message
func TestClient_RegisterWithUDB_NoUDB(t *testing.T) {
	// Get a Client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}

	err = testClient.GenerateKeys(nil, "password")
	if err != nil {
		t.Errorf("Failed to generate keys: %+v", err)
	}

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, ValidRegCode)
	if err != nil {
		t.Error(err)
	}

	err = testClient.RegisterWithNodes()
	if err != nil {
		t.Error(err.Error())
	}

	// Login to gateway
	_, err = testClient.Login("password")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	cb := func(err error) {
		t.Log(err)
	}

	err = testClient.StartMessageReceiver(cb)

	if err != nil {
		t.Errorf("Could not start message reception: %+v", err)
	}

	err = testClient.RegisterWithUDB("test", 1 * time.Second)
	if err != nil {
		return
	}
	t.Errorf("Expected error path: should not successfully register with udb")
}
*/

// Verify that a valid precanned user can register
func TestRegister_ValidRegParams___(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", def)
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

	// Register precanned user with all gateways
	regRes, err := client.RegisterWithPermissioning(false, ValidRegCode)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %+v", *regRes)
	}
	err = client.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	//Disconnect and shutdown servers
	disconnectServers()
}
