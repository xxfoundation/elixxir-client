package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

const currentNDFVersion = 0

func LoadNDF(kv *versioned.KV, key string) (*ndf.NetworkDefinition, error) {
	vo, err := kv.Get(key)
	if err != nil {
		return nil, err
	}

	ndf, _, err := ndf.DecodeNDF(string(vo.Data))
	if err != nil {
		return nil, err
	}

	return ndf, err
}

func SaveNDF(kv *versioned.KV, key string, ndf *ndf.NetworkDefinition) error {
	marshaled, err := ndf.Marshal()
	if err != nil {
		return err
	}

	now := time.Now()

	obj := versioned.Object{
		Version:   currentNDFVersion,
		Timestamp: now,
		Data:      marshaled,
	}

	return kv.Set(key, &obj)
}
