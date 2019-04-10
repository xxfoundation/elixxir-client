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
