package reception

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"reflect"
	"testing"
	"time"
)

func TestIdentityEncodeDecode(t *testing.T) {

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

func TestIdentityDelete(t *testing.T) {

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
