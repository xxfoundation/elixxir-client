////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"os"
	"testing"
	"time"
)

var gwAddress = "localhost:8080"

var Session user.SessionObj
var GatewayData TestInterface

func TestMain(m *testing.M) {
	rand.Seed(time.Now().Unix())
	gwAddress = fmt.Sprintf("localhost:%d", rand.Intn(1000)+5001)
	io.SendAddress = gwAddress
	io.ReceiveAddress = gwAddress
	GatewayData = TestInterface{
		LastReceivedMessage: pb.CmixMessage{},
	}
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(m.Run())
}

// Make sure InitClient registers storage.
func TestInitClient(t *testing.T) {
	globals.LocalStorage = nil
	err := InitClient(nil, "")
	if err != nil {
		t.Errorf("InitClient failed on valid input: %v", err)
	}
	if globals.LocalStorage == nil {
		t.Errorf("InitClient did not register storage.")
	}
	globals.LocalStorage = nil
}

func TestRegister(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "UAV6IWD6"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	q := large.NewInt(3)
	grp := cyclic.NewGroup(p, g, q)
	regRes, err := Register(registrationCode, gwAddress, 1, false, &grp)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
	globals.LocalStorage = nil
}

func TestRegisterBadNumNodes(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "UAV6IWD6"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	p := large.NewInt(int64(1))
	g := large.NewInt(int64(2))
	q := large.NewInt(int64(3))
	grp := cyclic.NewGroup(p, g, q)
	_, err = Register(registrationCode, gwAddress, 0, false, &grp)
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestRegisterBadHUID(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "OIF3OJ6I"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	p := large.NewInt(int64(1))
	g := large.NewInt(int64(2))
	q := large.NewInt(int64(3))
	grp := cyclic.NewGroup(p, g, q)
	_, err = Register(registrationCode, gwAddress, 1, false, &grp)
	if err == nil {
		t.Error("Registration worked with bad registration code!")
	}
	globals.LocalStorage = nil
}

func TestRegisterDeletedUser(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "UAV6IWD6"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	p := large.NewInt(int64(1))
	g := large.NewInt(int64(2))
	q := large.NewInt(int64(3))
	grp := cyclic.NewGroup(p, g, q)
	tempUser, _ := user.Users.GetUser(id.NewUserFromUint(5, t))
	user.Users.DeleteUser(id.NewUserFromUint(5, t))
	_, err = Register(registrationCode, gwAddress, 1, false, &grp)
	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s",
			err.Error())
	}
	user.Users.UpsertUser(tempUser)
	globals.LocalStorage = nil
}

func SetNulKeys() {
	// Set the transmit keys to be 1, so send/receive can work
	// FIXME: Why doesn't crypto panic when these keys are empty?
	keys := user.TheSession.GetKeys()
	grp := user.TheSession.GetGroup()
	for i := range keys {
		keys[i].TransmissionKeys.Base = grp.NewInt(1)
		keys[i].TransmissionKeys.Recursive = grp.NewInt(1)
	}
}

func TestSend(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, &GatewayData, "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	globals.LocalStorage = nil
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	grp := crypto.InitCrypto()
	registrationCode := "UAV6IWD6"
	userID, err := Register(registrationCode, gwAddress, 1, false, grp)
	session, err2 := Login(userID, gwAddress, "")
	SetNulKeys()

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}
	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(session.GetCurrentUser().Nick) == 0 {
		t.Errorf("Invalid login received: %v", session.GetCurrentUser().User)
	}

	// Test send with invalid sender ID
	err = Send(APIMessage{SenderID: id.NewUserFromUint(12, t),
		Payload:     []byte("test"),
		RecipientID: userID})
	if err != nil {
		// TODO: would be nice to catch the sender but we
		// don't have the interface/mocking for that.
		t.Errorf("error on first message send: %v", err)
	}

	// Test send with valid inputs
	err = Send(APIMessage{SenderID: userID, Payload: []byte("test"),
		RecipientID: userID})
	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}
}

func TestLogout(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	err := Logout()
	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}
	err = Logout()
	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}
}
