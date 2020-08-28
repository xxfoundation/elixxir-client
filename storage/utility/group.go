package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"time"
)

const currentGroupVersion = 0

func StoreGroup(kv *versioned.KV, grp *cyclic.Group, key string) error {
	now := time.Now()

	data, err := grp.GobEncode()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentGroupVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(key, &obj)
}

func LoadGroup(kv *versioned.KV, key string) (*cyclic.Group, error) {
	vo, err := kv.Get(key)
	if err != nil {
		return nil, err
	}

	grp := &cyclic.Group{}

	return grp, grp.GobDecode(vo.Data)
}
