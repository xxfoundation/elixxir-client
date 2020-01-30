////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"crypto/sha256"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the Registry interface
func TestUserSession(t *testing.T) {

	test := 11
	pass := 0

	u := new(User)
	// This is 65 so you can see the letter A in the gob if you need to make
	// sure that the gob contains the user ID
	UID := uint64(65)

	u.User = id.NewUserFromUint(UID, t)
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	nodeID := id.NewNodeFromUInt(1, t)

	topology := connect.NewCircuit([]*id.Node{nodeID})

	// Storage
	storage := &globals.RamStorage{}

	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}

	cmixGrp, e2eGrp := getGroups()

	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))

	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(storage,
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	err := ses.RegisterPermissioningSignature(regSignature)
	if err != nil {
		t.Errorf("failure in setting register up for permissioning: %s",
			err.Error())
	}

	ses.PushNodeKey(nodeID, NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	ses.SetLastMessageID("totally unique ID")

	err = ses.StoreSession()

	if err != nil {
		t.Errorf("Session not stored correctly: %s", err.Error())
	}

	ses.Immolate()

	//TODO: write test which validates the immolation

	ses, err = LoadSession(storage, "password")

	if err != nil {
		t.Errorf("Unable to login with valid user: %v",
			err.Error())
	} else {
		pass++
	}

	if ses.GetLastMessageID() != "totally unique ID" {
		t.Errorf("Last message ID should have been stored " +
			"and loaded")
	} else {
		pass++
	}

	ses.SetLastMessageID("test")

	if ses.GetLastMessageID() != "test" {
		t.Errorf("Last message ID not set correctly with" +
			" SetLastMessageID!")
	} else {
		pass++
	}

	if ses.GetNodeKeys(topology) == nil {
		t.Errorf("Keys not set correctly!")
	} else {

		test += len(ses.GetNodeKeys(topology))

		for i := 0; i < len(ses.GetNodeKeys(topology)); i++ {
			orig := privateKey.PrivateKey
			sesPriv := ses.GetRSAPrivateKey().PrivateKey
			if !reflect.DeepEqual(*ses.GetRSAPublicKey(), publicKey) {
				t.Errorf("Error: Public key not set correctly!")
			} else if sesPriv.E != orig.E {
				t.Errorf("Error: Private key not set correctly E!  \nExpected: %+v\nreceived: %+v",
					orig.E, sesPriv.E)
			} else if sesPriv.D.Cmp(orig.D) != 0 {
				t.Errorf("Error: Private key not set correctly D!  \nExpected: %+v\nreceived: %+v",
					orig.D, sesPriv.D)
			} else if sesPriv.N.Cmp(orig.N) != 0 {
				t.Errorf("Error: Private key not set correctly N!  \nExpected: %+v\nreceived: %+v",
					orig.N, sesPriv.N)
			} else if !reflect.DeepEqual(sesPriv.Primes, orig.Primes) {
				t.Errorf("Error: Private key not set correctly PRIMES!  \nExpected: %+v\nreceived: %+v",
					orig, sesPriv)
			} else if ses.GetNodeKeys(topology)[i].ReceptionKey.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Reception key not set correct!")
			} else if ses.GetNodeKeys(topology)[i].TransmissionKey.Cmp(
				grp.NewInt(2)) != 0 {
				t.Errorf("Transmission key not set correctly!")
			}

			pass++
		}
	}

	//TODO: FIX THIS?
	if ses.GetRSAPrivateKey() == nil {
		t.Errorf("Error: Private Keys not set correctly!")
	} else {
		pass++
	}

	err = ses.UpsertMap("test", 5)

	if err != nil {
		t.Errorf("Could not store in session map interface: %s",
			err.Error())
	}

	element, err := ses.QueryMap("test")

	if err != nil {
		t.Errorf("Could not read element in session map "+
			"interface: %s", err.Error())
	}

	if element.(int) != 5 {
		t.Errorf("Could not read element in session map "+
			"interface: Expected: 5, Recieved: %v", element)
	}

	ses.DeleteMap("test")

	_, err = ses.QueryMap("test")

	if err == nil {
		t.Errorf("Could not delete element in session map " +
			"interface")
	}

	//Logout
	ses.Immolate()

	// Error tests

	// Test nil LocalStorage

	_, err = LoadSession(nil, "password")

	if err == nil {
		t.Errorf("Error did not catch a nil LocalStorage")
	}

	// Test invalid / corrupted LocalStorage
	h := sha256.New()
	h.Write([]byte(string(20000)))
	randBytes := h.Sum(nil)
	storage.SaveA(randBytes)
	storage.SaveB(randBytes)

	defer func() {
		recover()
	}()

	_, err = LoadSession(storage, "password")
	if err == nil {
		t.Errorf("LoadSession should error on bad decrypt!")
	}
}

func TestSessionObj_DeleteContact(t *testing.T) {
	u := new(User)
	// This is 65 so you can see the letter A in the gob if you need to make
	// sure that the gob contains the user ID
	UID := uint64(65)

	u.User = id.NewUserFromUint(UID, t)
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	nodeID := id.NewNodeFromUInt(1, t)

	// Storage
	storage := &globals.RamStorage{}

	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}

	cmixGrp, e2eGrp := getGroups()

	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))

	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(storage,
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	err := ses.RegisterPermissioningSignature(regSignature)
	if err != nil {
		t.Errorf("failure in setting register up for permissioning: %s",
			err.Error())
	}

	ses.PushNodeKey(nodeID, NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	ses.StoreContactByValue("test", id.NewUserFromBytes([]byte("test")), []byte("test"))

	_, err = ses.DeleteContact(id.NewUserFromBytes([]byte("test")))
	if err != nil {
		t.Errorf("Failed to delete contact: %+v", err)
	}
}

func TestGetPubKey(t *testing.T) {
	u := new(User)
	UID := id.NewUserFromUint(1, t)

	u.User = UID
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}

	cmixGrp, e2eGrp := getGroups()

	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))

	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(&globals.RamStorage{},
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	err := ses.RegisterPermissioningSignature(regSignature)
	if err != nil {
		t.Errorf("failure in setting register up for permissioning: %s",
			err.Error())
	}

	ses.PushNodeKey(id.NewNodeFromUInt(1, t), NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	pubKey := *ses.GetRSAPublicKey()
	if !reflect.DeepEqual(pubKey, publicKey) {
		t.Errorf("Public key not returned correctly!")
	}
}

//Tests the isEmpty function before and after StoreSession
func TestSessionObj_StorageIsEmpty(t *testing.T) {
	// Generate all the values needed for a session
	u := new(User)
	// This is 65 so you can see the letter A in the gob if you need to make
	// sure that the gob contains the user ID
	UID := uint64(65)

	u.User = id.NewUserFromUint(UID, t)
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	nodeID := id.NewNodeFromUInt(1, t)
	// Storage
	storage := &globals.RamStorage{}

	//Keys
	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}
	cmixGrp, e2eGrp := getGroups()
	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))
	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(storage,
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	ses.PushNodeKey(nodeID, NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	ses.SetLastMessageID("totally unique ID")

	//Test that the session is empty before the StoreSession call
	if !ses.StorageIsEmpty() {
		t.Errorf("session should be empty before the StoreSession call")
	}
	err := ses.StoreSession()
	if err != nil {
		t.Errorf("Failed to store session: %v", err)
	}

	//Test that the session is not empty after the StoreSession call
	if ses.StorageIsEmpty() {
		t.Errorf("session should not be empty after a StoreSession call")
	}

}

// GetContactByValue happy path
func TestSessionObj_GetContactByValue(t *testing.T) {
	// Generate all the values needed for a session
	u := new(User)
	// This is 65 so you can see the letter A in the gob if you need to make
	// sure that the gob contains the user ID
	UID := uint64(65)

	u.User = id.NewUserFromUint(UID, t)
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	nodeID := id.NewNodeFromUInt(1, t)

	// Storage
	storage := &globals.RamStorage{}

	//Keys
	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}
	cmixGrp, e2eGrp := getGroups()
	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))
	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(storage,
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	err := ses.RegisterPermissioningSignature(regSignature)
	if err != nil {
		t.Errorf("failure in setting register up for permissioning: %s",
			err.Error())
	}

	ses.PushNodeKey(nodeID, NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	ses.StoreContactByValue("value", id.NewUserFromBytes([]byte("test")), []byte("test"))

	observedUser, observedPk := ses.GetContactByValue("value")

	if bytes.Compare([]byte("test"), observedPk) != 0 {
		t.Errorf("Failed to retieve public key using GetContactByValue; "+
			"Expected: %+v\n\tRecieved: %+v", privateKey.PublicKey.N.Bytes(), observedPk)
	}

	if !observedUser.Cmp(id.NewUserFromBytes([]byte("test"))) {
		t.Errorf("Failed to retrieve user using GetContactByValue;"+
			"Expected: %+v\n\tRecieved: %+v", u.User, observedUser)
	}
}

func TestGetPrivKey(t *testing.T) {
	u := new(User)
	UID := id.NewUserFromUint(1, t)

	u.User = UID
	u.Username = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))

	rng := rand.New(rand.NewSource(42))
	privateKey, _ := rsa.GenerateKey(rng, 768)
	publicKey := rsa.PublicKey{PublicKey: privateKey.PublicKey}

	cmixGrp, e2eGrp := getGroups()

	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewInt(1))
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewInt(1))

	privateKeyDHE2E := e2eGrp.RandomCoprime(e2eGrp.NewInt(1))
	publicKeyDHE2E := e2eGrp.ExpG(privateKeyDHE2E, e2eGrp.NewInt(1))

	ses := NewSession(&globals.RamStorage{},
		u, &publicKey, privateKey, publicKeyDH, privateKeyDH,
		publicKeyDHE2E, privateKeyDHE2E, make([]byte, 1), cmixGrp, e2eGrp,
		"password")

	regSignature := make([]byte, 768)
	rng.Read(regSignature)

	err := ses.RegisterPermissioningSignature(regSignature)
	if err != nil {
		t.Errorf("failure in setting register up for permissioning: %s",
			err.Error())
	}

	ses.PushNodeKey(id.NewNodeFromUInt(1, t), NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	})

	privKey := ses.GetRSAPrivateKey()
	if !reflect.DeepEqual(*privKey, *privateKey) {
		t.Errorf("Private key is not returned correctly!")
	}
}

func TestBruntString(t *testing.T) {
	// Generate a new user and record the pointer to the nick
	u := new(User)
	u.Username = "Mario"
	preBurnPointer := &u.Username

	// Burn the string and record the pointer to the nick
	u.Username = burntString(len(u.Username))
	postBurnPointer := &u.Username

	// Check the nick is not the same as before
	if u.Username == "Mario" {
		t.Errorf("String was not burnt")
	}

	// Check the pointer is the same (otherwise it wasn't overwritten)
	if preBurnPointer != postBurnPointer {
		t.Errorf("Pointer values are not the same")
	}
}

func getGroups() (*cyclic.Group, *cyclic.Group) {

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF", 16),
		large.NewIntFromString("2", 16))

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

// Tests that AppendGarbledMessage properly appends an array of messages by
// testing that the final buffer matches the values appended.
func TestSessionObj_AppendGarbledMessage(t *testing.T) {
	session := NewSession(nil, nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil, nil, "")
	msgs := GenerateTestMessages(10)

	session.AppendGarbledMessage(msgs...)

	if !reflect.DeepEqual(msgs, session.(*SessionObj).garbledMessages) {
		t.Errorf("AppendGarbledMessage() did not append the correct values"+
			"\n\texpected: %v\n\trecieved: %v",
			msgs, session.(*SessionObj).garbledMessages)
	}
}

// Tests that PopGarbledMessages returns the correct data and that the buffer
// is cleared.
func TestSessionObj_PopGarbledMessages(t *testing.T) {
	session := NewSession(nil, nil, nil, nil,
		nil, nil, nil,
		nil, nil, nil, nil, "")
	msgs := GenerateTestMessages(10)

	session.(*SessionObj).garbledMessages = msgs

	poppedMsgs := session.PopGarbledMessages()

	if !reflect.DeepEqual(msgs, poppedMsgs) {
		t.Errorf("PopGarbledMessages() did not pop the correct values"+
			"\n\texpected: %v\n\trecieved: %v",
			msgs, poppedMsgs)
	}

	if !reflect.DeepEqual([]*format.Message{}, session.(*SessionObj).garbledMessages) {
		t.Errorf("PopGarbledMessages() did not remove the values from the buffer"+
			"\n\texpected: %#v\n\trecieved: %#v",
			[]*format.Message{}, session.(*SessionObj).garbledMessages)
	}

}

// Tests ConvertSessionV1toV2() by creating an empty session object and setting
// the RegState to the version 1, running it through the function, and testing
// that RegState has values that match version 2.
func TestSessionObj_ConvertSessionV1toV2(t *testing.T) {
	ses := SessionObj{}
	number := uint32(0)
	ses.RegState = &number

	ConvertSessionV1toV2(&ses)

	if *ses.RegState != 0 {
		t.Errorf("ConvertSessionV1toV2() did not properly convert the "+
			"session object's RegState\n\texpected: %v\n\treceived: %v",
			0, *ses.RegState)
	}

	number = uint32(1)
	ses.RegState = &number

	ConvertSessionV1toV2(&ses)

	if *ses.RegState != 2000 {
		t.Errorf("ConvertSessionV1toV2() did not properly convert the "+
			"session object's RegState\n\texpected: %v\n\treceived: %v",
			2000, *ses.RegState)
	}

	number = uint32(2)
	ses.RegState = &number

	ConvertSessionV1toV2(&ses)

	if *ses.RegState != 3000 {
		t.Errorf("ConvertSessionV1toV2() did not properly convert the "+
			"session object's RegState\n\texpected: %v\n\treceived: %v",
			3000, *ses.RegState)
	}
}

func GenerateTestMessages(size int) []*format.Message {
	msgs := make([]*format.Message, size)

	for i := 0; i < size; i++ {
		msgs[i] = format.NewMessage()
		payloadBytes := make([]byte, format.PayloadLen)
		payloadBytes[0] = byte(i)
		msgs[i].SetPayloadA(payloadBytes)
		msgs[i].SetPayloadB(payloadBytes)
	}

	return msgs
}
