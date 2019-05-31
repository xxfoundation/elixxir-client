package rekey

import (
	"bytes"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
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
	recipientID *id.User,
	cryptoType format.CryptoType,
	message []byte) error {
	d.listener <- message
	return nil
}

// SendMessage without partitions to the server
func (d *dummyMessaging) SendMessageNoPartition(sess user.Session,
	recipientID *id.User,
	cryptoType format.CryptoType,
	message []byte) error {
	d.listener <- message
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(session user.Session,
	delay time.Duration) {}

func TestMain(m *testing.M) {
	grp := globals.InitCrypto()
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
		u, user.NewRegistry(grp), "", nil, myPubKey, myPrivKey, grp)
	ListenCh = make(chan []byte, 100)
	fakeComm := &dummyMessaging{
		listener: ListenCh,
	}
	InitRekey(session, fakeComm)

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
		CryptoType: format.None,
		Receiver:   partnerID,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekeytrigger processing
	if rekeyTriggerList.err != nil {
		t.Errorf("RekeyTrigger returned error: %v", rekeyTriggerList.err.Error())
	}
	// Get new PubKey from Rekey message and confirm value matches
	// with PubKey created from privKey in Rekey Context
	value := <- ListenCh
	grp := session.GetGroup()
	actualPubKey := grp.NewIntFromBytes(value)
	privKey := session.GetRekeyManager().GetCtx(partnerID).PrivKey
	expectedPubKey := grp.NewInt(1)
	grp.ExpG(privKey, expectedPubKey)

	if expectedPubKey.Cmp(actualPubKey) != 0 {
		t.Errorf("RekeyTrigger publicKey mismatch, expected %s," +
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
		CryptoType: format.None,
		Receiver:   partnerID,
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
		CryptoType: format.None,
		Receiver:   session.GetCurrentUser().User,
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
		CryptoType: format.None,
		Receiver:   session.GetCurrentUser().User,
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
		CryptoType: format.None,
		Receiver:   session.GetCurrentUser().User,
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
	grp := globals.InitCrypto()
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
		CryptoType: format.Rekey,
		Receiver:   session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check no error occurred in rekey processing
	if rekeyList.err != nil {
		t.Errorf("Rekey returned error: %v", rekeyList.err.Error())
	}
	// Confirm hash of baseKey matches expected
	value := <- ListenCh
	// Get hash as last 32 bytes of message bytes
	actual := value[len(value)-32:]
	km = session.GetKeyStore().GetRecvManager(partnerID)
	baseKey := grp.NewInt(1)
	grp.Exp(km.GetPubKey(), km.GetPrivKey(), baseKey)
	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	expected := h.Sum(nil)

	if !bytes.Equal(expected, actual) {
		t.Errorf("Rekey hash(baseKey) mismatch, expected %x," +
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
		CryptoType: format.None,
		Receiver:   partnerID,
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
		CryptoType: format.Rekey,
		Receiver:   session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Check error occurred on Rekey
	if rekeyList.err == nil {
		t.Errorf("Rekey should have returned error")
	}
}
