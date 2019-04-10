package keyStore

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
)

func actionPrint(act KeyAction) string {
	var ret string
	switch act {
	case None:
		ret = "None"
	case Rekey:
		ret = "Rekey"
	case Purge:
		ret = "Purge"
	case Deleted:
		ret = "Deleted"
	}
	return ret
}

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

	qString := "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"

	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)
	q := large.NewIntFromString(qString, base)

	grp := cyclic.NewGroup(p, g, q)

	return grp
}

// Test creation of KeyManager
func TestKeyManager_New(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	if km == nil {
		t.Errorf("NewKeyManager returned nil")
	}
}

// Test rekey trigger
func TestKeyManager_Rekey(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	var action KeyAction
	for i := 0; i < 9; i++ {
		action = km.UpdateState(false)
		if action != None {
			t.Errorf("Expected 'None' action, got %s instead",
				actionPrint(action))
		}
	}

	action = km.UpdateState(false)
	if action != Rekey {
		t.Errorf("Expected 'Rekey' action, got %s instead",
			actionPrint(action))
	}
}

// Test purge trigger
func TestKeyManager_Purge(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	var action KeyAction
	for i := 0; i < 9; i++ {
		action = km.UpdateState(true)
		if action != None {
			t.Errorf("Expected 'None' action, got %s instead",
				actionPrint(action))
		}
	}

	action = km.UpdateState(true)
	if action != Purge {
		t.Errorf("Expected 'Purge' action, got %s instead",
			actionPrint(action))
	}

	// Confirm that state is now deleted
	action = km.UpdateState(false)
	if action != Deleted {
		t.Errorf("Expected 'Deleted' action, got %s instead",
			actionPrint(action))
	}
}

// Test receive state update
func TestKeyManager_UpdateRecvState(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	expectedVal := uint64(0x0010000001000008)
	// Mark some keys as used and confirm expected value
	km.UpdateRecvState(3)
	km.UpdateRecvState(24)
	km.UpdateRecvState(52)

	if *km.recvState[0] != expectedVal {
		t.Errorf("UpdateRecvState failed, expected"+
			" %d, got %d", expectedVal, *km.recvState[0])
	}
}

// Test KeyManager Key Generation
func TestKeyManager_GenerateKeys(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	// Generate Keys
	km.GenerateKeys(grp, userID)

	// Confirm keys can be obtained from all maps
	actual, action := TransmissionKeys.Pop(partner)

	if actual == nil {
		t.Errorf("TransmissionKeys Map returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual, action = TransmissionReKeys.Pop(partner)

	if actual == nil {
		t.Errorf("TransmissionReKeys Map returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual = ReceptionKeys.Pop(km.receiveKeysFP[4])

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for Key")
	}

	actual = ReceptionKeys.Pop(km.receiveReKeysFP[8])

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

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	// Generate Keys
	km.GenerateKeys(grp, userID)

	// Confirm keys can be obtained from all maps
	actual, action := TransmissionKeys.Pop(partner)

	if actual == nil {
		t.Errorf("TransmissionKeys Map returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual, action = TransmissionReKeys.Pop(partner)

	if actual == nil {
		t.Errorf("TransmissionReKeys Map returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual = ReceptionKeys.Pop(km.receiveKeysFP[4])

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for Key")
	}

	actual = ReceptionKeys.Pop(km.receiveReKeysFP[8])

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for ReKey")
	}

	// Destroy KeyManager and confirm no more keys exist
	km.Destroy()

	actual, action = TransmissionKeys.Pop(partner)

	if actual != nil {
		t.Errorf("TransmissionKeys Map should have returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual, action = TransmissionReKeys.Pop(partner)

	if actual != nil {
		t.Errorf("TransmissionReKeys Map should have returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	for i := 0; i < 12; i++ {
		actual = ReceptionKeys.Pop(km.receiveKeysFP[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for Key")
		}
	}

	for i := 0; i < 10; i++ {
		actual = ReceptionKeys.Pop(km.receiveReKeysFP[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for ReKey")
		}
	}
}

// Test GOB Encode/Decode of KeyManager
func TestKeyManager_GobSimple(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	var byteBuf bytes.Buffer

	enc := gob.NewEncoder(&byteBuf)
	dec := gob.NewDecoder(&byteBuf)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

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

	if *km.state != *outKm.state {
		t.Errorf("GobEncoder/GobDecoder failed on State, "+
			"Expected: %v; Recieved: %v ",
			*km.state,
			*outKm.state)
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

	for i := 0; i < int(maxStates); i++ {
		if *km.recvState[i] != *outKm.recvState[i] {
			t.Errorf("GobEncoder/GobDecoder failed on RecvState[%d], "+
				"Expected: %v; Recieved: %v ",
				i,
				*km.recvState[i],
				*outKm.recvState[i])
		}
	}
}

// Tests that GobDecode() for Key Manager throws an error for a
// malformed byte array
func TestKeyManager_GobDecodeError(t *testing.T) {
	km := KeyManager{}
	err := km.GobDecode([]byte{})

	if !reflect.DeepEqual(err, errors.New("EOF")) {
		t.Errorf("GobDecode() did not produce the expected error\n\treceived: %v"+
			"\n\texpected: %v", err, errors.New("EOF"))
	}
}

// Test that key maps are reconstructed correctly after
// Key Manager GOB Encode/Decode
func TestKeyManager_Gob(t *testing.T) {
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	// Generate Keys
	km.GenerateKeys(grp, userID)

	// Generate keys here to have a way to compare after
	sendKeys := e2e.DeriveKeys(grp, baseKey, userID, uint(km.numKeys))
	sendReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, userID, uint(km.numReKeys))
	recvKeys := e2e.DeriveKeys(grp, baseKey, partner, uint(km.numKeys))
	recvReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, partner, uint(km.numReKeys))

	var expectedKeyMap = make(map[string]bool)

	for _, key := range sendKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())]=true
	}

	for _, key := range sendReKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())]=true
	}

	for _, key := range recvKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())]=true
	}

	for _, key := range recvReKeys {
		expectedKeyMap[base64.StdEncoding.EncodeToString(key.Bytes())]=true
	}

	// Use some send keys and mark on expected map as used
	key, _ := TransmissionKeys.Pop(partner)
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = TransmissionKeys.Pop(partner)
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = TransmissionKeys.Pop(partner)
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedSendKeys := 3

	key, _ = TransmissionReKeys.Pop(partner)
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key, _ = TransmissionReKeys.Pop(partner)
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedSendReKeys := 2

	// Use some receive keys and mark on expected map as used
	key = ReceptionKeys.Pop(km.receiveKeysFP[3])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ReceptionKeys.Pop(km.receiveKeysFP[8])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ReceptionKeys.Pop(km.receiveKeysFP[6])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	key = ReceptionKeys.Pop(km.receiveKeysFP[1])
	expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] = false
	usedRecvKeys := 4

	key = ReceptionKeys.Pop(km.receiveReKeysFP[4])
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

	// Destroy Key Manager (and maps) and confirm no more keys exist
	km.Destroy()

	actual, action := TransmissionKeys.Pop(partner)

	if actual != nil {
		t.Errorf("TransmissionKeys Map should have returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	actual, action = TransmissionReKeys.Pop(partner)

	if actual != nil {
		t.Errorf("TransmissionReKeys Map should have returned nil")
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	for i := 0; i < 12; i++ {
		actual = ReceptionKeys.Pop(km.receiveKeysFP[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for Key")
		}
	}

	for i := 0; i < 10; i++ {
		actual = ReceptionKeys.Pop(km.receiveReKeysFP[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for ReKey")
		}
	}

	// GOB Decode Key Manager
	outKm := &KeyManager{}
	err = dec.Decode(&outKm)

	if err != nil {
		t.Errorf("Error GOB Decoding KeyManager: %s", err)
	}

	// Generate Keys from decoded Key Manager
	outKm.GenerateKeys(grp, userID)

	// Confirm maps are the same as before delete

	// First, check that len of send Stacks matches expected
	if outKm.sendKeys.keys.Len() != int(outKm.numKeys) - usedSendKeys {
		t.Errorf("SendKeys Stack contains more keys than expected after decode." +
			" Expected: %d, Got: %d",
			int(outKm.numKeys) - usedSendKeys,
			outKm.sendKeys.keys.Len())
	}

	if outKm.sendReKeys.keys.Len() != int(outKm.numReKeys) - usedSendReKeys {
		t.Errorf("SendReKeys Stack contains more keys than expected after decode." +
			" Expected: %d, Got: %d",
			int(outKm.numReKeys) - usedSendReKeys,
			outKm.sendReKeys.keys.Len())
	}

	// Now confirm that all send keys are in the expected map
	for i := 0; i < int(outKm.numKeys) - usedSendKeys; i++ {
		key, _ := TransmissionKeys.Pop(partner)
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("SendKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	for i := 0; i < int(outKm.numReKeys) - usedSendReKeys; i++ {
		key, _ := TransmissionReKeys.Pop(partner)
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("SendReKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	// Check that len of fingerprint lists matches expected
	if len(outKm.receiveKeysFP) != int(outKm.numKeys) - usedRecvKeys {
		t.Errorf("ReceiveKeys list contains more keys than expected after decode." +
			" Expected: %d, Got: %d",
			int(outKm.numKeys) - usedRecvKeys,
			len(outKm.receiveKeysFP))
	}

	if len(outKm.receiveReKeysFP) != int(outKm.numReKeys) - usedRecvReKeys {
		t.Errorf("ReceiveReKeys list contains more keys than expected after decode." +
			" Expected: %d, Got: %d",
			int(outKm.numReKeys) - usedRecvReKeys,
			len(outKm.receiveReKeysFP))
	}

	// Now confirm that all receiving keys are in the expected map
	for i := 0; i < int(outKm.numKeys) - usedRecvKeys; i++ {
		key := ReceptionKeys.Pop(outKm.receiveKeysFP[i])
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("ReceiveKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}

	for i := 0; i < int(outKm.numReKeys) - usedRecvReKeys; i++ {
		key := ReceptionKeys.Pop(outKm.receiveReKeysFP[i])
		if expectedKeyMap[base64.StdEncoding.EncodeToString(key.key.Bytes())] != true {
			t.Errorf("ReceiveReKey %v was used or didn't exist before",
				key.KeyFingerprint())
		}
	}
}
