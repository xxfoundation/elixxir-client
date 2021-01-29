package reception

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestIdentity_EncodeDecode(t *testing.T) {

	kv := versioned.NewKV(make(ekv.Memstore))
	r := Identity{
		EphId:       ephemeral.Id{},
		Source:      &id.Permissioning,
		End:         time.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  time.Now().Round(0),
		EndValid:    time.Now().Round(0),
		RequestMask: 2*time.Hour,
		Ephemeral:   false,
	}
	err := r.store(kv)
	if err!=nil{
		t.Errorf("Failed to store: %s", err)
	}

	rLoad, err := loadIdentity(kv)
	if err!=nil{
		t.Errorf("Failed to load: %s", err)
	}

	if !reflect.DeepEqual(r, rLoad){
		t.Errorf("The two registrations are not the same\n saved:  %+v\n loaded: %+v", r, rLoad)
	}
}

func TestIdentity_Delete(t *testing.T) {

	kv := versioned.NewKV(make(ekv.Memstore))
	r := Identity{
		EphId:       ephemeral.Id{},
		Source:      &id.Permissioning,
		End:         time.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  time.Now().Round(0),
		EndValid:    time.Now().Round(0),
		RequestMask: 2 * time.Hour,
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
		t.Errorf("Load after delete succeded")
	}
}


func TestIdentity_String(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	timestamp  := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	received, _ := generateFakeIdentity(rng, 15, timestamp)

	expected := "-1763 U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID"

	s := received.String()
	if s != expected{
		t.Errorf("String did not return the correct value: " +
			"\n\t Expected: %s\n\t Received: %s", expected, s)
	}
}

func TestIdentity_CalculateKrSize(t *testing.T){
	deltas := []time.Duration{0, 2*time.Second, 2*time.Hour, 36*time.Hour,
		time.Duration(rand.Uint32())*time.Millisecond}
	for _, d := range deltas {
		expected := int(d.Seconds()+1)*maxRoundsPerSecond
		now := time.Now()
		id := Identity{
			StartValid:  now,
			EndValid:    now.Add(d),
		}

		krSize := id.calculateKrSize()
		if krSize != expected{
			t.Errorf("kr size not correct! expected: %v, recieved: %v",
				expected, krSize)
		}
	}
}