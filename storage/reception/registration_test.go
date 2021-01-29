package reception

import (
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

	_, err := newRegistration(id, kv)
	if err!=nil || !strings.Contains(err.Error(), "Cannot create a registration for an identity which has expired"){
		t.Errorf("Registeration creation failed when it should have " +
			"succeded: %+v", err)
	}
}