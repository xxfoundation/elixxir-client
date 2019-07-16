////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
	"time"
)

func TestRegistrationGob(t *testing.T) {
	// Get a Client
	testClient, err := NewClient(&globals.RamStorage{}, "", def)
	if err != nil {
		t.Error(err)
	}

	err = testClient.Connect()
	if err != nil {
		t.Error(err)
	}

	// populate a gob in the store
	_, err = testClient.Register(true, "UAV6IWD6", "", "")
	if err != nil {
		t.Error(err)
	}

	// get the gob out of there again
	sessionGob := testClient.storage.Load()
	var sessionBytes bytes.Buffer
	sessionBytes.Write(sessionGob)
	dec := gob.NewDecoder(&sessionBytes)
	Session := user.SessionObj{}
	err = dec.Decode(&Session)
	if err != nil {
		t.Error(err)
	}

	VerifyRegisterGobUser(testClient, t)
	VerifyRegisterGobKeys(testClient, t)
}

func VerifyRegisterGobUser(testClient *Client, t *testing.T) {

	session := testClient.session

	if session.GetCurrentUser().User != id.NewUserFromUint(5, t) {
		t.Errorf("User's ID was %q, expected %v",
			session.GetCurrentUser().User, 5)
	}
}

func VerifyRegisterGobKeys(testClient *Client, t *testing.T) {
	cmixGrp, _ := getGroups()
	h := sha256.New()
	h.Write([]byte(string(20005)))
	expectedTransmissionBaseKey := cmixGrp.NewIntFromBytes(h.Sum(nil))

	session := testClient.session

	if session.GetKeys(testClient.topology)[0].TransmissionKey.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			session.GetKeys(testClient.topology)[0].TransmissionKey.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(40005)))
	expectedReceptionBaseKey := cmixGrp.NewIntFromBytes(h.Sum(nil))
	if session.GetKeys(testClient.topology)[0].ReceptionKey.Cmp(
		expectedReceptionBaseKey) != 0 {
		t.Errorf("Reception base key was %v, expected %v",
			session.GetKeys(testClient.topology)[0].ReceptionKey.Text(16),
			expectedReceptionBaseKey.Text(16))
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
	testClient, err := NewClient(&globals.RamStorage{}, "", def)
	if err != nil {
		t.Error(err)
	}

	cmixGrp, e2eGrp := getGroups()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)
	params := signature.CustomDSAParams(
		cmixGrp.GetP(),
		cmixGrp.GetG(),
		cmixGrp.GetQ())
	rng := csprng.NewSystemRNG()
	myPrivKey := params.PrivateKeyGen(rng)
	myPrivKeyCyclic := cmixGrp.NewIntFromLargeInt(myPrivKey.GetKey())
	myPubKey := myPrivKey.PublicKeyGen()
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()
	partnerPubKeyCyclic := cmixGrp.NewIntFromLargeInt(partnerPubKey.GetKey())

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(testClient.storage,
		myUser, make(map[id.Node]user.NodeKeys), myPubKey, myPrivKey, cmixGrp, e2eGrp)

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
}

// Test all keys created with registerUserE2E match what is expected
func TestRegisterUserE2E_CheckAllKeys(t *testing.T) {
	testClient, err := NewClient(&globals.RamStorage{}, "", def)
	if err != nil {
		t.Error(err)
	}

	cmixGrp, e2eGrp := getGroups()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)
	params := signature.CustomDSAParams(
		cmixGrp.GetP(),
		cmixGrp.GetG(),
		cmixGrp.GetQ())
	rng := csprng.NewSystemRNG()
	myPrivKey := params.PrivateKeyGen(rng)
	myPrivKeyCyclic := cmixGrp.NewIntFromLargeInt(myPrivKey.GetKey())
	myPubKey := myPrivKey.PublicKeyGen()
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()
	partnerPubKeyCyclic := cmixGrp.NewIntFromLargeInt(partnerPubKey.GetKey())

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(testClient.storage,
		myUser, make(map[id.Node]user.NodeKeys), myPubKey,
		myPrivKey, cmixGrp, e2eGrp)

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
}

// FIXME Reinstate tests for the UDB api
//var ListenCh chan *format.Message
//var lastmsg string

//type dummyMessaging struct {
//	listener chan *format.Message
//}

// SendMessage to the server
//func (d *dummyMessaging) SendMessage(recipientID id.User,
//	message string) error {
//	jww.INFO.Printf("Sending: %s", message)
//	lastmsg = message
//	return nil
//}

// Listen for messages from a given sender
//func (d *dummyMessaging) Listen(senderID id.User) chan *format.Message {
//	return d.listener
//}

// StopListening to a given switchboard (closes and deletes)
//func (d *dummyMessaging) StopListening(listenerCh chan *format.Message) {}

// MessageReceiver thread to get new messages
//func (d *dummyMessaging) MessageReceiver(delay time.Duration) {}

//var pubKeyBits []string
//var keyFingerprint string
//var pubKey []byte

// SendMsg puts a fake udb response message on the channel
//func SendMsg(msg string) {
//	m, _ := format.NewMessage(13, 1, msg)
//	ListenCh <- &m[0]
//}

//func TestRegisterPubKeyByteLen(t *testing.T) {
//	ListenCh = make(chan *format.Message, 100)
//	io.Messaging = &dummyMessaging{
//		switchboard: ListenCh,
//	}
//	pubKeyBits = []string{
//		"S8KXBczy0jins9uS4LgBPt0bkFl8t00MnZmExQ6GcOcu8O7DKgAsNz" +
//			"LU7a+gMTbIsS995IL/kuFF8wcBaQJBY23095PMSQ/nMuetzhk9HdXxrGIiKBo3C/n4SClp" +
//			"q4H+PoF9XziEVKua8JxGM2o83KiCK3tNUpaZbAAElkjueY4=",
//		"8Lg/eoeKGgPlleTYfO3JyGfnwBtLi73ti0h2dBQWW94JTqTQDr+z" +
//			"xVpLzdgTt+87TkAl0yXu9mOUXqGJ+51lTcRlIdIpWpfgUbibdRme8IThg0RNCF31ESKCts" +
//			"o8gJ8mSVljIXxrC+Uuoi+Gl1LNN5nPARykatx0Y70xNdJd2BQ=",
//	}
//	pubKey = make([]byte, 256)
//	for i := range pubKeyBits {
//		pubkeyBytes, _ := base64.StdEncoding.DecodeString(pubKeyBits[i])
//		for j := range pubkeyBytes {
//			pubKey[j+i*128] = pubkeyBytes[j]
//		}
//	}
//
//	keyFingerprint = "8oKh7TYG4KxQcBAymoXPBHSD/uga9pX3Mn/jKhvcD8M="
//	//SendMsg("SEARCH blah@privategrity.com NOTFOUND")
//	SendMsg(fmt.Sprintf("GETKEY %s NOTFOUND", keyFingerprint))
//	SendMsg("PUSHKEY ACK NEED 128")
//	SendMsg(fmt.Sprintf("PUSHKEY COMPLETE %s", keyFingerprint))
//	SendMsg("REGISTRATION COMPLETE")
//
//	err := bots.Register("EMAIL", "blah@privategrity.com", pubKey)
//
//	if err != nil {
//		t.Errorf("Unexpected error: %s", err.Error())
//	}
//	if len(lastmsg) != 81 {
//		t.Errorf("Message wrong length: %d v. expected 81", len(lastmsg))
//	}
//}

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
		large.NewIntFromString("2", 16),
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
		large.NewIntFromString("2", 16),
		large.NewIntFromString("2", 16))

	return cmixGrp, e2eGrp

}
