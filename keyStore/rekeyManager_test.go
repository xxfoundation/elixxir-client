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

// Test all other functions of RekeyManager
func TestRekeyManager(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107),
		large.NewInt(2),
		large.NewInt(5))
	baseKey := grp.NewInt(57)
	privKey := grp.NewInt(5)
	pubKey := grp.NewInt(42)
	partner := id.NewUserFromUint(14, t)
	userID := id.NewUserFromUint(18, t)
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
		t.Errorf("GetOutCtx returned something but expected nil")
	}

	// Get added value and compare
	actual = rkm.GetCtx(partner)

	if actual == nil {
		t.Errorf("GetOutCtx returned nil")
	} else if actual.BaseKey.Cmp(baseKey) != 0 {
		t.Errorf("BaseKey doesn't match for RekeyContext added to Incoming map")
	} else if actual.PrivKey.Cmp(privKey) != 0 {
		t.Errorf("PrivKey doesn't match for RekeyContext added to Incoming map")
	} else if actual.PubKey.Cmp(pubKey) != 0 {
		t.Errorf("PubKey doesn't match for RekeyContext added to Incoming map")
	}

	// Delete value and confirm it's gone
	rkm.DeleteCtx(partner)

	actual = rkm.GetCtx(partner)

	if actual != nil {
		t.Errorf("GetOutCtx returned something but expected nil after deletion")
	}
}
