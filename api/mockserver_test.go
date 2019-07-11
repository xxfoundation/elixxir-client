////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"os"
	"testing"
)

const NumGWs = 3
const RegPort = 5000
const RegGWsStartPort = 10000
const SessionGWPort = 15000

var RegAddress = fmtAddress(RegPort)
var RegGWAddresses [NumGWs]string
var SessionGWAddress = fmtAddress(SessionGWPort)
var RegGWComms [NumGWs]*gateway.GatewayComms
var RegComms *registration.RegistrationComms
var SessionGWComms *gateway.GatewayComms

const ValidRegCode = "UAV6IWD6"
const InvalidRegCode = "INVALID_REG_CODE"

var RegGWHandlers = [NumGWs]*TestInterface{
	{LastReceivedMessage: pb.Slot{}},
	{LastReceivedMessage: pb.Slot{}},
	{LastReceivedMessage: pb.Slot{}},
}

var RegHandler = MockRegistration{}

var SessionGWHandler = TestInterface{LastReceivedMessage: pb.Slot{}}
var Session user.SessionObj

var


// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	// Set logging params
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(testMainWrapper(m))
}

// Verify that a valid precanned user can register
func TestRegister_ValidPrecannedRegCodeReturnsZeroID(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", getNDF())
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", "", "")

	// Register precanned user with all gateways
	regRes, err := client.Register(true, ValidRegCode,
		"")

	// Verify registration succeeds with valid precanned registration code
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
}

// Verify that a valid precanned user can register
func TestRegister_ValidRegParams___(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", RegAddress, "")

	// Register precanned user with all gateways
	regRes, err := client.Register(false, ValidRegCode,
		"", false, getGroup())
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
}

// Verify that registering with an invalid registration code will fail
func TestRegister_InvalidPrecannedRegCodeReturnsError(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", "", "")

	// Register with invalid reg code
	_, err = client.Register(true, InvalidRegCode,
		"", false, getGroup())
	if err == nil {
		t.Error("Registration worked with invalid registration code!")
	}
}

func TestRegister_DeletedUserReturnsErr(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", "", "")

	// ...
	tempUser, _ := user.Users.GetUser(id.NewUserFromUint(5, t))
	user.Users.DeleteUser(id.NewUserFromUint(5, t))

	// Register
	_, err = client.Register(true, ValidRegCode,
		"", false, getGroup())
	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s", err.Error())
	}

	// ...
	user.Users.UpsertUser(tempUser)
}

func TestSend(t *testing.T) {
	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", "", "")

	// Register with a valid registration code
	userID, err := client.Register(true, ValidRegCode,
		"", false, getGroup())

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	// Login to gateway
	_, err = client.Login(userID, "")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	// Test send with invalid sender ID
	err = client.Send(
		APIMessage{
			SenderID:    id.NewUserFromUint(12, t),
			Payload:     []byte("test"),
			RecipientID: userID,
		},
	)

	if err != nil {
		// TODO: would be nice to catch the sender but we
		// don't have the interface/mocking for that.
		t.Errorf("error on first message send: %+v", err)
	}

	// Test send with valid inputs
	err = client.Send(APIMessage{SenderID: userID, Payload: []byte("test"),
		RecipientID: client.GetCurrentUser()})

	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}

	err = client.Logout()

	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}
}

func TestLogout(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Connect to gateways and reg server
	client.Connect(RegGWAddresses[:], "", "", "")

	// Logout before logging in should return an error
	err = client.Logout()

	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}

	// Register with a valid registration code
	userID, err := client.Register(true, ValidRegCode,
		"", false, getGroup())

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	// Login to gateway
	_, err = client.Login(userID, "")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	err = client.Logout()

	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}

	// Logout after logout has been called should return an error
	err = client.Logout()

	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	// Start mock gateways used by registration and defer their shutdown (may not be needed)
	for i, handler := range RegGWHandlers {
		RegGWAddresses[i] = fmtAddress(RegGWsStartPort + i)
		gw := gateway.StartGateway(RegGWAddresses[i],
			handler, "", "")
		RegGWComms[i] = gw
	}

	// Start mock registration server and defer its shutdown
	RegComms = registration.StartRegistrationServer(RegAddress,
		&RegHandler, "", "")

	// Start session gateway and defer its shutdown
	SessionGWComms = gateway.StartGateway(SessionGWAddress,
		&SessionGWHandler, "", "")

	defer testWrapperShutdown()
	return m.Run()
}

func testWrapperShutdown() {
	for _, gw := range RegGWComms {
		gw.Shutdown()
	}
	RegComms.Shutdown()
	SessionGWComms.Shutdown()
}

func getGroup() *cyclic.Group {
	return globals.InitCrypto()
}

func fmtAddress(port int) string { return fmt.Sprintf("localhost:%d", port) }

func getNDF()*ndf.NetworkDefinition{
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
			"847AEF49F66E43873",
			SmallPrime: "2",
			Generator: "2",
		},
		CMIX:ndf.Group{
			Prime: "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
				"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
				"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
				"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
				"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
				"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
				"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
				"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
				"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
				"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
				"15728E5A8AACAA68FFFFFFFFFFFFFFFF",
				SmallPrime:"2",
				Generator:"2",
		},
	}
}