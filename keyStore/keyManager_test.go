package keyStore

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// initGroup sets up the cryptographic constants for cMix
func initGroup() *cyclic.Group {

	base := 16

	pString := "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

	gString := "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)

	grp := cyclic.NewGroup(p, g)

	return grp
}

// Test creation of KeyManager
func TestKeyManager_New(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, nil, nil,
		partner, true, 12, 10, 10)

	if km == nil {
		t.Errorf("NewManager returned nil")
	}
}

// Test KeyManager base key getter
func TestKeyManager_GetBaseKey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	result := km.GetBaseKey()

	if result.Cmp(baseKey) != 0 {
		t.Errorf("GetBaseKey returned wrong value, "+
			"expected: %s, got: %s",
			privKey.Text(10), result.Text(10))
	}
}

// Test KeyManager private key getter
func TestKeyManager_GetPrivKey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	result := km.GetPrivKey()

	if result.Cmp(privKey) != 0 {
		t.Errorf("GetPrivKey returned wrong value, "+
			"expected: %s, got: %s",
			privKey.Text(10), result.Text(10))
	}
}

// Test KeyManager public key getter
func TestKeyManager_GetPubKey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	result := km.GetPubKey()

	if result.Cmp(pubKey) != 0 {
		t.Errorf("GetPubKey returned wrong value, "+
			"expected: %s, got: %s",
			pubKey.Text(10), result.Text(10))
	}
}

// Test KeyManager partner getter
func TestKeyManager_GetPartner(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	result := km.GetPartner()

	if *result != *partner {
		t.Errorf("GetPartner returned wrong value, "+
			"expected: %s, got: %s",
			*partner, *result)
	}
}

// Test rekey trigger
func TestKeyManager_Rekey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, nil, nil,
		partner, true, 12, 10, 10)

	var action Action
	for i := 0; i < 9; i++ {
		action = km.updateState(false)
		if action != None {
			t.Errorf("Expected 'None' action, got %s instead",
				action)
		}
	}

	action = km.updateState(false)
	if action != Rekey {
		t.Errorf("Expected 'Rekey' action, got %s instead",
			action)
	}
}

// Test purge trigger
func TestKeyManager_Purge(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, nil, nil,
		partner, true, 12, 10, 10)

	var action Action
	for i := 0; i < 9; i++ {
		action = km.updateState(true)
		if action != None {
			t.Errorf("Expected 'None' action, got %s instead",
				action)
		}
	}

	action = km.updateState(true)
	if action != Purge {
		t.Errorf("Expected 'Purge' action, got %s instead",
			action)
	}

	// Confirm that state is now deleted
	action = km.updateState(false)
	if action != Deleted {
		t.Errorf("Expected 'Deleted' action, got %s instead",
			action)
	}
}

// Test receive state update
func TestKeyManager_UpdateRecvState(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewManager(baseKey, nil, nil,
		partner, false, 12, 10, 10)

	expectedVal := uint64(0x0010000001000008)
	// Mark some keys as used and confirm expected value
	km.updateRecvState(false, 3)
	km.updateRecvState(false, 24)
	km.updateRecvState(false, 52)

	if *km.recvKeysState[0] != expectedVal {
		t.Errorf("UpdateRecvState failed for Key, expected"+
			" %d, got %d", expectedVal, *km.recvKeysState[0])
	}

	expectedVal = uint64(0x0000080000040020)
	// Mark some Rekeys as used and confirm expected value
	km.updateRecvState(true, 5)
	km.updateRecvState(true, 18)
	km.updateRecvState(true, 43)

	if *km.recvReKeysState[0] != expectedVal {
		t.Errorf("UpdateRecvState failed for ReKey, expected"+
			" %d, got %d", expectedVal, *km.recvReKeysState[0])
	}
}

// Test KeyManager Key Generation
func TestKeyManager_GenerateKeys(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)

	ks := NewStore()
	kmSend := NewManager(baseKey, nil, nil,
		partner, true, 12, 10, 10)

	// Generate Send Keys
	kmSend.GenerateKeys(grp, userID)
	ks.AddSendManager(kmSend)

	kmRecv := NewManager(baseKey, nil, nil,
		partner, false, 12, 10, 10)

	// Generate Receive Keys
	e2ekeys := kmRecv.GenerateKeys(grp, userID)
	ks.AddRecvManager(kmRecv)
	ks.AddReceiveKeysByFingerprint(e2ekeys)

	// Confirm Send KeyManager is stored correctly in KeyStore map
	retKM := ks.GetSendManager(partner)
	if retKM != kmSend {
		t.Errorf("KeyManager stored in KeyStore is not the same")
	}

	// Confirm keys can be correctly pop'ed from KeyManager
	actual, action := retKM.PopKey()

	if actual == nil {
		t.Errorf("KeyManager returned nil when poping key")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	actual, action = retKM.PopRekey()

	if actual == nil {
		t.Errorf("KeyManager returned nil when poping rekey")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	// Confirm Receive Keys can be obtained from KeyStore
	actual = ks.GetRecvKey(kmRecv.recvKeysFingerprint[4])
	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for Key")
	}

	actual = ks.GetRecvKey(e2ekeys[8].KeyFingerprint())

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for ReKey")
	}
}

// Test KeyManager destroy
func TestKeyManager_Destroy(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)

	ks := NewStore()
	km := NewManager(baseKey, nil, nil,
		partner, true, 12, 10, 10)

	// Generate Send Keys
	km.GenerateKeys(grp, userID)
	ks.AddSendManager(km)

	km2 := NewManager(baseKey, nil, nil,
		partner, false, 12, 10, 10)

	// Generate Receive Keys
	e2ekeys := km2.GenerateKeys(grp, userID)
	// TODO add ks keys here
	ks.AddRecvManager(km2)
	ks.AddReceiveKeysByFingerprint(e2ekeys)

	// Confirm Send KeyManager is stored correctly in KeyStore map
	retKM := ks.GetSendManager(partner)
	if retKM != km {
		t.Errorf("KeyManager stored in KeyStore is not the same")
	}

	// Confirm keys can be correctly pop'ed from KeyManager
	actual, action := retKM.PopKey()

	if actual == nil {
		t.Errorf("KeyManager returned nil when poping key")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	actual, action = retKM.PopRekey()

	if actual == nil {
		t.Errorf("KeyManager returned nil when poping rekey")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			action)
	}

	// Confirm Receive Keys can be obtained from KeyStore
	actual = ks.GetRecvKey(km2.recvKeysFingerprint[4])

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for Key")
	}

	actual = ks.GetRecvKey(km2.recvReKeysFingerprint[8])
	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for ReKey")
	}

	// Destroy KeyManager and confirm KeyManager is gone from map
	km.Destroy(ks)

	retKM = ks.GetSendManager(partner)
	if retKM != nil {
		t.Errorf("KeyManager was not properly removed from KeyStore")
	}

}

// Test GOB Encode/Decode of KeyManager
// and do a simple comparison after
func TestKeyManager_GobSimple(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)

	var byteBuf bytes.Buffer

	enc := gob.NewEncoder(&byteBuf)
	dec := gob.NewDecoder(&byteBuf)

	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	err := enc.Encode(km)

	if err != nil {
		t.Errorf("Error GOB Encoding KeyManager: %s", err)
	}

	outKm := &KeyManager{}

	err = dec.Decode(&outKm)

	if err != nil {
		t.Errorf("Error GOB Decoding KeyManager: %s", err)
	}

	if km.baseKey.Cmp(outKm.baseKey) != 0 {
		t.Errorf("GobEncoder/GobDecoder failed on BaseKey, "+
			"Expected: %v; Recieved: %v ",
			km.baseKey.TextVerbose(10, 12),
			outKm.baseKey.TextVerbose(10, 12))
	}

	if *km.partner != *outKm.partner {
		t.Errorf("GobEncoder/GobDecoder failed on Partner, "+
			"Expected: %v; Recieved: %v ",
			*km.partner,
			*outKm.partner)
	}

	if *km.sendState != *outKm.sendState {
		t.Errorf("GobEncoder/GobDecoder failed on State, "+
			"Expected: %v; Recieved: %v ",
			*km.sendState,
			*outKm.sendState)
	}

	if km.ttl != outKm.ttl {
		t.Errorf("GobEncoder/GobDecoder failed on TTL, "+
			"Expected: %v; Recieved: %v ",
			km.ttl,
			outKm.ttl)
	}

	if km.numKeys != outKm.numKeys {
		t.Errorf("GobEncoder/GobDecoder failed on NumKeys, "+
			"Expected: %v; Recieved: %v ",
			km.numKeys,
			outKm.numKeys)
	}

	if km.numReKeys != outKm.numReKeys {
		t.Errorf("GobEncoder/GobDecoder failed on NumReKeys, "+
			"Expected: %v; Recieved: %v ",
			km.numReKeys,
			outKm.numReKeys)
	}

	for i := 0; i < int(numStates); i++ {
		if *km.recvKeysState[i] != *outKm.recvKeysState[i] {
			t.Errorf("GobEncoder/GobDecoder failed on RecvKeysState[%d], "+
				"Expected: %v; Recieved: %v ",
				i,
				*km.recvKeysState[i],
				*outKm.recvKeysState[i])
		}
	}

	for i := 0; i < int(numReStates); i++ {
		if *km.recvReKeysState[i] != *outKm.recvReKeysState[i] {
			t.Errorf("GobEncoder/GobDecoder failed on RecvReKeysState[%d], "+
				"Expected: %v; Recieved: %v ",
				i,
				*km.recvReKeysState[i],
				*outKm.recvReKeysState[i])
		}
	}
}

// Tests that GobDecode() for Key Manager throws an error for a
// malformed byte array
func TestKeyManager_GobDecodeError(t *testing.T) {
	km := KeyManager{}
	err := km.GobDecode([]byte{})

	if err.Error() != "EOF" {
		t.Errorf("GobDecode() did not produce the expected error\n\treceived: %v"+
			"\n\texpected: %v", err, errors.New("EOF"))
	}
}

// Test that key maps are reconstructed correctly after
// Key Manager GOB Encode/Decode
func TestKeyManager_Gob(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)

	ks := NewStore()
	km := NewManager(baseKey, privKey, pubKey,
		partner, true, 12, 10, 10)

	// Generate Send Keys
	km.GenerateKeys(grp, userID)
	ks.AddSendManager(km)

	km2 := NewManager(baseKey, privKey, pubKey,
		partner, false, 12, 10, 10)

	// Generate Receive Keys
	e2ekeys := km2.GenerateKeys(grp, userID)
	ks.AddRecvManager(km2)
	ks.AddReceiveKeysByFingerprint(e2ekeys)

	// Generate keys here to have a way to compare after
	sendKeys := e2e.DeriveKeys(grp, baseKey, userID, uint(km.numKeys))
	sendReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, userID, uint(km.numReKeys))
	recvKeys := e2e.DeriveKeys(grp, baseKey, partner, uint(km.numKeys))
	recvReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, partner, uint(km.numReKeys))

	var expectedKeyMap = make(map[string]bool)

	for _, key := range sendKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())] = true
	}

	for _, key := range sendReKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())] = true
	}

	for _, key := range recvKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())] = true
	}

	for _, key := range recvReKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())] = true
	}

	// Use some send keys and mark on expected map as used
	retKM := ks.GetSendManager(partner)
	if retKM != km {
		t.Errorf("KeyManager stored in KeyStore is not the same")
	}
	key, _ := retKM.PopKey()
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = retKM.PopKey()
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = retKM.PopKey()
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedSendKeys := 3

	key, _ = retKM.PopRekey()
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = retKM.PopRekey()
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedSendReKeys := 2

	// Use some receive keys and mark on expected map as used
	key = ks.GetRecvKey(km2.recvKeysFingerprint[3])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ks.GetRecvKey(km2.recvKeysFingerprint[8])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ks.GetRecvKey(km2.recvKeysFingerprint[6])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ks.GetRecvKey(km2.recvKeysFingerprint[1])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedRecvKeys := 4

	key = ks.GetRecvKey(km2.recvReKeysFingerprint[4])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedRecvReKeys := 1

	// Now GOB Encode Key Manager
	var byteBuf bytes.Buffer

	enc := gob.NewEncoder(&byteBuf)
	dec := gob.NewDecoder(&byteBuf)

	err := enc.Encode(km)

	if err != nil {
		t.Errorf("Error GOB Encoding KeyManager: %s", err)
	}

	// Destroy KeyManager and confirm KeyManager is gone from map
	km.Destroy(ks)

	retKM = ks.GetSendManager(partner)
	if retKM != nil {
		t.Errorf("KeyManager was not properly removed from KeyStore")
	}

	// GOB Decode Key Manager
	sendKm := &KeyManager{}
	err = dec.Decode(&sendKm)

	if err != nil {
		t.Errorf("Error GOB Decoding KeyManager: %s", err)
	}

	err = enc.Encode(km2)

	if err != nil {
		t.Errorf("Error GOB Encoding KeyManager2: %s", err)
	}

	// Destroy Key Manager (and maps) and confirm no more receive keys exist
	km2.Destroy(ks)

	// GOB Decode Key Manager2
	outKm2 := &KeyManager{}
	err = dec.Decode(&outKm2)

	if err != nil {
		t.Errorf("Error GOB Decoding KeyManager2: %s", err)
	}

	// Generate Keys from decoded Key Managers
	e2ekeys = sendKm.GenerateKeys(grp, userID)
	ks.AddSendManager(sendKm)
	//ks.AddReceiveKeysByFingerprint(e2ekeys)

	e2ekeys = outKm2.GenerateKeys(grp, userID)
	ks.AddRecvManager(km)
	ks.AddReceiveKeysByFingerprint(e2ekeys)

	// Confirm maps are the same as before delete

	// First, check that len of send Stacks matches expected
	if sendKm.sendKeys.keys.Len() != int(sendKm.numKeys)-usedSendKeys {
		t.Errorf("SendKeys Stack contains more keys than expected after decode."+
			" Expected: %d, Got: %d",
			int(sendKm.numKeys)-usedSendKeys,
			sendKm.sendKeys.keys.Len())
	}

	if sendKm.sendReKeys.keys.Len() != int(sendKm.numReKeys)-usedSendReKeys {
		t.Errorf("SendReKeys Stack contains more keys than expected after decode."+
			" Expected: %d, Got: %d",
			int(sendKm.numReKeys)-usedSendReKeys,
			sendKm.sendReKeys.keys.Len())
	}

	// Now confirm that all send keys are in the expected map
	retKM = ks.GetSendManager(partner)
	for i := 0; i < int(sendKm.numKeys)-usedSendKeys; i++ {
		key, _ := retKM.PopKey()
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("SendKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	for i := 0; i < int(sendKm.numReKeys)-usedSendReKeys; i++ {
		key, _ := retKM.PopRekey()
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("SendReKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	// Check that len of fingerprint lists matches expected
	if len(outKm2.recvKeysFingerprint) != int(outKm2.numKeys)-usedRecvKeys {
		t.Errorf("ReceiveKeys list contains more keys than expected after decode."+
			" Expected: %d, Got: %d",
			int(outKm2.numKeys)-usedRecvKeys,
			len(outKm2.recvKeysFingerprint))
	}

	if len(outKm2.recvReKeysFingerprint) != int(outKm2.numReKeys)-usedRecvReKeys {
		t.Errorf("ReceiveReKeys list contains more keys than expected after decode."+
			" Expected: %d, Got: %d",
			int(outKm2.numReKeys)-usedRecvReKeys,
			len(outKm2.recvReKeysFingerprint))
	}

	// Now confirm that all receiving keys are in the expected map
	for i := 0; i < int(outKm2.numKeys)-usedRecvKeys; i++ {
		key := ks.GetRecvKey(outKm2.recvKeysFingerprint[i])
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("ReceiveKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	for i := 0; i < int(outKm2.numReKeys)-usedRecvReKeys; i++ {
		key := ks.GetRecvKey(outKm2.recvReKeysFingerprint[i])
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("ReceiveReKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}
}
