package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

func TestRekeyManager_New(t *testing.T) {
	rkm := NewRekeyManager()

	if rkm == nil {
		t.Errorf("NewRekeyManager returned nil")
	}
}

// Test Incoming Context map functions
func TestRekeyManager_InCtx(t *testing.T) {
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
	rkm.AddInCtx(partner, val)

	// Confirm different partner returns nil
	actual := rkm.GetInCtx(userID)

	if actual != nil {
		t.Errorf("GetInCtx returned something but expected nil")
	}

	// Get added value and compare
	actual = rkm.GetInCtx(partner)

	if actual == nil {
		t.Errorf("GetInCtx returned nil")
	} else if actual.BaseKey.Cmp(baseKey) != 0 {
		t.Errorf("BaseKey doesn't match for RekeyContext added to Incoming map")
	} else if actual.PrivKey.Cmp(privKey) != 0 {
		t.Errorf("PrivKey doesn't match for RekeyContext added to Incoming map")
	} else if actual.PubKey.Cmp(pubKey) != 0 {
		t.Errorf("PubKey doesn't match for RekeyContext added to Incoming map")
	}

	// Delete value and confirm it's gone
	rkm.DeleteInCtx(partner)

	actual = rkm.GetInCtx(partner)

	if actual != nil {
		t.Errorf("GetInCtx returned something but expected nil after deletion")
	}
}

// Test Outgoing Context map functions
func TestRekeyManager_OutCtx(t *testing.T) {
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
	rkm.AddOutCtx(partner, val)

	// Confirm different partner returns nil
	actual := rkm.GetOutCtx(userID)

	if actual != nil {
		t.Errorf("GetOutCtx returned something but expected nil")
	}

	// Get added value and compare
	actual = rkm.GetOutCtx(partner)

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
	rkm.DeleteOutCtx(partner)

	actual = rkm.GetOutCtx(partner)

	if actual != nil {
		t.Errorf("GetOutCtx returned something but expected nil after deletion")
	}
}
