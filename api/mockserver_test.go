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
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"os"
	"testing"
	"time"
)

const NumNodes = 3
const NumGWs = NumNodes
const RegPort = 5000
const GWErrorPort = 7800
const GWsStartPort = 7900
const PermErrorServerPort = 4000

var RegHandler = MockRegistration{}
var RegComms *registration.Comms
var NDFErrorReg = MockPerm_NDF_ErrorCase{}
var ErrorDef *ndf.NetworkDefinition

const ValidRegCode = "UAV6IWD6"
const InvalidRegCode = "INVALID_REG_CODE_"

var RegGWHandlers [3]*GatewayHandler = [NumGWs]*GatewayHandler{
	{LastReceivedMessage: pb.Slot{}},
	{LastReceivedMessage: pb.Slot{}},
	{LastReceivedMessage: pb.Slot{}},
}
var GWComms [NumGWs]*gateway.Comms
var GWErrComms [NumGWs]*gateway.Comms

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	// Set logging params
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(testMainWrapper(m))
}

//Happy path: test message receiver stating up
func TestClient_StartMessageReceiver_MultipleMessages(t *testing.T) {
	// Initialize client with dummy storage
	testDef := getNDF()
	for i := 0; i < NumNodes; i++ {
		gw := ndf.Gateway{
			Address: string(fmtAddress(GWErrorPort + i)),
		}
		testDef.Gateways = append(testDef.Gateways, gw)
		GWErrComms[i] = gateway.StartGateway(gw.Address,
			&GatewayHandlerMultipleMessages{}, nil, nil)

	}

	testDef.Nodes = def.Nodes

	storage := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello", "", testDef)
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// InitNetwork to gateways and reg server
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Client failed of connect: %+v", err)
	}

	err = client.GenerateKeys(nil, "password")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register with a valid registration code
	_, err = client.RegisterWithPermissioning(true, ValidRegCode)

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	err = client.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	err = client.session.StoreSession()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Login to gateway
	_, err = client.Login("password")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	cb := func(err error) {
		t.Log(err)
	}

	err = client.StartMessageReceiver(cb)
	if err != nil {
		t.Errorf("%+v", err)
	}

	time.Sleep(3 * time.Second)
	for _, gw := range GWErrComms {
		gw.DisconnectAll()
	}

}

func TestRegister_ValidPrecannedRegCodeReturnsZeroID(t *testing.T) {
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
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register precanned user with all gateways
	regRes, err := client.RegisterWithPermissioning(true, ValidRegCode)

	// Verify registration succeeds with valid precanned registration code
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
	disconnectServers()
}

// Verify that registering with an invalid registration code will fail
func TestRegister_InvalidPrecannedRegCodeReturnsError(t *testing.T) {
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
	//Generate keys s.t. reg status is prepped for registration
	err = client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register with invalid reg code
	uid, err := client.RegisterWithPermissioning(true, InvalidRegCode)
	if err == nil {
		t.Errorf("Registration worked with invalid registration code! UID: %v", uid)
	}

	//Disconnect and shutdown servers
	disconnectServers()
}

//Test that not running generateKeys results in an error. Without running the aforementioned function,
// the registration state should be invalid and it should not run
func TestRegister_InvalidRegState(t *testing.T) {
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
	//Individually run the helper functions for GenerateKeys, put info into client
	privKey, pubKey, err := generateRsaKeys(nil)
	if err != nil {
		t.Errorf("%+v", err)
	}
	cmixGrp, e2eGrp := generateGroups(def)
	salt, _, usr, err := generateUserInformation(pubKey)
	if err != nil {
		t.Errorf("%+v", err)
	}
	e2ePrivKey, e2ePubKey, err := generateE2eKeys(cmixGrp, e2eGrp)
	if err != nil {
		t.Errorf("%+v", err)
	}
	cmixPrivKey, cmixPubKey, err := generateCmixKeys(cmixGrp)

	client.session = user.NewSession(nil, usr, pubKey, privKey, cmixPubKey, cmixPrivKey, e2ePubKey, e2ePrivKey, salt, cmixGrp, e2eGrp, "")

	//
	_, err = client.RegisterWithPermissioning(false, ValidRegCode)
	if err == nil {
		t.Errorf("Registration worked with invalid registration state!")
	}

}

func TestRegister_DeletedUserReturnsErr(t *testing.T) {
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

	// ...
	tempUser, _ := user.Users.GetUser(id.NewUserFromUint(5, t))
	user.Users.DeleteUser(id.NewUserFromUint(5, t))
	err = client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register
	_, err = client.RegisterWithPermissioning(true, ValidRegCode)
	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s", err.Error())
	}

	// ...
	user.Users.UpsertUser(tempUser)
	//Disconnect and shutdown servers
	disconnectServers()
}

func TestSend(t *testing.T) {
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

	err = client.GenerateKeys(nil, "password")

	// Register with a valid registration code
	userID, err := client.RegisterWithPermissioning(true, ValidRegCode)

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	err = client.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	err = client.session.StoreSession()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Login to gateway
	_, err = client.Login("password")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	cb := func(err error) {
		t.Log(err)
	}

	err = client.StartMessageReceiver(cb)

	if err != nil {
		t.Errorf("Could not start message reception: %+v", err)
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
	disconnectServers()
}

func TestLogout(t *testing.T) {
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

	// Logout before logging in should return an error
	err = client.Logout()

	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}

	err = client.GenerateKeys(nil, "password")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register with a valid registration code
	_, err = client.RegisterWithPermissioning(true, ValidRegCode)

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	err = client.RegisterWithNodes()
	if err != nil {
		t.Error(err)
	}

	// Login to gateway
	_, err = client.Login("password")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	cb := func(err error) {
		t.Log(err)
	}

	err = client.StartMessageReceiver(cb)

	if err != nil {
		t.Errorf("Failed to start message reciever: %s", err.Error())
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

	disconnectServers()
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	def = getNDF()
	ErrorDef = getNDF()
	// Start mock registration server and defer its shutdown
	def.Registration = ndf.Registration{
		Address: fmtAddress(RegPort),
	}
	ErrorDef.Registration = ndf.Registration{
		Address: fmtAddress(PermErrorServerPort),
	}

	for i := 0; i < NumNodes; i++ {
		nIdBytes := make([]byte, id.NodeIdLen)
		nIdBytes[0] = byte(i)
		n := ndf.Node{
			ID: nIdBytes,
		}
		def.Nodes = append(def.Nodes, n)
		ErrorDef.Nodes = append(ErrorDef.Nodes, n)
	}
	startServers()
	defer testWrapperShutdown()
	return m.Run()
}

func testWrapperShutdown() {

	for _, gw := range GWComms {
		gw.Shutdown()

	}
	RegComms.Shutdown()
}

func fmtAddress(port int) string { return fmt.Sprintf("localhost:%d", port) }

func getNDF() *ndf.NetworkDefinition {
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
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}

func startServers() {
	//func StartRegistrationServer(id, localServer string, handler Handler, certPEMblock, keyPEMblock []byte) *Comms {
	RegComms = registration.StartRegistrationServer( "testServer", def.Registration.Address, &RegHandler, nil, nil)
	def.Gateways = make([]ndf.Gateway, 0)

	//Start up gateways
	for i, handler := range RegGWHandlers {

		gw := ndf.Gateway{
			Address: fmtAddress(GWsStartPort + i),
		}

		def.Gateways = append(def.Gateways, gw)
		GWComms[i] = gateway.StartGateway("testGateway", gw.Address, handler, nil, nil)
	}
}

func disconnectServers() {
	for _, gw := range GWComms {
		gw.DisconnectAll()

	}
	RegComms.DisconnectAll()
}
