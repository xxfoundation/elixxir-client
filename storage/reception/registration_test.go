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
	//generate an identity for use
	rng := rand.New(rand.NewSource(42))

	timestamp  := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity

	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Time{}
	id.ExtraChecks = 0

	_, err := newRegistration(id, kv)
	if err==nil || !strings.Contains(err.Error(), "Cannot create a registration for an identity which has expired"){
		t.Errorf("Registeration creation succeded with expired identity")
	}
}

func TestNewRegistration_Ephemeral(t *testing.T) {
	//generate an identity for use
	rng := rand.New(rand.NewSource(42))

	timestamp  := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity

	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1*time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = true

	reg, err := newRegistration(id, kv)
	if err!=nil{
		t.Errorf("Registeration creation failed when it should have " +
			"succeded: %+v", err)
		t.FailNow()
	}

	if reg.knownRounds == nil{
		t.Errorf("Ephemenral identity does not have a known rounds")
	}

	if reg.knownRoundsStorage!=nil{
		t.Errorf("Ephemenral identity has a known rounds storage")
	}

	//check if the known rounds is stored, it should not be
	if _, err = utility.LoadKnownRounds(reg.kv,knownRoundsStorageKey,id.calculateKrSize()); err==nil{
		t.Errorf("Ephemeral identity stored the known rounds when it " +
			"shouldnt")
	}


	if _, err = reg.kv.Get(identityStorageKey); err==nil{
		t.Errorf("Ephemeral identity stored the idenity when it " +
			"shouldnt")
	}
}

func TestNewRegistration_Persistent(t *testing.T) {
	//generate an identity for use
	rng := rand.New(rand.NewSource(42))

	timestamp  := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity

	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1*time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = false

	reg, err := newRegistration(id, kv)
	if err!=nil{
		t.Errorf("Registeration creation failed when it should have " +
			"succeded: %+v", err)
		t.FailNow()
	}

	if reg.knownRounds == nil{
		t.Errorf("Persistent identity does not have a known rounds")
	}

	if reg.knownRoundsStorage==nil{
		t.Errorf("Persistent identity does not have a known rounds storage")
	}

	//check if the known rounds is stored, it should not be
	if _, err = utility.LoadKnownRounds(reg.kv,knownRoundsStorageKey,id.calculateKrSize()); err!=nil{
		t.Errorf("Persistent identity did not store known rounds when " +
			"it should: %+v", err)
	}

	if _, err = reg.kv.Get(identityStorageKey); err!=nil{
		t.Errorf("Persistent identity did not store the idenity when " +
			"it should")
	}
}

func TestLoadRegistration(t *testing.T) {
	//generate an identity for use
	rng := rand.New(rand.NewSource(42))

	timestamp  := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	idu, _ := generateFakeIdentity(rng, 15, timestamp)
	id := idu.Identity

	kv := versioned.NewKV(make(ekv.Memstore))

	id.End = time.Now().Add(1*time.Hour)
	id.ExtraChecks = 2
	id.Ephemeral = false

	_, err := newRegistration(id, kv)
	if err!=nil{
		t.Errorf("Registeration creation failed when it should have " +
			"succeded: %+v", err)
		t.FailNow()
	}

	reg, err := loadRegistration(idu.EphId, idu.Source, idu.StartValid, kv)
	if err!=nil{
		t.Errorf("Registeration loading failed: %+v", err)
		t.FailNow()
	}

	if reg.knownRounds != nil{
		t.Errorf("Loading has a seperated known rounds, it shouldnt")
	}

	if reg.knownRoundsStorage==nil{
		t.Errorf("Loading identity does not have a known rounds storage")
	}
}