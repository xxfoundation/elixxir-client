////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partition

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
)

const currentMultiPartMessagePartVersion = 0

func loadPart(kv *utility.KV, partNum uint8) ([]byte, error) {
	key := makeMultiPartMessagePartKey(partNum)

	data, err := kv.Get(key, currentMultiPartMessageVersion)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func savePart(kv *utility.KV, partNum uint8, part []byte) error {
	key := makeMultiPartMessagePartKey(partNum)

	obj := &versioned.Object{
		Version:   currentMultiPartMessageVersion,
		Timestamp: netTime.Now(),
		Data:      part,
	}
	return kv.Set(key, obj.Marshal())
}

func deletePart(kv *utility.KV, partNum uint8) error {
	key := makeMultiPartMessagePartKey(partNum)
	return kv.Delete(key, currentMultiPartMessageVersion)
}

// makeMultiPartMessagePartKey makes the key for a part.
func makeMultiPartMessagePartKey(part uint8) string {
	return "part:" + strconv.FormatUint(uint64(part), 10)
}
