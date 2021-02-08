package reception

import (
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestNewRegistration_Failed(t *testing.T) {
	// Generate an identity for use
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity
	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Time{}
	id.ExtraChecks = 0

	_, err := newRegistration(id, kv)
	if err == nil || !strings.Contains(err.Error(), "Cannot create a registration for an identity which has expired") {
		t.Error("Registration creation succeeded with expired identity.")
	}
}

func TestNewRegistration_Ephemeral(t *testing.T) {
	// Generate an identity for use
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity
	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1 * time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = true

	reg, err := newRegistration(id, kv)
	if err != nil {
		t.Fatalf("Registration creation failed when it should have "+
			"succeeded: %+v", err)
	}

	if reg.knownRounds == nil {
		t.Error("Ephemeral identity does not have a known rounds.")
	}

	if reg.knownRoundsStorage != nil {
		t.Error("Ephemeral identity has a known rounds storage.")
	}

	// Check if the known rounds is stored, it should not be
	if _, err = utility.LoadKnownRounds(reg.kv, knownRoundsStorageKey, id.calculateKrSize()); err == nil {
		t.Error("Ephemeral identity stored the known rounds when it should not have.")
	}

	if _, err = reg.kv.Get(identityStorageKey); err == nil {
		t.Error("Ephemeral identity stored the identity when it should not have.")
	}
}

func TestNewRegistration_Persistent(t *testing.T) {
	// Generate an identity for use
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity
	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1 * time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = false

	reg, err := newRegistration(id, kv)
	if err != nil {
		t.Fatalf("Registration creation failed when it should have "+
			"succeeded: %+v", err)
	}

	if reg.knownRounds == nil {
		t.Error("Persistent identity does not have a known rounds.")
	}

	if reg.knownRoundsStorage == nil {
		t.Error("Persistent identity does not have a known rounds storage.")
	}

	// Check if the known rounds is stored, it should not be
	if _, err = utility.LoadKnownRounds(reg.kv, knownRoundsStorageKey, id.calculateKrSize()); err != nil {
		t.Errorf("Persistent identity did not store known rounds when "+
			"it should: %+v", err)
	}

	if _, err = reg.kv.Get(identityStorageKey); err != nil {
		t.Error("Persistent identity did not store the identity when it should.")
	}
}

func TestLoadRegistration(t *testing.T) {
	// Generate an identity for use
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity
	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1 * time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = false

	_, err := newRegistration(id, kv)
	if err != nil {
		t.Fatalf("Registration creation failed when it should have "+
			"succeeded: %+v", err)
	}

	reg, err := loadRegistration(idu.EphId, idu.Source, idu.StartValid, kv)
	if err != nil {
		t.Fatalf("Registration loading failed: %+v", err)
	}

	if reg.knownRounds != nil {
		t.Error("Loading has a separated known rounds, it should not have.")
	}

	if reg.knownRoundsStorage == nil {
		t.Error("Loading identity does not have a known rounds storage.")
	}
}

// TODO: finish
func Test_registration_Delete(t *testing.T) {
	// Generate an identity for use
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity
	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1 * time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = false

	r, err := newRegistration(id, kv)
	if err != nil {
		t.Fatalf("Registration creation failed when it should have "+
			"succeeded: %+v", err)
	}

	err = r.Delete()
	if err != nil {
		t.Errorf("Delete() returned an error: %+v", err)
	}
}
