package rekey

import (
	"bytes"
	"fmt"
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
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
	"time"
)

var ListenCh chan []byte

type dummyMessaging struct {
	listener chan []byte
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(sess user.Session,
	topology *circuit.Circuit,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	d.listener <- message
	return nil
}

// SendMessage without partitions to the server
func (d *dummyMessaging) SendMessageNoPartition(sess user.Session,
	topology *circuit.Circuit,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	d.listener <- message
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(session user.Session,
	delay time.Duration) {
}

func TestMain(m *testing.M) {

	grp, e2eGrp := getGroups()
	user.InitUserRegistry(grp)
	params := signature.CustomDSAParams(
		grp.GetP(),
		grp.GetG(),
		grp.GetQ())
	rng := csprng.NewSystemRNG()
	u := &user.User{
		User: id.NewUserFromUints(&[4]uint64{0, 0, 0, 18}),
		Nick: "Bernie",
	}
	myPrivKey := params.PrivateKeyGen(rng)
	myPrivKeyCyclic := grp.NewIntFromLargeInt(myPrivKey.GetKey())
	myPubKey := myPrivKey.PublicKeyGen()
	partnerID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 12})
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()
	partnerPubKeyCyclic := grp.NewIntFromLargeInt(partnerPubKey.GetKey())

	session := user.NewSession(&globals.RamStorage{},
		u, nil, myPubKey, myPrivKey, grp, e2eGrp)
	ListenCh = make(chan []byte, 100)
	fakeComm := &dummyMessaging{
		listener: ListenCh,
	}
	InitRekey(session, fakeComm, circuit.New([]*id.Node{id.NewNodeFromBytes(make([]byte, id.NodeIdLen))}))

	// Create E2E relationship with partner
	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKeyCyclic,
		myPrivKeyCyclic,
		grp)

	// Generate key TTL and number of keys
	keyParams := session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		keyParams.MinKeys, keyParams.MaxKeys, keyParams.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, myPrivKeyCyclic,
		partnerPubKeyCyclic, partnerID, true,
		numKeys, keysTTL, keyParams.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, u.User, session.GetKeyStore())

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, myPrivKeyCyclic,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, keyParams.NumRekeys)

	// Generate Receive Keys
	km.GenerateKeys(grp, u.User, session.GetKeyStore())

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: myPrivKeyCyclic,
		CurrPubKey:  partnerPubKeyCyclic,
	}

	session.GetRekeyManager().AddKeys(partnerID, keys)

	os.Exit(m.Run())
}

// Test RekeyTrigger
func TestRekeyTrigger(t *testing.T) {
	partnerID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 12})
	km := session.GetKeyStore().GetRecvManager(partnerID)
	partnerPubKey := km.GetPubKey()
	// Test receiving a RekeyTrigger message
	msg := &parse.Message{
		Sender: session.GetCurrentUser().User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekeytrigger processing
	if rekeyTriggerList.err != nil {
		t.Errorf("RekeyTrigger returned error: %v", rekeyTriggerList.err.Error())
	}
	// Get new PubKey from Rekey message and confirm value matches
	// with PubKey created from privKey in Rekey Context
	value := <-ListenCh
	grp := session.GetCmixGroup()
	actualPubKey := grp.NewIntFromBytes(value)
	privKey := session.GetRekeyManager().GetCtx(partnerID).PrivKey
	expectedPubKey := grp.NewInt(1)
	grp.ExpG(privKey, expectedPubKey)

	if expectedPubKey.Cmp(actualPubKey) != 0 {
		t.Errorf("RekeyTrigger publicKey mismatch, expected %s,"+
			" got %s", expectedPubKey.Text(10),
			actualPubKey.Text(10))
	}

	// Check that trying to send another rekeyTrigger message returns an error
	msg = &parse.Message{
		Sender: session.GetCurrentUser().User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	session.GetSwitchboard().Speak(msg)

	// Check that error occurred in rekeytrigger for repeated message
	if rekeyTriggerList.err == nil {
		t.Errorf("RekeyTrigger should have returned error")
	}
}

// Test RekeyConfirm
func TestRekeyConfirm(t *testing.T) {
	partnerID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 12})
	rekeyCtx := session.GetRekeyManager().GetCtx(partnerID)
	baseKey := rekeyCtx.BaseKey
	// Test receiving a RekeyConfirm message with wrong H(baseKey)
	msg := &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body:        baseKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check that error occurred in RekeyConfirm when hash is wrong
	if rekeyConfirmList.err == nil {
		t.Errorf("RekeyConfirm should have returned error")
	}

	// Test with correct hash
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	msg = &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body:        h.Sum(nil),
		},
		InferredType: parse.None,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekeyConfirm processing
	if rekeyConfirmList.err != nil {
		t.Errorf("RekeyConfirm returned error: %v", rekeyConfirmList.err.Error())
	}

	// Confirm that user Private key in Send Key Manager
	// differs from the one stored in session
	if session.GetPrivateKey().GetKey().Cmp(
		session.GetKeyStore().GetSendManager(partnerID).
			GetPrivKey().GetLargeInt()) == 0 {
		t.Errorf("PrivateKey remained unchanged after Outgoing Rekey!")
	}

	// Check that trying to send another rekeyConfirm message causes an error
	// since no Rekey is in progress anymore
	msg = &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body:        h.Sum(nil),
		},
		InferredType: parse.None,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check that error occurred in RekeyConfirm for repeated message
	if rekeyConfirmList.err == nil {
		t.Errorf("RekeyConfirm should have returned error")
	}
}

// Test Rekey
func TestRekey(t *testing.T) {
	partnerID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 12})
	km := session.GetKeyStore().GetSendManager(partnerID)
	// Generate new partner public key
	grp, _ := getGroups()
	params := signature.CustomDSAParams(
		grp.GetP(),
		grp.GetG(),
		grp.GetQ())
	rng := csprng.NewSystemRNG()
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()
	// Test receiving a Rekey message
	msg := &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NO_TYPE),
			Body:        partnerPubKey.GetKey().Bytes(),
		},
		InferredType: parse.Rekey,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekey processing
	if rekeyList.err != nil {
		t.Errorf("Rekey returned error: %v", rekeyList.err.Error())
	}
	// Confirm hash of baseKey matches expected
	var value []byte

	cont := true

	for cont {
		select {
		case value = <-ListenCh:
			fmt.Println("aaa")
		default:
			cont = false
		}

	}

	// Get hash as last 32 bytes of message bytes
	actual := value[len(value)-32:]
	km = session.GetKeyStore().GetRecvManager(partnerID)
	baseKey := grp.NewInt(1)
	grp.Exp(km.GetPubKey(), km.GetPrivKey(), baseKey)
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	expected := h.Sum(nil)

	if !bytes.Equal(expected, actual) {
		t.Errorf("Rekey hash(baseKey) mismatch, expected %x,"+
			" got %x", expected, actual)
	}

	// Confirm that keys rotated properly in RekeyManager
	rkm := session.GetRekeyManager()
	keys := rkm.GetKeys(partnerID)

	if keys.CurrPrivKey.GetLargeInt().
		Cmp(session.GetPrivateKey().GetKey()) == 0 {
		t.Errorf("Own PrivateKey didn't update properly after both parties rekeys")
	}

	if keys.CurrPubKey.GetLargeInt().
		Cmp(partnerPubKey.GetKey()) != 0 {
		t.Errorf("Partner PublicKey didn't update properly after both parties rekeys")
	}
}

// Test Rekey errors
func TestRekey_Errors(t *testing.T) {
	partnerID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 12})
	km := session.GetKeyStore().GetRecvManager(partnerID)
	partnerPubKey := km.GetPubKey()
	// Delete RekeyKeys so that RekeyTrigger and rekey error out
	session.GetRekeyManager().DeleteKeys(partnerID)
	// Test receiving a RekeyTrigger message
	msg := &parse.Message{
		Sender: session.GetCurrentUser().User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	session.GetSwitchboard().Speak(msg)

	// Check error occurred on RekeyTrigger
	if rekeyTriggerList.err == nil {
		t.Errorf("RekeyTrigger should have returned error")
	}

	// Test receiving a Rekey message
	msg = &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NO_TYPE),
			Body:        []byte{},
		},
		InferredType: parse.Rekey,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check error occurred on Rekey
	if rekeyList.err == nil {
		t.Errorf("Rekey should have returned error")
	}
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
			"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16),
		large.NewIntFromString("F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F", 16))

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
