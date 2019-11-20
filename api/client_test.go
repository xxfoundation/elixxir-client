////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"reflect"
	"strings"
	"testing"
	"time"
)

var TestKeySize = 768

func dummyConnectionStatusHandler(status uint32, timeout int) {
	return
}

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

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6", "", "", "password", nil)
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
	VerifyRegisterGobKeys(Session, testClient.topology, t)

	disconnectServers()
}

//Happy path for a non precan user
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
	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6", "", "", "password", nil)
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

//Test GetRemoveVersion returns the expected value (globals.SEMVER)
func TestClient_GetRemoteVersion(t *testing.T) {
	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)

	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}
	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(false, "UAV6IWD6", "", "", "password", nil)
	if err != nil {
		t.Error(err)
	}
	//Get it from a good version
	if strings.Compare(globals.SEMVER, testClient.GetRegistrationVersion()) != 0 {
		t.Errorf("Client not up to date: Recieved %v Expected %v", testClient.GetRegistrationVersion(), globals.SEMVER)
	}

}

func TestClient_Register_NoUpdatingNDF(t *testing.T) {
	mockRegError := registration.StartRegistrationServer(errorDef.Registration.Address, &NDFErrorReg,
		nil, nil)
	defer mockRegError.Shutdown()
	def.Gateways = make([]ndf.Gateway, 0)

	//Start up gateways
	for i, _ := range RegGWHandlers {

		gw := ndf.Gateway{
			Address: fmtAddress(GWsStartPort + i),
		}

		def.Gateways = append(def.Gateways, gw)
	}

	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)

	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Expected error path, should not have gotted ndf from connect")
	}
}

//Error path: Force an error in connect through mockPerm_CheckVersion_ErrorCase
func TestClient_CheckVersionErr(t *testing.T) {
	mockRegError := registration.StartRegistrationServer(errorDef.Registration.Address,
		&MockPerm_CheckVersion_ErrorCase{}, nil, nil)
	defer mockRegError.Shutdown()
	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", errorDef)

	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		return
	}
	t.Error("Expected error case: UpdateVersion should have returned an error")
}

//Error Path: Force error in connect by providing a bad version through mockPerm
func TestClient_CheckVersion_BadVersion(t *testing.T) {
	mockRegError := registration.StartRegistrationServer(errorDef.Registration.Address,
		&MockPerm_CheckVersion_BadVersion{}, nil, nil)
	defer mockRegError.Shutdown()

	//Make mock client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", errorDef)

	if err != nil {
		t.Error(err)
	}

	//Check version here should return a version that does not match the global being checked
	err = testClient.InitNetwork()
	if err != nil {
		return
	}
	t.Errorf("Expected error case: Version from mock permissioning should not match expected")
}

func VerifyRegisterGobUser(session user.Session, t *testing.T) {

	expectedUser := id.NewUserFromUint(5, t)

	if reflect.DeepEqual(session.GetCurrentUser().User, &expectedUser) {
		t.Errorf("Incorrect User ID; \n   expected %q \n   recieved: %q",
			expectedUser, session.GetCurrentUser().User)
	}
}

func VerifyRegisterGobKeys(session user.Session, topology *circuit.Circuit, t *testing.T) {
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

// Make sure that a formatted text message can deserialize to the text
// message we would expect
func TestFormatTextMessage(t *testing.T) {
	msgText := "Hello"
	msg := FormatTextMessage(msgText)
	parsed := cmixproto.TextMessage{}
	err := proto.Unmarshal(msg, &parsed)
	// Make sure it parsed correctly
	if err != nil {
		t.Errorf("Got error parsing text message: %v", err.Error())
	}
	// Check the field that we explicitly set by calling the method
	if parsed.Message != msgText {
		t.Errorf("Got wrong text from parsing message. Got %v, expected %v",
			parsed.Message, msgText)
	}
	// Make sure that timestamp is reasonable
	timeDifference := time.Now().Unix() - parsed.Time
	if timeDifference > 2 || timeDifference < -2 {
		t.Errorf("Message timestamp was off by more than one second. "+
			"Original time: %x, parsed time: %x", time.Now().Unix(), parsed.Time)
	}
	t.Logf("message: %q", msg)
}

func TestParsedMessage_GetSender(t *testing.T) {
	pm := ParsedMessage{}
	sndr := pm.GetSender()

	if !reflect.DeepEqual(sndr, []byte{}) {
		t.Errorf("Sender not empty from typed message")
	}
}

func TestParsedMessage_GetPayload(t *testing.T) {
	pm := ParsedMessage{}
	payload := []byte{0, 1, 2, 3}
	pm.Payload = payload
	pld := pm.GetPayload()

	if !reflect.DeepEqual(pld, payload) {
		t.Errorf("Output payload does not match input payload: %v %v", payload, pld)
	}
}

func TestParsedMessage_GetRecipient(t *testing.T) {
	pm := ParsedMessage{}
	rcpt := pm.GetRecipient()

	if !reflect.DeepEqual(rcpt, []byte{}) {
		t.Errorf("Recipient not empty from typed message")
	}
}

func TestParsedMessage_GetMessageType(t *testing.T) {
	pm := ParsedMessage{}
	var typeTest int32
	typeTest = 6
	pm.Typed = typeTest
	typ := pm.GetMessageType()

	if typ != typeTest {
		t.Errorf("Returned type does not match")
	}
}

func TestParse(t *testing.T) {
	ms := parse.Message{}
	ms.Body = []byte{0, 1, 2}
	ms.MessageType = int32(cmixproto.Type_NO_TYPE)
	ms.Receiver = id.ZeroID
	ms.Sender = id.ZeroID

	messagePacked := ms.Pack()

	msOut, err := ParseMessage(messagePacked)

	if err != nil {
		t.Errorf("Message failed to parse: %s", err.Error())
	}

	if msOut.GetMessageType() != int32(ms.MessageType) {
		t.Errorf("Types do not match after message parse: %v vs %v", msOut.GetMessageType(), ms.MessageType)
	}

	if !reflect.DeepEqual(ms.Body, msOut.GetPayload()) {
		t.Errorf("Bodies do not match after message parse: %v vs %v", msOut.GetPayload(), ms.Body)
	}

}

// Test that registerUserE2E correctly creates keys and adds them to maps
func TestRegisterUserE2E(t *testing.T) {
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	rng := csprng.NewSystemRNG()
	cmixGrp, e2eGrp := getGroups()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)

	myPrivKeyCyclic := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())
	myPubKeyCyclic := e2eGrp.ExpG(myPrivKeyCyclic, e2eGrp.NewMaxInt())

	partnerPubKeyCyclic := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())

	privateKeyRSA, _ := rsa.GenerateKey(rng, TestKeySize)
	publicKeyRSA := rsa.PublicKey{PublicKey: privateKeyRSA.PublicKey}
	regSignature := make([]byte, 8)

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(testClient.storage,
		myUser, make(map[id.Node]user.NodeKeys), &publicKeyRSA,
		privateKeyRSA, nil, nil, myPubKeyCyclic, myPrivKeyCyclic, make([]byte, 1), cmixGrp,
		e2eGrp, "password", regSignature)

	testClient.session = session

	testClient.registerUserE2E(partner, partnerPubKeyCyclic.Bytes())

	// Confirm we can get all types of keys
	km := session.GetKeyStore().GetSendManager(partner)
	if km == nil {
		t.Errorf("KeyStore returned nil when obtaining KeyManager for sending")
	}
	key, action := km.PopKey()
	if key == nil {
		t.Errorf("TransmissionKeys map returned nil")
	} else if key.GetOuterType() != parse.E2E {
		t.Errorf("Key type expected 'E2E', got %s",
			key.GetOuterType())
	} else if action != keyStore.None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	key, action = km.PopRekey()
	if key == nil {
		t.Errorf("TransmissionReKeys map returned nil")
	} else if key.GetOuterType() != parse.Rekey {
		t.Errorf("Key type expected 'Rekey', got %s",
			key.GetOuterType())
	} else if action != keyStore.None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	// Generate one reception key of each type to test
	// fingerprint map
	baseKey, _ := diffieHellman.CreateDHSessionKey(partnerPubKeyCyclic, myPrivKeyCyclic, e2eGrp)
	recvKeys := e2e.DeriveKeys(e2eGrp, baseKey, partner, uint(1))
	recvReKeys := e2e.DeriveEmergencyKeys(e2eGrp, baseKey, partner, uint(1))

	h, _ := hash.NewCMixHash()
	h.Write(recvKeys[0].Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	key = session.GetKeyStore().GetRecvKey(fp)
	if key == nil {
		t.Errorf("ReceptionKeys map returned nil for Key")
	} else if key.GetOuterType() != parse.E2E {
		t.Errorf("Key type expected 'E2E', got %s",
			key.GetOuterType())
	}

	h.Reset()
	h.Write(recvReKeys[0].Bytes())
	copy(fp[:], h.Sum(nil))

	key = session.GetKeyStore().GetRecvKey(fp)
	if key == nil {
		t.Errorf("ReceptionKeys map returned nil for ReKey")
	} else if key.GetOuterType() != parse.Rekey {
		t.Errorf("Key type expected 'Rekey', got %s",
			key.GetOuterType())
	}
	disconnectServers()
}

// Test all keys created with registerUserE2E match what is expected
func TestRegisterUserE2E_CheckAllKeys(t *testing.T) {
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	cmixGrp, e2eGrp := getGroups()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)

	rng := csprng.NewSystemRNG()
	myPrivKeyCyclic := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())
	myPubKeyCyclic := e2eGrp.ExpG(myPrivKeyCyclic, e2eGrp.NewMaxInt())

	partnerPrivKeyCyclic := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())
	partnerPubKeyCyclic := e2eGrp.ExpG(partnerPrivKeyCyclic, e2eGrp.NewMaxInt())

	privateKeyRSA, _ := rsa.GenerateKey(rng, TestKeySize)
	publicKeyRSA := rsa.PublicKey{PublicKey: privateKeyRSA.PublicKey}

	regSignature := make([]byte, 8)

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(testClient.storage,
		myUser, make(map[id.Node]user.NodeKeys), &publicKeyRSA,
		privateKeyRSA, nil, nil, myPubKeyCyclic, myPrivKeyCyclic, make([]byte, 1), cmixGrp,
		e2eGrp, "password", regSignature)

	testClient.session = session

	testClient.registerUserE2E(partner, partnerPubKeyCyclic.Bytes())

	// Generate all keys and confirm they all match
	keyParams := testClient.GetKeyParams()
	baseKey, _ := diffieHellman.CreateDHSessionKey(partnerPubKeyCyclic, myPrivKeyCyclic, e2eGrp)
	keyTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		keyParams.MinKeys, keyParams.MaxKeys, keyParams.TTLParams)

	sendKeys := e2e.DeriveKeys(e2eGrp, baseKey, userID, uint(numKeys))
	sendReKeys := e2e.DeriveEmergencyKeys(e2eGrp, baseKey,
		userID, uint(keyParams.NumRekeys))
	recvKeys := e2e.DeriveKeys(e2eGrp, baseKey, partner, uint(numKeys))
	recvReKeys := e2e.DeriveEmergencyKeys(e2eGrp, baseKey,
		partner, uint(keyParams.NumRekeys))

	// Confirm all keys
	km := session.GetKeyStore().GetSendManager(partner)
	if km == nil {
		t.Errorf("KeyStore returned nil when obtaining KeyManager for sending")
	}
	for i := 0; i < int(numKeys); i++ {
		key, action := km.PopKey()
		if key == nil {
			t.Errorf("TransmissionKeys map returned nil")
		} else if key.GetOuterType() != parse.E2E {
			t.Errorf("Key type expected 'E2E', got %s",
				key.GetOuterType())
		}

		if i < int(keyTTL-1) {
			if action != keyStore.None {
				t.Errorf("Expected 'None' action, got %s instead",
					action)
			}
		} else {
			if action != keyStore.Rekey {
				t.Errorf("Expected 'Rekey' action, got %s instead",
					action)
			}
		}

		if key.GetKey().Cmp(sendKeys[int(numKeys)-1-i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				sendKeys[int(numKeys)-1-i].Text(10),
				key.GetKey().Text(10))
		}
	}

	for i := 0; i < int(keyParams.NumRekeys); i++ {
		key, action := km.PopRekey()
		if key == nil {
			t.Errorf("TransmissionReKeys map returned nil")
		} else if key.GetOuterType() != parse.Rekey {
			t.Errorf("Key type expected 'Rekey', got %s",
				key.GetOuterType())
		}

		if i < int(keyParams.NumRekeys-1) {
			if action != keyStore.None {
				t.Errorf("Expected 'None' action, got %s instead",
					action)
			}
		} else {
			if action != keyStore.Purge {
				t.Errorf("Expected 'Purge' action, got %s instead",
					action)
			}
		}

		if key.GetKey().Cmp(sendReKeys[int(keyParams.NumRekeys)-1-i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				sendReKeys[int(keyParams.NumRekeys)-1-i].Text(10),
				key.GetKey().Text(10))
		}
	}

	h, _ := hash.NewCMixHash()
	fp := format.Fingerprint{}

	for i := 0; i < int(numKeys); i++ {
		h.Reset()
		h.Write(recvKeys[i].Bytes())
		copy(fp[:], h.Sum(nil))
		key := session.GetKeyStore().GetRecvKey(fp)
		if key == nil {
			t.Errorf("ReceptionKeys map returned nil for Key")
		} else if key.GetOuterType() != parse.E2E {
			t.Errorf("Key type expected 'E2E', got %s",
				key.GetOuterType())
		}

		if key.GetKey().Cmp(recvKeys[i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				recvKeys[i].Text(10),
				key.GetKey().Text(10))
		}
	}

	for i := 0; i < int(keyParams.NumRekeys); i++ {
		h.Reset()
		h.Write(recvReKeys[i].Bytes())
		copy(fp[:], h.Sum(nil))
		key := session.GetKeyStore().GetRecvKey(fp)
		if key == nil {
			t.Errorf("ReceptionKeys map returned nil for Rekey")
		} else if key.GetOuterType() != parse.Rekey {
			t.Errorf("Key type expected 'Rekey', got %s",
				key.GetOuterType())
		}

		if key.GetKey().Cmp(recvReKeys[i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				recvReKeys[i].Text(10),
				key.GetKey().Text(10))
		}
	}
	disconnectServers()
}

// Test happy path for precannedRegister
func TestClient_precannedRegister(t *testing.T) {
	//Start client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)

	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}

	nk := make(map[id.Node]user.NodeKeys)

	_, _, nk, err = testClient.precannedRegister("UAV6IWD6", "tony_johns", nk)
	if err != nil {
		t.Errorf("Error during precannedRegister: %+v", err)
	}

	//Disconnect and shutdown servers
	disconnectServers()
}

// Test happy path for sendRegistrationMessage
func TestClient_sendRegistrationMessage(t *testing.T) {

	//Start client
	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}

	rng := csprng.NewSystemRNG()
	privateKeyRSA, _ := rsa.GenerateKey(rng, TestKeySize)
	publicKeyRSA := rsa.PublicKey{PublicKey: privateKeyRSA.PublicKey}

	_, err = testClient.sendRegistrationMessage("UAV6IWD6", &publicKeyRSA)
	if err != nil {
		t.Errorf("Error during sendRegistrationMessage: %+v", err)
	}

	//Disconnect and shutdown servers
	disconnectServers()
}

// Test happy path for requestNonce
func TestClient_requestNonce(t *testing.T) {
	cmixGrp, _ := getGroups()
	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewMaxInt())
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewMaxInt())
	rng := csprng.NewSystemRNG()
	privateKeyRSA, _ := rsa.GenerateKey(rng, TestKeySize)
	publicKeyRSA := rsa.PublicKey{PublicKey: privateKeyRSA.PublicKey}

	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}

	salt := make([]byte, 256)
	_, err = csprng.NewSystemRNG().Read(salt)
	if err != nil {
		t.Errorf("Unable to generate salt! %s", err)
	}

	gwID := id.NewNodeFromBytes(testClient.ndf.Nodes[0].ID).NewGateway()
	_, _, err = testClient.requestNonce(salt, []byte("test"), publicKeyDH, &publicKeyRSA, privateKeyRSA, gwID)
	if err != nil {
		t.Errorf("Error during requestNonce: %+v", err)
	}

	disconnectServers()
}

// Test happy path for confirmNonce
func TestClient_confirmNonce(t *testing.T) {

	testClient, err := NewClient(&globals.RamStorage{}, "", "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Error(err)
	}
	rng := csprng.NewSystemRNG()
	privateKeyRSA, _ := rsa.GenerateKey(rng, TestKeySize)
	gwID := id.NewNodeFromBytes(testClient.ndf.Nodes[0].ID).NewGateway()
	err = testClient.confirmNonce([]byte("user"), []byte("test"), privateKeyRSA, gwID)
	if err != nil {
		t.Errorf("Error during confirmNonce: %+v", err)
	}
	//Disconnect and shutdown servers
	disconnectServers()
}

func getGroups() (*cyclic.Group, *cyclic.Group) {

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48"+
			"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F"+
			"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5"+
			"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2"+
			"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41"+
			"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE"+
			"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15"+
			"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
		large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613"+
			"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4"+
			"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472"+
			"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5"+
			"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA"+
			"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71"+
			"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0"+
			"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16))

	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))

	return cmixGrp, e2eGrp

}

// Test happy path for client.GetSession
func TestClient_GetSession(t *testing.T) {

	//Start client
	testClient, _ := NewClient(&globals.RamStorage{}, "", "", def)

	testClient.session = &user.SessionObj{}

	if !reflect.DeepEqual(testClient.GetSession(), testClient.session) {
		t.Error("Received session not the same as the real session")
	}

}

// Test happy path for client.GetCommManager
func TestClient_GetCommManager(t *testing.T) {

	//Start client
	testClient, _ := NewClient(&globals.RamStorage{}, "", "", def)

	testClient.commManager = &io.ReceptionManager{}

	if !reflect.DeepEqual(testClient.GetReceptionManager(), testClient.commManager) {
		t.Error("Received session not the same as the real session")
	}

}
