package receptionID

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

func TestIdentity_EncodeDecode(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	r := Identity{
		EphemeralIdentity: EphemeralIdentity{
			EphId:  ephemeral.Id{},
			Source: &id.Permissioning,
		},
		AddressSize: 15,
		End:         netTime.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  netTime.Now().Round(0),
		EndValid:    netTime.Now().Round(0),
		Ephemeral:   false,
	}

	err := r.store(kv)
	if err != nil {
		t.Errorf("Failed to store: %+v", err)
	}

	rLoad, err := loadIdentity(kv)
	if err != nil {
		t.Errorf("Failed to load: %+v", err)
	}

	if !r.Equal(rLoad) {
		t.Errorf("Registrations are not the same.\nsaved:  %+v\nloaded: %+v",
			r, rLoad)
	}
}

func TestIdentity_Delete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	r := Identity{
		EphemeralIdentity: EphemeralIdentity{
			EphId:  ephemeral.Id{},
			Source: &id.Permissioning,
		},
		AddressSize: 15,
		End:         netTime.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  netTime.Now().Round(0),
		EndValid:    netTime.Now().Round(0),
		Ephemeral:   false,
	}

	err := r.store(kv)
	if err != nil {
		t.Errorf("Failed to store: %s", err)
	}

	err = r.delete(kv)
	if err != nil {
		t.Errorf("Failed to delete: %s", err)
	}

	_, err = loadIdentity(kv)
	if err == nil {
		t.Error("Load after delete succeeded.")
	}
}

func TestIdentity_String(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	received, _ := generateFakeIdentity(rng, 15, timestamp)
	expected := "-1763 U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID"

	s := received.String()
	if s != expected {
		t.Errorf("String did not return the correct value."+
			"\nexpected: %s\nreceived: %s", expected, s)
	}
}

func TestIdentity_Equal(t *testing.T) {
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	a, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, timestamp)
	b, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, timestamp)
	c, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, netTime.Now())

	if !a.Identity.Equal(b.Identity) {
		t.Errorf("Equal() found two equal identities as unequal."+
			"\na: %s\nb: %s", a, b)
	}

	if a.Identity.Equal(c.Identity) {
		t.Errorf("Equal() found two unequal identities as equal."+
			"\na: %s\nc: %s", a, c)
	}
}
