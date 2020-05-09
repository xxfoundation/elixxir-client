package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// Test creation of RekeyManager
func TestRekeyManager_New(t *testing.T) {
	rkm := NewRekeyManager()

	if rkm == nil {
		t.Errorf("NewRekeyManager returned nil")
	}
}

// Test all Ctx related functions of RekeyManager
func TestRekeyManager_Ctx(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewIdFromUInt(14, id.User, t)
	userID := id.NewIdFromUInt(18, id.User, t)
	rkm := NewRekeyManager()

	val := &RekeyContext{
		BaseKey: baseKey,
		PrivKey: privKey,
		PubKey:  pubKey,
	}

	// Add RekeyContext to map
	rkm.AddCtx(partner, val)

	// Confirm different partner returns nil
	actual := rkm.GetCtx(userID)

	if actual != nil {
		t.Errorf("GetCtx returned something but expected nil")
	}

	// Get added value and compare
	actual = rkm.GetCtx(partner)

	if actual == nil {
		t.Errorf("GetCtx returned nil")
	} else if actual.BaseKey.Cmp(baseKey) != 0 {
		t.Errorf("BaseKey doesn't match for RekeyContext added to Contexts map")
	} else if actual.PrivKey.Cmp(privKey) != 0 {
		t.Errorf("PrivKey doesn't match for RekeyContext added to Contexts map")
	} else if actual.PubKey.Cmp(pubKey) != 0 {
		t.Errorf("PubKey doesn't match for RekeyContext added to Contexts map")
	}

	// Delete value and confirm it's gone
	rkm.DeleteCtx(partner)

	actual = rkm.GetCtx(partner)

	if actual != nil {
		t.Errorf("GetCtx returned something but expected nil after deletion")
	}
}

// Test all Keys related functions of RekeyManager
func TestRekeyManager_Keys(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewIdFromUInt(14, id.User, t)
	userID := id.NewIdFromUInt(18, id.User, t)
	rkm := NewRekeyManager()

	val := &RekeyKeys{
		CurrPrivKey: privKey,
		CurrPubKey:  pubKey,
	}

	// Add RekeyKeys to map
	rkm.AddKeys(partner, val)

	// Confirm different partner returns nil
	actual := rkm.GetKeys(userID)

	if actual != nil {
		t.Errorf("GetNodeKeys returned something but expected nil")
	}

	// Get added value and compare
	actual = rkm.GetKeys(partner)

	if actual == nil {
		t.Errorf("GetNodeKeys returned nil")
	} else if actual.CurrPrivKey.Cmp(privKey) != 0 {
		t.Errorf("CurrPrivKey doesn't match for RekeyKeys added to Keys map")
	} else if actual.CurrPubKey.Cmp(pubKey) != 0 {
		t.Errorf("CurrPubKey doesn't match for RekeyKeys added to Keys map")
	}

	// Delete value and confirm it's gone
	rkm.DeleteKeys(partner)

	actual = rkm.GetKeys(partner)

	if actual != nil {
		t.Errorf("GetNodeKeys returned something but expected nil after deletion")
	}

	// Confirm RekeyKeys behavior of key rotation
	newPrivKey := grp.NewInt(7)
	newPubKey := grp.NewInt(91)

	// Add new PrivKey
	val.NewPrivKey = newPrivKey

	// Call rotate and confirm nothing changes
	val.RotateKeysIfReady()

	if val.CurrPrivKey.Cmp(privKey) != 0 {
		t.Errorf("CurrPrivKey doesn't match for RekeyKeys after adding new PrivateKey")
	} else if val.CurrPubKey.Cmp(pubKey) != 0 {
		t.Errorf("CurrPubKey doesn't match for RekeyKeys after adding new PrivateKey")
	}

	// Add new PubKey, rotate, and confirm keys change
	val.NewPubKey = newPubKey
	val.RotateKeysIfReady()

	if val.CurrPrivKey.Cmp(newPrivKey) != 0 {
		t.Errorf("CurrPrivKey doesn't match for RekeyKeys after key rotation")
	} else if val.CurrPubKey.Cmp(newPubKey) != 0 {
		t.Errorf("CurrPubKey doesn't match for RekeyKeys after key rotation")
	}
}
