package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
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

// Test KeyManager destroy
func TestKeyManager_Destroy(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	partner := id.NewUserFromUint(14, t)

	km := NewKeyManager(baseKey, partner, 12, 10, 10)

	// Create stacks, add some keys and store in global maps
	km.sendKeys = NewKeyStack()
	km.sendReKeys = NewKeyStack()
	TransmissionKeys.Store(partner, km.sendKeys)
	TransmissionReKeys.Store(partner, km.sendReKeys)
	for i := 0; i < 12; i++ {
		key := new(E2EKey)
		key.outer = format.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = km
		km.sendKeys.Push(key)
	}
	for i := 0; i < 10; i++ {
		key := new(E2EKey)
		key.outer = format.Rekey
		key.key = grp.NewInt(int64(i + 2))
		key.manager = km
		km.sendReKeys.Push(key)
	}
	// Create fingerprint lists and add keys to global map
	fpList := make([]format.Fingerprint, 12)
	for i := 0; i < 12; i++ {
		key := new(E2EKey)
		key.outer = format.E2E
		key.key = grp.NewInt(int64(i + 2))
		key.manager = km
		fpList[i] = key.KeyFingerprint()
		ReceptionKeys.Store(fpList[i], key)
	}
	fpReList := make([]format.Fingerprint, 10)
	for i := 0; i < 10; i++ {
		key := new(E2EKey)
		key.outer = format.Rekey
		key.key = grp.NewInt(int64(i + 2))
		key.manager = km
		fpReList[i] = key.KeyFingerprint()
		ReceptionKeys.Store(fpReList[i], key)
	}
	km.receiveKeysFP = fpList
	km.receiveReKeysFP = fpReList

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

	actual = ReceptionKeys.Pop(fpList[4])

	if actual == nil {
		t.Errorf("ReceptionKeys Map returned nil for Key")
	}

	actual = ReceptionKeys.Pop(fpReList[8])

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
		actual = ReceptionKeys.Pop(fpList[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for Key")
		}
	}

	for i := 0; i < 10; i++ {
		actual = ReceptionKeys.Pop(fpReList[i])
		if actual != nil {
			t.Errorf("ReceptionKeys Map should have returned nil for ReKey")
		}
	}
}
