////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bots

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"strings"
	"testing"
	"time"
)

var ListenCh chan *format.Message

type dummyMessaging struct {
	listener    chan *format.Message
	switchboard *switchboard.Switchboard
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	jww.INFO.Printf("Sending: %s", string(message))
	return nil
}

// SendMessage without partitions to the server
func (d *dummyMessaging) SendMessageNoPartition(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	jww.INFO.Printf("Sending: %s", string(message))
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(session user.Session,
	delay time.Duration, transmissionHost *connect.Host, callback func(error)) {
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) GetSwitchboard() *switchboard.Switchboard {
	return d.switchboard
}

var pubKeyBits string
var keyFingerprint string
var pubKey []byte

func TestMain(m *testing.M) {
	u := &storage.User{
		User:     new(id.ID),
		Username: "Bernie",
	}
	binary.BigEndian.PutUint64(u.User[:], 18)
	u.User.SetType(id.User)

	cmixGrp, e2eGrp := getGroups()

	fakeSession := user.NewSession(&globals.RamStorage{}, "password")
	fakeSession2 := storage.InitTestingSession(m)
	fakeSession2.CommitUserData(&storage.UserData{
		ThisUser: &storage.User{
			User:     u.User,
			Username: u.Username,
		},
		CmixGrp: cmixGrp,
		E2EGrp:  e2eGrp,
	})
	fakeComm := &dummyMessaging{
		listener:    ListenCh,
		switchboard: switchboard.NewSwitchboard(),
	}
	h := connect.Host{}
	nodeID := new(id.ID)
	nodeID.SetType(id.Node)
	topology := connect.NewCircuit([]*id.ID{nodeID})

	InitBots(fakeSession, *fakeSession2, fakeComm, topology, &h)

	// Make the reception channels buffered for this test
	// which overwrites the channels registered in InitBots
	pushKeyResponseListener = make(channelResponseListener, 100)
	getKeyResponseListener = make(channelResponseListener, 100)
	registerResponseListener = make(channelResponseListener, 100)
	searchResponseListener = make(channelResponseListener, 100)

	pubKeyBits = "S8KXBczy0jins9uS4LgBPt0bkFl8t00MnZmExQ6GcOcu8O7DKgAsNzLU7a+gMTbIsS995IL/kuFF8wcBaQJBY23095PMSQ/nMuetzhk9HdXxrGIiKBo3C/n4SClpq4H+PoF9XziEVKua8JxGM2o83KiCK3tNUpaZbAAElkjueY7wuD96h4oaA+WV5Nh87cnIZ+fAG0uLve2LSHZ0FBZb3glOpNAOv7PFWkvN2BO37ztOQCXTJe72Y5ReoYn7nWVNxGUh0ilal+BRuJt1GZ7whOGDRE0IXfURIoK2yjyAnyZJWWMhfGsL5S6iL4aXUs03mc8BHKRq3HRjvTE10l3YFA=="
	pubKey, _ = base64.StdEncoding.DecodeString(pubKeyBits)

	keyFingerprint = fingerprint(pubKey)

	os.Exit(m.Run())
}

// TestRegister smoke tests the registration functionality.
func TestRegister(t *testing.T) {
	// Send response messages from fake UDB in advance
	pushKeyResponseListener <- fmt.Sprintf("PUSHKEY COMPLETE %s", keyFingerprint)
	registerResponseListener <- "REGISTRATION COMPLETE"

	dummyRegState := func(int) {
		return
	}
	err := Register("EMAIL", "rick@elixxir.io", pubKey, dummyRegState, 30*time.Second)
	if err != nil {
		t.Errorf("Registration failure: %s", err.Error())
	}
	// Send response messages from fake UDB in advance
	pushKeyResponseListener <- fmt.Sprintf("PUSHKEY Failed: Could not push key %s becasue key already exists", keyFingerprint)
	err = Register("EMAIL", "rick@elixxir.io", pubKey, dummyRegState, 30*time.Second)
	if err == nil {
		t.Errorf("Registration duplicate did not fail")
	}
}

// TestSearch smoke tests the search function
func TestSearch(t *testing.T) {
	publicKeyString := base64.StdEncoding.EncodeToString(pubKey)

	// Send response messages from fake UDB in advance
	searchResponseListener <- "blah@elixxir.io FOUND UR69db14ZyicpZVqJ1HFC5rk9UZ8817aV6+VHmrJpGc= AAAAAAAAABoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD 8oKh7TYG4KxQcBAymoXPBHSD/uga9pX3Mn/jKhvcD8M="
	getKeyResponseListener <- fmt.Sprintf("GETKEY %s %s", keyFingerprint,
		publicKeyString)

	dummySearchState := func(int) {
		return
	}

	searchedUser, err := Search("EMAIL", "blah@elixxir.io",
		dummySearchState, 30*time.Second)
	if err != nil {
		t.Fatalf("Error on Search: %s", err.Error())
	}
	if !searchedUser.Id.Cmp(id.NewIdFromUInt(26, id.User, t)) {
		t.Errorf("Search did not return user ID 26! returned %s", searchedUser)
	}
	//Test the timeout capabilities
	searchedUser, err = Search("EMAIL", "blah@elixxir.io", dummySearchState, 1*time.Millisecond)
	if err == nil {
		t.Fatal("udb search timeout should have caused error")
	}
	if strings.Compare(err.Error(), "UDB search timeout exceeded on user lookup") != 0 {
		t.Errorf("error: %v", err)
	}
}

// messages using switchboard
// Test LookupNick function
func TestNicknameFunctions(t *testing.T) {
	// Test receiving a nickname request
	userData, _ := sessionV2.GetUserData()
	curUser := userData.ThisUser.User

	msg := &parse.Message{
		Sender: curUser,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NICKNAME_REQUEST),
			Body:        []byte{},
		},
		InferredType: parse.Unencrypted,
		Receiver:     curUser,
	}
	comms.GetSwitchboard().Speak(msg)

	// Test nickname lookup

	// send response to switchboard
	msg = &parse.Message{
		Sender: curUser,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NICKNAME_RESPONSE),
			Body:        []byte(userData.ThisUser.Username),
		},
		InferredType: parse.Unencrypted,
		Receiver:     curUser,
	}
	comms.GetSwitchboard().Speak(msg)
	// AFter sending the message, perform the lookup to read it
	nick, err := LookupNick(curUser)
	if err != nil {
		t.Errorf("Error on LookupNick: %s", err.Error())
	}
	if nick != userData.ThisUser.Username {
		t.Errorf("LookupNick returned wrong value. Expected %s,"+
			" Got %s", userData.ThisUser.Username, nick)
	}
}

type errorMessaging struct{}

// SendMessage that just errors out
func (e *errorMessaging) SendMessage(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	return errors.New("This is an error")
}

// SendMessage no partition that just errors out
func (e *errorMessaging) SendMessageNoPartition(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	return errors.New("This is an error")
}

// MessageReceiver thread to get new messages
func (e *errorMessaging) MessageReceiver(session user.Session,
	delay time.Duration, transmissionHost *connect.Host, callback func(error)) {
}

func (e *errorMessaging) GetSwitchboard() *switchboard.Switchboard {
	return nil
}

// Test LookupNick returns error on sending problem
func TestLookupNick_error(t *testing.T) {
	userData, _ := sessionV2.GetUserData()
	// Replace comms with errorMessaging
	comms = &errorMessaging{}
	_, err := LookupNick(userData.ThisUser.User)
	if err == nil {
		t.Errorf("LookupNick should have returned an error")
	}
}

func getGroups() (cmixGrp *cyclic.Group, e2eGrp *cyclic.Group) {

	cmixPrime := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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

	e2ePrime := "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
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
		"847AEF49F66E43873"

	cmixGrp = cyclic.NewGroup(large.NewIntFromString(cmixPrime, 16),
		large.NewIntFromUInt(2))

	e2eGrp = cyclic.NewGroup(large.NewIntFromString(e2ePrime, 16),
		large.NewIntFromUInt(2))

	return
}
