package backup

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	jsonStorageVersion = 0
	jsonStorageKey     = "JsonStorage"
)

func storeJson(json string, kv *versioned.KV) error {
	obj := &versioned.Object{
		Version:   jsonStorageVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(json),
	}

	return kv.Set(jsonStorageKey, jsonStorageVersion, obj)
}

func loadJson(kv *versioned.KV) string {
	obj, err := kv.Get(passwordStorageKey, passwordStorageVersion)
	if err != nil {
		return ""
	}

	return string(obj.Data)
}
