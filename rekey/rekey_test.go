package rekey

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/client/userRegistry"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
	"time"
)

var ListenCh chan []byte

type dummyMessaging struct {
	listener    chan []byte
	switchboard *switchboard.Switchboard
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	d.listener <- message
	return nil
}

// SendMessage without partitions to the server
func (d *dummyMessaging) SendMessageNoPartition(sess user.Session,
	topology *connect.Circuit,
	recipientID *id.ID,
	cryptoType parse.CryptoType,
	message []byte, transmissionHost *connect.Host) error {
	d.listener <- message
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(session user.Session,
	delay time.Duration, transmissionHost *connect.Host, callback func(error)) {
}

// GetSwitchboard to access switchboard
func (d *dummyMessaging) GetSwitchboard() *switchboard.Switchboard {
	return d.switchboard
}

func TestMain(m *testing.M) {

	grp, e2eGrp := getGroups()
	userRegistry.InitUserRegistry(grp)
	rng := csprng.NewSystemRNG()
	u := &storage.User{
		User:     new(id.ID),
		Username: "Bernie",
	}
	binary.BigEndian.PutUint64(u.User[:], 18)
	u.User.SetType(id.User)
	myPrivKeyCyclicCMIX := grp.RandomCoprime(grp.NewMaxInt())
	myPubKeyCyclicCMIX := grp.ExpG(myPrivKeyCyclicCMIX, grp.NewInt(1))
	myPrivKeyCyclicE2E := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())
	myPubKeyCyclicE2E := e2eGrp.ExpG(myPrivKeyCyclicE2E, e2eGrp.NewInt(1))
	partnerID := new(id.ID)
	binary.BigEndian.PutUint64(partnerID[:], 12)
	partnerID.SetType(id.User)

	partnerPubKeyCyclic := e2eGrp.RandomCoprime(e2eGrp.NewMaxInt())

	privateKeyRSA, _ := rsa.GenerateKey(rng, 768)
	publicKeyRSA := rsa.PublicKey{PublicKey: privateKeyRSA.PublicKey}

	session := user.NewSession(&globals.RamStorage{}, "password")
	ListenCh = make(chan []byte, 100)
	fakeComm := &dummyMessaging{
		listener:    ListenCh,
		switchboard: switchboard.NewSwitchboard(),
	}

	sessionV2 := storage.InitTestingSession(m)

	userData := &storage.UserData{
		ThisUser:         u,
		RSAPrivateKey:    privateKeyRSA,
		RSAPublicKey:     &publicKeyRSA,
		CMIXDHPrivateKey: myPrivKeyCyclicCMIX,
		CMIXDHPublicKey:  myPubKeyCyclicCMIX,
		E2EDHPrivateKey:  myPrivKeyCyclicE2E,
		E2EDHPublicKey:   myPubKeyCyclicE2E,
		CmixGrp:          grp,
		E2EGrp:           e2eGrp,
		Salt:             make([]byte, 1),
	}
	sessionV2.CommitUserData(userData)

	rekeyChan2 := make(chan struct{}, 50)
	nodeID := new(id.ID)
	nodeID.SetType(id.Node)
	InitRekey(session, *sessionV2, fakeComm, connect.NewCircuit([]*id.ID{nodeID}), nil, rekeyChan2)

	// Create E2E relationship with partner
	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKeyCyclic,
		myPrivKeyCyclicE2E,
		e2eGrp)

	// Generate key TTL and number of keys
	keyParams := session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		keyParams.MinKeys, keyParams.MaxKeys, keyParams.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, myPrivKeyCyclicE2E,
		partnerPubKeyCyclic, partnerID, true,
		numKeys, keysTTL, keyParams.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, u.User)
	session.GetKeyStore().AddSendManager(km)

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, myPrivKeyCyclicE2E,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, keyParams.NumRekeys)

	// Generate Receive Keys
	e2ekeys := km.GenerateKeys(grp, u.User)
	session.GetKeyStore().AddReceiveKeysByFingerprint(e2ekeys)
	session.GetKeyStore().AddRecvManager(km)
	session.GetKeyStore().AddReceiveKeysByFingerprint(e2ekeys)

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: myPrivKeyCyclicE2E,
		CurrPubKey:  partnerPubKeyCyclic,
	}

	session.GetRekeyManager().AddKeys(partnerID, keys)

	os.Exit(m.Run())
}

// Test RekeyTrigger
func TestRekeyTrigger(t *testing.T) {
	partnerID := new(id.ID)
	binary.BigEndian.PutUint64(partnerID[:], 12)
	partnerID.SetType(id.User)
	km := session.GetKeyStore().GetRecvManager(partnerID)
	userData, _ := sessionV2.GetUserData()
	partnerPubKey := km.GetPubKey()
	// Test receiving a RekeyTrigger message
	msg := &parse.Message{
		Sender: userData.ThisUser.User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	comms.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekeytrigger processing
	if rekeyTriggerList.err != nil {
		t.Errorf("RekeyTrigger returned error: %v", rekeyTriggerList.err.Error())
	}
	// Get new PubKey from Rekey message and confirm value matches
	// with PubKey created from privKey in Rekey Context
	value := <-ListenCh
	grpE2E := userData.E2EGrp
	actualPubKey := grpE2E.NewIntFromBytes(value)
	privKey := session.GetRekeyManager().GetCtx(partnerID).PrivKey
	fmt.Println("privKey: ", privKey.Text(16))
	expectedPubKey := grpE2E.NewInt(1)
	grpE2E.ExpG(privKey, expectedPubKey)
	fmt.Println("new pub key: ", value)

	if expectedPubKey.Cmp(actualPubKey) != 0 {
		t.Errorf("RekeyTrigger publicKey mismatch, expected %s,"+
			" got %s", expectedPubKey.Text(16),
			actualPubKey.Text(16))
	}

	// Check that trying to send another rekeyTrigger message returns an error
	msg = &parse.Message{
		Sender: userData.ThisUser.User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	comms.GetSwitchboard().Speak(msg)
	time.Sleep(time.Second)
	// Check that error occurred in rekeytrigger for repeated message
	if rekeyTriggerList.err == nil {
		t.Errorf("RekeyTrigger should have returned error")
	}
}

// Test RekeyConfirm
func TestRekeyConfirm(t *testing.T) {
	partnerID := new(id.ID)
	binary.BigEndian.PutUint64(partnerID[:], 12)
	partnerID.SetType(id.User)
	rekeyCtx := session.GetRekeyManager().GetCtx(partnerID)
	baseKey := rekeyCtx.BaseKey
	userData, _ := sessionV2.GetUserData()
	// Test receiving a RekeyConfirm message with wrong H(baseKey)
	msg := &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body:        baseKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     userData.ThisUser.User,
	}
	comms.GetSwitchboard().Speak(msg)
	time.Sleep(time.Second)
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
		Receiver:     userData.ThisUser.User,
	}
	comms.GetSwitchboard().Speak(msg)
	time.Sleep(time.Second)
	// Check no error occurred in rekeyConfirm processing
	if rekeyConfirmList.err != nil {
		t.Errorf("RekeyConfirm returned error: %v", rekeyConfirmList.err.Error())
	}

	// Confirm that user Private key in Send Key Manager
	// differs from the one stored in session
	if userData.E2EDHPrivateKey.GetLargeInt().Cmp(
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
		Receiver:     userData.ThisUser.User,
	}
	comms.GetSwitchboard().Speak(msg)
	time.Sleep(time.Second)
	// Check that error occurred in RekeyConfirm for repeated message
	if rekeyConfirmList.err == nil {
		t.Errorf("RekeyConfirm should have returned error")
	}
}

// Test Rekey
func TestRekey(t *testing.T) {
	partnerID := new(id.ID)
	binary.BigEndian.PutUint64(partnerID[:], 12)
	partnerID.SetType(id.User)
	km := session.GetKeyStore().GetSendManager(partnerID)
	userData, _ := sessionV2.GetUserData()
	// Generate new partner public key
	_, grp := getGroups()
	privKey := grp.RandomCoprime(grp.NewMaxInt())
	pubKey := grp.ExpG(privKey, grp.NewMaxInt())
	// Test receiving a Rekey message
	msg := &parse.Message{
		Sender: partnerID,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NO_TYPE),
			Body:        pubKey.Bytes(),
		},
		InferredType: parse.Rekey,
		Receiver:     userData.ThisUser.User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekey processing
	if rekeyList.err != nil {
		t.Errorf("Rekey returned error: %v", rekeyList.err.Error())
	}
	// Confirm hash of baseKey matches expected
	value := <-ListenCh
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
	if keys.CurrPubKey.GetLargeInt().Cmp(userData.E2EDHPublicKey.GetLargeInt()) == 0 {
		t.Errorf("Own publicKey didn't update properly after both parties rekeys")

	}
	if keys.CurrPrivKey.GetLargeInt().
		Cmp(userData.E2EDHPrivateKey.GetLargeInt()) == 0 {
		t.Errorf("Own PrivateKey didn't update properly after both parties rekeys")
		t.Errorf("%s\n%s", keys.CurrPrivKey.GetLargeInt().Text(16), userData.E2EDHPrivateKey.GetLargeInt().Text(16))
	}

	if keys.CurrPubKey.GetLargeInt().
		Cmp(pubKey.GetLargeInt()) != 0 {
		t.Errorf("Partner PublicKey didn't update properly after both parties rekeys")
	}
}

// Test Rekey errors
func TestRekey_Errors(t *testing.T) {
	partnerID := new(id.ID)
	binary.BigEndian.PutUint64(partnerID[:], 12)
	partnerID.SetType(id.User)
	km := session.GetKeyStore().GetRecvManager(partnerID)
	partnerPubKey := km.GetPubKey()
	// Delete RekeyKeys so that RekeyTrigger and rekey error out
	userData, _ := sessionV2.GetUserData()
	session.GetRekeyManager().DeleteKeys(partnerID)
	// Test receiving a RekeyTrigger message
	msg := &parse.Message{
		Sender: userData.ThisUser.User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_TRIGGER),
			Body:        partnerPubKey.Bytes(),
		},
		InferredType: parse.None,
		Receiver:     partnerID,
	}
	comms.GetSwitchboard().Speak(msg)

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
		Receiver:     userData.ThisUser.User,
	}
	comms.GetSwitchboard().Speak(msg)
	time.Sleep(time.Second)
	// Check error occurred on Rekey
	if rekeyList.err == nil {
		t.Errorf("Rekey should have returned error")
	}
}

func getGroups() (*cyclic.Group, *cyclic.Group) {

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("F6FAC7E480EE519354C058BF856AEBDC43AD60141BAD5573910476D030A869979A7E23F5FC006B6CE1B1D7CDA849BDE46A145F80EE97C21AA2154FA3A5CF25C75E225C6F3384D3C0C6BEF5061B87E8D583BEFDF790ECD351F6D2B645E26904DE3F8A9861CC3EAD0AA40BD7C09C1F5F655A9E7BA7986B92B73FD9A6A69F54EFC92AC7E21D15C9B85A76084D1EEFBC4781B91E231E9CE5F007BC75A8656CBD98E282671C08A5400C4E4D039DE5FD63AA89A618C5668256B12672C66082F0348B6204DD0ADE58532C967D055A5D2C34C43DF9998820B5DFC4C49C6820191CB3EC81062AA51E23CEEA9A37AB523B24C0E93B440FDC17A50B219AB0D373014C25EE8F", 16),
		large.NewIntFromString("B22FDF91EE6BA01BDE4969C1A986EA1F81C4A1795921403F3437D681D05E95167C2F6414CCB74AC8D6B3BA8C0E85C7E4DEB0E8B5256D37BC5C21C8BE068F5342858AFF2FC7FF2644EBED8B10271941C74C86CCD71AA6D2D98E4C8C70875044900F842998037A7DFB9BC63BAF1BC2800E73AF9615E4F5B869D4C6DE6E5F48FACE9CA594CC5D228CB7F763A0AD6BF6ED78B27F902D9ADA38A1FCD7D09E398CE377BB15A459044D3B8541DC6D8049B66AE1662682254E69FAD31CA0016251D0522EF8FE587A3F6E3AB1E5F9D8C2998874ABAB205217E95B234A7D3E69713B884918ADB57360B5DE97336C7DC2EB8A3FEFB0C4290E7A92FF5758529AC45273135427", 16))

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
