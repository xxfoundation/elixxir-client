////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentRoundVersion = 0

// StoreRound stores the round using the key.
func StoreRound(kv versioned.KV, round Round, key string) error {
	now := netTime.Now()

	marshaled, err := proto.Marshal(round.Raw)

	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentRoundVersion,
		Timestamp: now,
		Data:      marshaled,
	}

	return kv.Set(key, &obj)
}

// LoadRound stores the round using the key.
func LoadRound(kv versioned.KV, key string) (Round, error) {
	vo, err := kv.Get(key, currentRoundVersion)
	if err != nil {
		return Round{}, err
	}

	ri := &pb.RoundInfo{}
	err = proto.Unmarshal(vo.Data, ri)
	if err != nil {
		return Round{}, err
	}

	return MakeRound(ri), nil
}

func DeleteRound(kv versioned.KV, key string) error {
	return kv.Delete(key, currentRoundVersion)
}
