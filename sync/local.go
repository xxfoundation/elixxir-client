package sync

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

type LocalStoreEkv struct {
	data versioned.KV
}

// todo: docstring
func (ls *LocalStoreEkv) Read(path string) ([]byte, error) {
	obj, err := ls.data.Get(path, 0)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// todo: docstring
func (ls *LocalStoreEkv) Write(path string, data []byte) error {
	return ls.data.Set(path, &versioned.Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}
