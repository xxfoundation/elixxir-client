package api

import (
	"crypto/sha256"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
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
	_, err = testClient.GenerateSessionInformation(def, nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6", "", "", "password", &SessionInformation{})
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

	_, err = testClient.GenerateSessionInformation(def, nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// populate a gob in the store
	_, err = testClient.RegisterWithPermissioning(true, "UAV6IWD6", "", "", "password",
		&SessionInformation{})
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

//Error path: Using a reg server that will cause an error
func TestClient_Register_NoUpdatingNDF(t *testing.T) {
	mockRegError := registration.StartRegistrationServer(ErrorDef.Registration.Address, &NDFErrorReg,
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

	mockRegError.Shutdown()
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
