package utility

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmix/historical"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentRoundVersion = 0

// StoreRound stores the round using the key.
func StoreRound(kv *versioned.KV, round historical.Round, key string) error {
	now := netTime.Now()

	marshaled, err := proto.Marshal(round.Raw)

	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentCyclicVersion,
		Timestamp: now,
		Data:      marshaled,
	}

	return kv.Set(key, currentRoundVersion, &obj)
}

// LoadRound stores the round using the key.
func LoadRound(kv *versioned.KV, key string) (historical.Round, error) {
	vo, err := kv.Get(key, currentRoundVersion)
	if err != nil {
		return historical.Round{}, err
	}

	ri := &pb.RoundInfo{}
	err = proto.Unmarshal(vo.Data, ri)
	if err != nil {
		return historical.Round{}, err
	}

	return historical.MakeRound(ri), nil
}

func DeleteRound(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentCyclicVersion)
}
