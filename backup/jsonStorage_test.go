package backup

import (
	"testing"

	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

func Test_storeJson_loadJson(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	json := "{'data': {'one': 1}}"

	err := storeJson(json, kv)
	if err != nil {
		t.Errorf("Failed to store JSON: %+v", err)
	}

	loaded := loadJson(kv)
	if loaded != json {
		t.Errorf("Did not receive expected data from KV.\n\tExpected: %s, Received: %s\n", json, loaded)
	}
}
